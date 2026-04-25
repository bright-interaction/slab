// Package crmsync forwards Atomicsite visitor analytics events to BrightCRM
// via the /webhooks/incoming/atomicsite receiver. The package owns the wire
// format, HMAC signing, throttling, and the no-op fallback used when the
// CRM webhook URL/secret are unset (dev environments).
package crmsync

import "time"

// EventName is the discriminator the CRM uses to pick an activity_type.
// Keep these aligned with brightcrm/internal/handler/atomicsite_webhook.go's
// mapAtomicsiteActivityType.
type EventName string

const (
	EventVisit      EventName = "visit"
	EventPageView   EventName = "page_view"
	EventFormView   EventName = "form_view"
	EventDownload   EventName = "download"
	EventIdentified EventName = "identified"
)

// Event is the JSON payload Atomicsite POSTs to BrightCRM.
type Event struct {
	Event         EventName      `json:"event"`
	SiteID        string         `json:"site_id"`
	VisitorID     string         `json:"visitor_id"`
	Email         string         `json:"email,omitempty"`
	Name          string         `json:"name,omitempty"`
	CompanyDomain string         `json:"company_domain,omitempty"`
	Page          Page           `json:"page"`
	OccurredAt    time.Time      `json:"occurred_at"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// Page captures the URL-level context for a tracked event.
type Page struct {
	URL      string `json:"url"`
	Path     string `json:"path"`
	Title    string `json:"title"`
	Referrer string `json:"referrer"`
}
