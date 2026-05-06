// Package identify: Phase 31.3 (2026-05-06) -- shared visitor-identify
// flow used by both POST /t/identify (explicit SDK call from the
// rendered site) and FormHandler.Submit (auto-identify when a form
// submission carries an email-shaped value).
//
// One Recorder is constructed at server boot and shared by both
// handlers so identify semantics are one place: validate the email
// shape, update visit_sessions.email + identified_at, fire a
// crmsync "identified" event so BrightCRM gets the freshly-named
// contact along with their full UTM / page history.
package identify

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/brightinteraction/atomicsite/internal/crmsync"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// Recorder owns the queries + crmsync handles. nil-safe: a nil
// Recorder no-ops every call so unit tests don't need to plumb
// the dependency.
type Recorder struct {
	queries   *store.Queries
	crmClient *crmsync.Client
}

// NewRecorder builds a Recorder. Pass a disabled crmsync client
// (empty webhookURL/secret) to skip CRM dispatch -- the DB update
// still runs.
func NewRecorder(queries *store.Queries, crmClient *crmsync.Client) *Recorder {
	return &Recorder{queries: queries, crmClient: crmClient}
}

// Page captures URL context from the call site. Same shape as
// crmsync.Page so the forwarded event body matches the existing
// /t/inbound contract on the BrightCRM side.
type Page struct {
	URL      string
	Path     string
	Title    string
	Referrer string
}

// Outcome reports what happened. Callers use it for response shaping
// (200 vs 404 when no session exists) and for slog metadata.
type Outcome struct {
	Updated   bool
	VisitorID string
}

// ErrInvalidEmail is returned when the email argument is not a valid
// RFC 5322 address. The handler turns this into a 400.
var ErrInvalidEmail = errors.New("identify: invalid email")

// Identify validates the email shape, updates visit_sessions.email +
// identified_at if a session exists for (siteID, fingerprint), and
// fires a best-effort crmsync identified event so the CRM matches
// the freshly-named visitor to their behavioural history.
//
// The DB update is the source of truth; the CRM dispatch is async
// (SendAsync) so a slow webhook never blocks the caller. If no
// session row exists the call returns Outcome{Updated:false} with
// no error -- the caller decides whether to treat that as 404 or
// to upsert a fresh session first.
func (r *Recorder) Identify(ctx context.Context, siteID, fingerprint, email string, page Page) (Outcome, error) {
	if r == nil || r.queries == nil {
		return Outcome{}, nil
	}
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 320 {
		return Outcome{}, ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return Outcome{}, ErrInvalidEmail
	}

	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := r.queries.SetVisitSessionEmail(ctx, store.SetVisitSessionEmailParams{
		Email:        email,
		IdentifiedAt: now,
		LastSeenAt:   now,
		SiteID:       siteID,
		Fingerprint:  fingerprint,
	})
	if err != nil {
		slog.Warn("identify: set email failed",
			"site_id", siteID, "fingerprint", fingerprint, "err", err)
		return Outcome{}, err
	}
	if rows == 0 {
		return Outcome{Updated: false}, nil
	}

	// Pull visitor_id for the CRM payload.
	sess, err := r.queries.GetSessionByFingerprint(ctx, store.GetSessionByFingerprintParams{
		SiteID:      siteID,
		Fingerprint: fingerprint,
	})
	visitorID := ""
	if err == nil {
		visitorID = sess.VisitorID
	}

	r.dispatchCRM(siteID, visitorID, email, page)
	return Outcome{Updated: true, VisitorID: visitorID}, nil
}

// dispatchCRM is split out so unit tests can verify the call without
// standing up an HTTP server: a fake Client passed via NewRecorder
// records the SendAsync invocation.
func (r *Recorder) dispatchCRM(siteID, visitorID, email string, page Page) {
	if r.crmClient == nil || !r.crmClient.Enabled() {
		return
	}
	r.crmClient.SendAsync(crmsync.Event{
		Event:      crmsync.EventIdentified,
		SiteID:     siteID,
		VisitorID:  visitorID,
		Email:      email,
		OccurredAt: time.Now().UTC(),
		Page: crmsync.Page{
			URL:      page.URL,
			Path:     page.Path,
			Title:    page.Title,
			Referrer: page.Referrer,
		},
		Metadata: map[string]any{
			"source": "form_or_identify",
		},
	})
}

// emailFieldRE looks for plausibly-named email fields in form
// submissions. Used by FormHandler.Submit to pick the visitor's
// email out of the submitted JSON without requiring a fixed schema.
var emailFieldRE = regexp.MustCompile(`(?i)^(email|e_mail|e-mail|emailaddress|email_address|user_email|your[-_]email|contact[-_]email)$`)

// emailValueRE is a minimal email-shape fallback. We use it only as
// a tiebreaker when a field's name doesn't match emailFieldRE -- a
// last-resort scan of values in case the form labels its email
// field something exotic ("contact me at"). RFC 5322 has thousands
// of edge cases; we don't try to be exhaustive.
var emailValueRE = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// PickEmail scans form-field key/value pairs and returns the most
// likely email address. Preference order: keys matching emailFieldRE
// (the labeled email field) first, then any value that looks like
// an email by emailValueRE. Returns "" when nothing plausible is
// found.
func PickEmail(fields map[string]string) string {
	// Pass 1: labelled email field.
	for k, v := range fields {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if emailFieldRE.MatchString(k) && emailValueRE.MatchString(v) {
			return v
		}
	}
	// Pass 2: any value that looks like an email.
	for _, v := range fields {
		v = strings.TrimSpace(v)
		if emailValueRE.MatchString(v) {
			return v
		}
	}
	return ""
}
