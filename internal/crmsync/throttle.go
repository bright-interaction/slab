package crmsync

import (
	"sync"
	"time"
)

// Throttler suppresses repeated visitor events that fire within a short
// window. The goal is to keep BrightCRM's activity log readable: a single
// visitor scrolling through ten pages in a minute should generate one
// activity row, not ten.
//
// Identified events bypass the throttle entirely so we never lose the
// moment a visitor links to a contact.
type Throttler struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time
	interval time.Duration
	clock    func() time.Time
}

// NewThrottler builds a Throttler. interval is the minimum gap between two
// non-identified events for the same (site_id, visitor_id). Pass 0 to
// disable throttling (every event passes).
func NewThrottler(interval time.Duration) *Throttler {
	return &Throttler{
		lastSeen: make(map[string]time.Time),
		interval: interval,
		clock:    time.Now,
	}
}

// Allow reports whether event should be forwarded to BrightCRM. Side
// effect: when Allow returns true and the event is throttle-eligible, the
// last-seen timestamp for (site, visitor) is updated.
func (t *Throttler) Allow(event Event) bool {
	if t == nil || t.interval <= 0 {
		return true
	}
	if event.Event == EventIdentified {
		// Identified events always pass and reset the window so an
		// identification doesn't keep a fresh post-identification visit out.
		t.mu.Lock()
		defer t.mu.Unlock()
		t.lastSeen[throttleKey(event)] = t.clock()
		return true
	}
	if event.VisitorID == "" {
		// No visitor key to throttle on; let it through and let the CRM
		// decide whether to drop it.
		return true
	}

	key := throttleKey(event)
	now := t.clock()

	t.mu.Lock()
	defer t.mu.Unlock()
	last, ok := t.lastSeen[key]
	if ok && now.Sub(last) < t.interval {
		return false
	}
	t.lastSeen[key] = now
	return true
}

// throttleKey is the (site, visitor) tuple used to dedupe events.
func throttleKey(e Event) string {
	return e.SiteID + "|" + e.VisitorID
}
