// Package handlers: audit log helper + admin read endpoint.
//
// The single AuditLog() helper is called by every destructive
// handler AFTER the underlying mutation has succeeded. We do not
// log "tried but failed" rows; the slog stream already carries
// those, and audit_log is for the GDPR / SOC2 compliance feed
// where "we deleted X" is the question being answered.
//
// On any DB write failure here we slog.Warn but never propagate the
// error — losing an audit row must never break the user-facing
// operation, and the WAL mode + same-DB guarantees mean the gap
// window is tiny.
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/store"
)

// AuditAction enumerates the handful of verbs we currently log.
// Keep the set small — a sprawling enum makes the feed unreadable.
const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionRevoke AuditAction = "revoke"
	AuditActionRotate AuditAction = "rotate"
	AuditActionInvite AuditAction = "invite"
	AuditActionAccept AuditAction = "accept"
)

// AuditAction is a typed string so call sites can't pass arbitrary
// verb strings.
type AuditAction string

// AuditLog writes a single row. siteID may be "" for non-site-scoped
// actions (workspace member invite, secret rotation). resourceID may
// be "" when the resource doesn't have a stable ID yet (failed
// create) but typically carries the row's primary key. changes is
// any JSON-serialisable struct; nil emits "{}".
func AuditLog(
	ctx context.Context,
	queries *store.Queries,
	r *http.Request,
	siteID string,
	action AuditAction,
	resourceType, resourceID string,
	changes any,
) {
	actorID := ""
	actorRole := ""
	if u := authmw.GetUser(r); u != nil {
		actorID = u.ID
		actorRole = u.Role
	}
	changesJSON := "{}"
	if changes != nil {
		if b, err := json.Marshal(changes); err == nil {
			changesJSON = string(b)
		}
	}
	if err := queries.WriteAuditLog(ctx, store.WriteAuditLogParams{
		ID:           newID(),
		ActorUserID:  actorID,
		ActorRole:    actorRole,
		ActorIp:      auditClientIP(r),
		SiteID:       siteID,
		Action:       string(action),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ChangesJson:  changesJSON,
	}); err != nil {
		slog.Warn("audit_log: write failed (non-fatal)",
			"action", action,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"err", err,
		)
	}
}

// auditClientIP returns the visitor's IP using the same priority
// chain as the analytics tracker (CF-Connecting-IP first, then
// X-Forwarded-For, then RemoteAddr) so audit and analytics never
// disagree on the same request.
func auditClientIP(r *http.Request) string {
	if v := r.Header.Get("CF-Connecting-IP"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		// First entry is the closest client.
		for i, c := range v {
			if c == ',' {
				return v[:i]
			}
		}
		return v
	}
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// AuditHandler exposes the read surface (admin only).
type AuditHandler struct {
	queries *store.Queries
}

func NewAuditHandler(queries *store.Queries) *AuditHandler {
	return &AuditHandler{queries: queries}
}

// SiteFeed returns recent audit rows for one site. Wired under the
// site-scoped router so site_members rows gate access; workspace
// admins see all sites via the global feed.
//
// GET /api/sites/{siteID}/audit-log?limit=200
func (h *AuditHandler) SiteFeed(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	limit := parseLimit(r, 200, 1000)
	rows, err := h.queries.ListAuditLogBySite(r.Context(), store.ListAuditLogBySiteParams{
		SiteID: siteID,
		Limit:  limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load audit log")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit_log": rows})
}

// GlobalFeed returns rows across every site. Workspace admin only,
// gated by RequireAdmin in the route group.
//
// GET /api/admin/audit-log?limit=500
func (h *AuditHandler) GlobalFeed(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 500, 5000)
	rows, err := h.queries.ListAuditLogGlobal(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load audit log")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit_log": rows})
}

// parseLimit reads ?limit= as an int clamped to [1, max] with a
// fallback default. Out-of-range values fall back to the default
// rather than 400 because audit feeds are non-critical and we
// prefer to render something.
func parseLimit(r *http.Request, def, max int64) int64 {
	q := r.URL.Query().Get("limit")
	if q == "" {
		return def
	}
	n, err := strconv.ParseInt(q, 10, 64)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
