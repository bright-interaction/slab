// Package handlers: site_domains CRUD + verification + reconcile.
//
// site_domains drives the "type a custom hostname into the admin and have
// it answer with TLS, security headers, and the right tenant's content"
// pipeline. The flow:
//
//  1. Admin POSTs the hostname. We mint a verify_token and persist the
//     row at status='pending'.
//  2. Admin points DNS at our edge IP. The /.well-known/atomic-verify/
//     handler below answers with the token; the verification poller
//     fetches that URL via plain HTTP and flips the row to 'verified'
//     when the token matches.
//  3. Reconciler issues a Let's Encrypt cert via certbot HTTP-01,
//     writes a per-domain nginx server block with the A+ headers
//     snippet + per-site CSP override, and reloads nginx.
//  4. Steady-state polling probes the live HTTPS endpoint and keeps
//     the row at 'live' (or flips back to 'error' with last_error
//     surfaced in the admin).
//
// This file owns 1, 2, and the public verify endpoint. Steps 3 and 4
// live in internal/domains (separate package) so the reconcile loop
// doesn't import handlers.
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"

	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// hostnamePattern matches RFC 1123 hostnames for the admin "Add domain"
// flow. We deliberately exclude ports, schemes, and trailing dots; the
// admin types a bare hostname.
var hostnamePattern = regexp.MustCompile(`^([a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// reservedHostnameSuffixes are hostnames the operator's own infra owns;
// adding them as a tenant custom domain would wreck the wildcard server
// block. The list is intentionally small: the deployment-suffix env var
// is checked dynamically in the handler, not hardcoded here.
var reservedHostnameSuffixes = []string{
	".localhost",
	".local",
}

// DomainHandler manages /api/sites/{siteID}/domains routes.
type DomainHandler struct {
	queries *store.Queries
	// reconcile is fired-and-forgotten when a domain row changes state
	// so the background reconciler can pick it up immediately. nil-safe;
	// when no reconciler is wired the polling loop will still catch up
	// on its own cadence.
	reconcile func()
	// reservedSuffix is the wildcard tenant suffix the deployment
	// already routes (e.g. ".atomicsite.example.com").
	// Custom hostnames ending in this suffix would create a vhost that
	// shadows the wildcard server block, so we reject up front.
	// Empty disables the check (single-tenant deployments).
	reservedSuffix string
}

// NewDomainHandler builds the handler. reconcile may be nil during boot
// before the reconcile manager is wired; nil-safe. reservedSuffix is
// the wildcard tenant suffix that custom hostnames must NOT end with.
func NewDomainHandler(queries *store.Queries, reconcile func(), reservedSuffix string) *DomainHandler {
	return &DomainHandler{queries: queries, reconcile: reconcile, reservedSuffix: strings.ToLower(strings.TrimSpace(reservedSuffix))}
}

func (h *DomainHandler) signal() {
	if h.reconcile != nil {
		h.reconcile()
	}
}

// generateVerifyToken returns a 24-byte hex token (192 bits of entropy)
// suitable for /.well-known/atomic-verify lookups.
func generateVerifyToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// validateHostname rejects values that would break nginx routing or
// collide with reserved suffixes. Returns a normalised lowercase form.
func validateHostname(s string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(s))
	v = strings.TrimSuffix(v, ".")
	if v == "" {
		return "", errors.New("hostname is required")
	}
	if len(v) > 253 {
		return "", errors.New("hostname too long (max 253 chars)")
	}
	if strings.ContainsAny(v, " \t\r\n/?#") {
		return "", errors.New("hostname contains illegal characters")
	}
	if !hostnamePattern.MatchString(v) {
		return "", errors.New("invalid hostname format (expected example.com or sub.example.com)")
	}
	for _, suf := range reservedHostnameSuffixes {
		if strings.HasSuffix(v, suf) || v == strings.TrimPrefix(suf, ".") {
			return "", errors.New("hostname suffix is reserved")
		}
	}
	return v, nil
}

// List returns every domain row for the site.
//
// GET /api/sites/{siteID}/domains
func (h *DomainHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	rows, err := h.queries.ListDomainsBySite(r.Context(), siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list domains")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"domains": rows})
}

// Create attaches a hostname to the site. Returns the row plus the
// verify_token + the well-known URL the admin can curl to confirm
// DNS is wired correctly.
//
// POST /api/sites/{siteID}/domains
func (h *DomainHandler) Create(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	if _, err := h.queries.GetSiteByID(r.Context(), siteID); err != nil {
		writeError(w, http.StatusNotFound, "Site not found")
		return
	}
	var req struct {
		Hostname    string `json:"hostname"`
		IsCanonical bool   `json:"is_canonical"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	host, err := validateHostname(req.Hostname)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Reject hostnames inside the wildcard tenant suffix. Routing for
	// those is already owned by the existing nginx wildcard + Caddy
	// {labels.3} rule, and a per-domain vhost would shadow it.
	if h.reservedSuffix != "" {
		if host == strings.TrimPrefix(h.reservedSuffix, ".") || strings.HasSuffix(host, h.reservedSuffix) {
			writeError(w, http.StatusBadRequest, "This hostname is reserved by the platform's wildcard subdomain. Use a different domain or update the site slug instead.")
			return
		}
	}
	if existing, err := h.queries.GetDomainByHostname(r.Context(), host); err == nil {
		if existing.SiteID == siteID {
			writeError(w, http.StatusConflict, "This hostname is already attached to this site")
			return
		}
		writeError(w, http.StatusConflict, "This hostname is already in use")
		return
	}
	token, err := generateVerifyToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate verify token")
		return
	}
	id := newID()
	canonical := int64(0)
	if req.IsCanonical {
		canonical = 1
	}
	if err := h.queries.CreateDomain(r.Context(), store.CreateDomainParams{
		ID:          id,
		SiteID:      siteID,
		Hostname:    host,
		VerifyToken: token,
		IsCanonical: canonical,
	}); err != nil {
		// UNIQUE race: a concurrent request landed the same hostname
		// between the GetDomainByHostname check above and this insert.
		// Surface as 409 (the post-check would have caught it had it
		// run after the racing commit).
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "This hostname is already in use")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create domain row")
		return
	}
	if req.IsCanonical {
		_ = h.queries.SetDomainCanonical(r.Context(), store.SetDomainCanonicalParams{
			ID:     id,
			SiteID: siteID,
		})
	}
	row, _ := h.queries.GetDomainByID(r.Context(), id)
	h.signal()
	writeJSON(w, http.StatusCreated, row)
}

// SetCanonical flips the chosen domain to canonical for its site. The
// build pipeline reads the canonical row when site.domain is empty.
//
// POST /api/sites/{siteID}/domains/{domainID}/canonical
func (h *DomainHandler) SetCanonical(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "domainID")
	row, err := h.queries.GetDomainByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Domain not found")
		return
	}
	if err := h.queries.SetDomainCanonical(r.Context(), store.SetDomainCanonicalParams{
		ID:     id,
		SiteID: siteID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to set canonical")
		return
	}
	row, _ = h.queries.GetDomainByID(r.Context(), id)
	writeJSON(w, http.StatusOK, row)
}

// Delete removes a domain. Owner-only because removing the canonical
// hostname could take a customer-facing site offline. The reconciler
// observes the deletion via its periodic sweep (or via the signal()
// hook) and tears the corresponding nginx server block + cert down.
//
// DELETE /api/sites/{siteID}/domains/{domainID}
func (h *DomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if !authmw.RequireOwnerOrAdmin(w, r, h.queries) {
		return
	}
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "domainID")
	row, err := h.queries.GetDomainByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Domain not found")
		return
	}
	if err := h.queries.DeleteDomain(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete domain")
		return
	}
	h.signal()
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Refresh re-runs the verification + cert reconciliation for one
// domain right now (instead of waiting for the periodic poller).
// Useful when the admin just updated DNS and wants immediate feedback.
//
// POST /api/sites/{siteID}/domains/{domainID}/refresh
func (h *DomainHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	id := urlParam(r, "domainID")
	row, err := h.queries.GetDomainByID(r.Context(), id)
	if err != nil || row.SiteID != siteID {
		writeError(w, http.StatusNotFound, "Domain not found")
		return
	}
	h.signal()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}
