// Package webhook owns the outbound webhook subscription + delivery
// pipeline. Two pieces:
//
//   - Emitter: handler-side helper that creates webhook_deliveries
//     rows for every active subscription on the site that matches
//     the event filter. Synchronous DB write; never blocks on HTTP.
//
//   - DeliveryManager: background worker that ticks every
//     deliveryTickInterval, pulls pending+retrying rows due for
//     delivery, POSTs each with an HMAC-SHA256 signature, and writes
//     the terminal status (success / dropped) or schedules the next
//     retry with exponential backoff.
//
// HMAC signing: SHA-256 of the request body, hex-encoded, sent as
// `X-Atomicsite-Signature: sha256=<hex>`. The receiver computes the
// same digest with their copy of the secret and constant-time-
// compares. Standard pattern (Stripe, GitHub, Shopify all use this
// shape).
//
// Retry schedule: 5 attempts total. Backoff in seconds:
// 60, 300, 1800, 7200, 86400 (1 min, 5 min, 30 min, 2 h, 24 h).
// After attempt 5 the delivery is marked 'dropped'. The receiver's
// idempotency contract is to deduplicate on `X-Atomicsite-Delivery`
// (the delivery ID, stable across retries).

package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bright-interaction/slab/internal/store"
)

// Event types. Add to this list when wiring a new emit point. Keep
// names lowercase, dot-separated, hierarchical (resource.verb).
const (
	EventPageCreated   = "page.created"
	EventPageUpdated   = "page.updated"
	EventPageDeleted   = "page.deleted"
	EventItemCreated   = "collection_item.created"
	EventItemUpdated   = "collection_item.updated"
	EventItemDeleted   = "collection_item.deleted"
	EventBuildSucceeded = "build.succeeded"
	EventBuildFailed    = "build.failed"
	EventFormSubmitted  = "form.submitted"
)

// MaxAttempts caps the retry chain so a permanently-broken
// subscription doesn't fill the deliveries table forever.
const MaxAttempts = 5

// backoffSchedule[i] is the seconds-to-wait after attempt i (0-indexed).
// First entry covers the gap between create + first attempt; the
// worker uses backoffSchedule[attempts] when scheduling the next try.
var backoffSchedule = []time.Duration{
	0,                // attempt 0 -> first try is immediate
	60 * time.Second, // after 1st failure -> retry in 1 min
	5 * time.Minute,  // after 2nd
	30 * time.Minute, // after 3rd
	2 * time.Hour,    // after 4th
	24 * time.Hour,   // after 5th (still retried? no, MaxAttempts=5)
}

// deliveryTickInterval is how often the worker pulls due rows. 5s
// keeps perceived latency low for fast-failing receivers without
// hammering the DB.
const deliveryTickInterval = 5 * time.Second

// deliveryHTTPTimeout caps each POST. Receivers that don't ACK
// within this budget count as failure for retry purposes.
const deliveryHTTPTimeout = 15 * time.Second

// deliveriesPerTick caps the per-tick batch size. SQLite + a
// reasonable receiver fleet handles 50/tick = 600/min comfortably.
const deliveriesPerTick = 50

// Emitter persists webhook_delivery rows for every active
// subscription on a site that matches a given event. Wired into
// every content mutation handler that wants to emit.
type Emitter struct {
	queries *store.Queries
}

func NewEmitter(queries *store.Queries) *Emitter {
	return &Emitter{queries: queries}
}

// Emit creates a delivery row per matching active subscription. Best
// effort: a DB write failure is logged but does not propagate to the
// caller (the underlying content write has already succeeded; we don't
// want a webhook hiccup to roll back legitimate user data).
func (e *Emitter) Emit(ctx context.Context, siteID, eventType string, payload any) {
	if e == nil || e.queries == nil || siteID == "" || eventType == "" {
		return
	}
	subs, err := e.queries.ListActiveWebhookSubscriptions(ctx, siteID)
	if err != nil {
		slog.Warn("webhook emit: list subs failed", "site_id", siteID, "event", eventType, "err", err)
		return
	}
	if len(subs) == 0 {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("webhook emit: payload marshal failed", "event", eventType, "err", err)
		return
	}
	for _, sub := range subs {
		if !subscriptionMatches(sub.EventTypesJson, eventType) {
			continue
		}
		if err := e.queries.CreateWebhookDelivery(ctx, store.CreateWebhookDeliveryParams{
			ID:             newID(),
			SubscriptionID: sub.ID,
			SiteID:         siteID,
			EventType:      eventType,
			PayloadJson:    string(body),
		}); err != nil {
			slog.Warn("webhook emit: create delivery failed", "sub_id", sub.ID, "err", err)
		}
	}
}

// subscriptionMatches reports whether a subscription's event-type
// filter accepts the given event. Empty array = subscribe to every
// event.
func subscriptionMatches(eventTypesJSON, eventType string) bool {
	if eventTypesJSON == "" || eventTypesJSON == "[]" {
		return true
	}
	var types []string
	if err := json.Unmarshal([]byte(eventTypesJSON), &types); err != nil {
		// Malformed filter: fail-closed (don't deliver) so a
		// corrupt write doesn't spam every receiver.
		return false
	}
	for _, t := range types {
		if t == eventType {
			return true
		}
		if t == "*" {
			return true
		}
	}
	return false
}

// DeliveryManager drives the background worker. One process owns
// one DeliveryManager. Wired in cmd/server/main.go alongside the
// retention manager.
type DeliveryManager struct {
	queries *store.Queries
	client  *http.Client

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

func NewDeliveryManager(queries *store.Queries) *DeliveryManager {
	return &DeliveryManager{
		queries: queries,
		client: &http.Client{
			Timeout: deliveryHTTPTimeout,
		},
	}
}

// Start launches the tick loop. Cancel parent ctx to stop, or call
// Stop for bounded-wait shutdown.
func (m *DeliveryManager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})
	m.mu.Unlock()

	go func() {
		defer close(m.done)
		t := time.NewTicker(deliveryTickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.runOnce(ctx)
			}
		}
	}()
}

// Stop cancels the ticker and waits up to ctx's deadline for the
// loop to exit. Safe to call when Start was never invoked.
func (m *DeliveryManager) Stop(ctx context.Context) {
	m.mu.Lock()
	cancel := m.cancel
	done := m.done
	m.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// RunOnce processes one batch synchronously. Useful for tests.
func (m *DeliveryManager) RunOnce(ctx context.Context) {
	m.runOnce(ctx)
}

func (m *DeliveryManager) runOnce(ctx context.Context) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	rows, err := m.queries.ListPendingWebhookDeliveries(ctx, store.ListPendingWebhookDeliveriesParams{
		NextAttemptAt: now,
		Limit:         deliveriesPerTick,
	})
	if err != nil {
		slog.Warn("webhook worker: list pending failed", "err", err)
		return
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			return
		}
		m.deliver(ctx, row)
	}
}

func (m *DeliveryManager) deliver(ctx context.Context, row store.WebhookDelivery) {
	sub, err := m.queries.GetWebhookSubscription(ctx, row.SubscriptionID)
	if err != nil {
		// Subscription was deleted; mark the delivery dropped so it
		// doesn't keep getting picked up.
		_ = m.queries.MarkWebhookDeliveryDropped(ctx, store.MarkWebhookDeliveryDroppedParams{
			ID:                 row.ID,
			LastResponseStatus: 0,
			LastError:          "subscription not found",
		})
		return
	}
	body := []byte(row.PayloadJson)
	sig := signBody(sub.Secret, body)

	postCtx, cancel := context.WithTimeout(ctx, deliveryHTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(postCtx, http.MethodPost, sub.TargetUrl, bytes.NewReader(body))
	if err != nil {
		m.scheduleRetry(ctx, row, 0, "request build: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Atomicsite-Event", row.EventType)
	req.Header.Set("X-Atomicsite-Delivery", row.ID)
	req.Header.Set("X-Atomicsite-Signature", "sha256="+sig)
	req.Header.Set("X-Atomicsite-Attempt", strconv.FormatInt(row.Attempts+1, 10))

	resp, err := m.client.Do(req)
	if err != nil {
		m.scheduleRetry(ctx, row, 0, "transport: "+err.Error())
		return
	}
	// Drain + close so connection reuse works.
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = m.queries.MarkWebhookDeliverySucceeded(ctx, store.MarkWebhookDeliverySucceededParams{
			ID:                 row.ID,
			LastResponseStatus: int64(resp.StatusCode),
		})
		_ = m.queries.UpdateWebhookSubscriptionStatus(ctx, store.UpdateWebhookSubscriptionStatusParams{
			ID:              sub.ID,
			LastDeliveredAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
			LastError:       "",
		})
		return
	}
	m.scheduleRetry(ctx, row, resp.StatusCode, fmt.Sprintf("http %d", resp.StatusCode))
}

func (m *DeliveryManager) scheduleRetry(ctx context.Context, row store.WebhookDelivery, status int, errMsg string) {
	nextAttempts := row.Attempts + 1
	if nextAttempts >= MaxAttempts {
		_ = m.queries.MarkWebhookDeliveryDropped(ctx, store.MarkWebhookDeliveryDroppedParams{
			ID:                 row.ID,
			LastResponseStatus: int64(status),
			LastError:          errMsg,
		})
		_ = m.queries.UpdateWebhookSubscriptionStatus(ctx, store.UpdateWebhookSubscriptionStatusParams{
			ID:              row.SubscriptionID,
			LastDeliveredAt: row.CreatedAt,
			LastError:       errMsg,
		})
		return
	}
	wait := backoffSchedule[nextAttempts]
	next := time.Now().UTC().Add(wait).Format("2006-01-02 15:04:05")
	_ = m.queries.MarkWebhookDeliveryRetrying(ctx, store.MarkWebhookDeliveryRetryingParams{
		ID:                 row.ID,
		LastResponseStatus: int64(status),
		LastError:          errMsg,
		NextAttemptAt:      next,
	})
}

// signBody returns the lowercase hex of HMAC-SHA256(secret, body).
// Receivers should constant-time compare against this string.
func signBody(secret string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// SignBody is exported so test receivers (and the admin "verify
// signature" helper) can compute the canonical hex without
// duplicating the HMAC dance.
func SignBody(secret string, body []byte) string { return signBody(secret, body) }

// newID mints a 24-hex ID for delivery + subscription rows. We don't
// import the handlers package here, so we mint locally.
func newID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
