package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/store"
)

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// validateEmail returns nil for empty (optional fields) or a valid RFC 5322-ish email.
// It also blocks newlines to prevent injection into generated text files.
func validateEmail(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// Block ASCII control chars and unicode line separators
	if strings.ContainsAny(s, "\n\r\t") {
		return fmt.Errorf("email contains whitespace control characters")
	}
	for _, r := range s {
		if r == '\u2028' || r == '\u2029' || r == '\u0085' {
			return fmt.Errorf("email contains unicode line separators")
		}
	}
	if len(s) > 254 {
		return fmt.Errorf("email too long")
	}
	// RFC 5322 forbids consecutive dots anywhere in the address
	if strings.Contains(s, "..") {
		return fmt.Errorf("email contains consecutive dots")
	}
	if !emailPattern.MatchString(s) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// sanitizeLine removes CR/LF/tab and unicode line separators from a value
// intended for a single-line text file (security.txt, humans.txt, etc.).
func sanitizeLine(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\u2028", " ")
	s = strings.ReplaceAll(s, "\u2029", " ")
	s = strings.ReplaceAll(s, "\u0085", " ")
	return strings.TrimSpace(s)
}

// ProfileHandler handles business profile CRUD.
type ProfileHandler struct {
	cfg     *config.Config
	queries *store.Queries
}

func NewProfileHandler(cfg *config.Config, queries *store.Queries) *ProfileHandler {
	return &ProfileHandler{cfg: cfg, queries: queries}
}

// Get returns the site profile. Returns an empty profile shape if none exists.
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")
	p, err := h.queries.GetSiteProfile(r.Context(), siteID)
	if err != nil {
		// Empty profile is fine
		writeJSON(w, http.StatusOK, map[string]any{
			"site_id": siteID,
		})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Upsert creates or updates the site profile.
func (h *ProfileHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	siteID := urlParam(r, "siteID")

	var req struct {
		BusinessName   string `json:"business_name"`
		RegistrationNr string `json:"registration_nr"`
		Country        string `json:"country"`
		ContactEmail   string `json:"contact_email"`
		ContactPhone   string `json:"contact_phone"`
		PrivacyEmail   string `json:"privacy_email"`
		SecurityEmail  string `json:"security_email"`
		AddressLine1   string `json:"address_line1"`
		AddressLine2   string `json:"address_line2"`
		City           string `json:"city"`
		PostalCode     string `json:"postal_code"`
	}
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	// Validate emails (rejects control chars + enforces format)
	for fieldName, val := range map[string]string{
		"contact_email":  req.ContactEmail,
		"privacy_email":  req.PrivacyEmail,
		"security_email": req.SecurityEmail,
	} {
		if err := validateEmail(val); err != nil {
			writeError(w, http.StatusBadRequest, fieldName+": "+err.Error())
			return
		}
	}

	err := h.queries.UpsertSiteProfile(r.Context(), store.UpsertSiteProfileParams{
		ID:             newID(),
		SiteID:         siteID,
		BusinessName:   sanitizeLine(req.BusinessName),
		RegistrationNr: sanitizeLine(req.RegistrationNr),
		Country:        sanitizeLine(req.Country),
		ContactEmail:   strings.TrimSpace(req.ContactEmail),
		ContactPhone:   sanitizeLine(req.ContactPhone),
		PrivacyEmail:   strings.TrimSpace(req.PrivacyEmail),
		SecurityEmail:  strings.TrimSpace(req.SecurityEmail),
		AddressLine1:   sanitizeLine(req.AddressLine1),
		AddressLine2:   sanitizeLine(req.AddressLine2),
		City:           sanitizeLine(req.City),
		PostalCode:     sanitizeLine(req.PostalCode),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save profile")
		return
	}

	p, _ := h.queries.GetSiteProfile(r.Context(), siteID)
	writeJSON(w, http.StatusOK, p)
}
