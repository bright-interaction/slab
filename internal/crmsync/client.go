package crmsync

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SignatureHeader is the HTTP header BrightCRM expects on every signed
// payload. Mirrored on the receiver side as VerifyWebhookSignature(...,
// "X-Slab-Signature").
const SignatureHeader = "X-Slab-Signature"

// defaultTimeout caps how long a single Send call waits for BrightCRM. The
// CRM hop must never block the user-facing /t/consent response, so callers
// should also fire Send from a goroutine with their own timeout.
const defaultTimeout = 5 * time.Second

// Client posts visitor analytics events to BrightCRM.
//
// A zero-configuration Client (empty webhookURL or secret) is a valid no-op:
// Send returns nil immediately. This keeps dev environments running without
// a CRM endpoint.
type Client struct {
	webhookURL string
	mu         sync.RWMutex // guards secret so /admin/reload-secrets can swap
	secret     string
	httpClient *http.Client
}

// NewClient builds a Client. Pass an empty webhookURL or secret to disable
// outbound CRM sync (Send becomes a no-op).
func NewClient(webhookURL, secret string) *Client {
	return &Client{
		webhookURL: webhookURL,
		secret:     secret,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// UpdateSecret hot-swaps the signing secret. Called by /admin/reload-secrets
// when Dockyard rotates the shared HMAC value.
func (c *Client) UpdateSecret(s string) {
	c.mu.Lock()
	c.secret = s
	c.mu.Unlock()
}

// Enabled reports whether this Client will actually deliver events. Useful
// for tests and for short-circuiting upstream throttling logic.
func (c *Client) Enabled() bool {
	if c == nil || c.webhookURL == "" {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.secret != ""
}

// Send marshals event, signs it with HMAC-SHA256 (hex), and POSTs to the
// configured BrightCRM webhook URL. 200 and 204 responses are treated as
// success. Disabled clients return nil without a network call.
func (c *Client) Send(ctx context.Context, event Event) error {
	if !c.Enabled() {
		return nil
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("crmsync: marshal event: %w", err)
	}

	// Sign with the current secret only. Rotation is coordinated by Dockyard:
	// it pushes a new value into c.secret (via /admin/reload-secrets), the
	// receiving end keeps the old value in its PREVIOUS slot for the grace
	// window, in-flight requests still verify there. We never sign with the
	// previous value once rotated. RWMutex copy snapshots the value so a
	// concurrent UpdateSecret swap never tears.
	c.mu.RLock()
	currentSecret := c.secret
	c.mu.RUnlock()
	mac := hmac.New(sha256.New, []byte(currentSecret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("crmsync: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(SignatureHeader, sig)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("crmsync: post: %w", err)
	}
	defer resp.Body.Close()
	// Drain the body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("crmsync: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// SendAsync runs Send on a fresh background context with the package's
// default timeout. Errors are logged via slog and dropped, since callers
// usually don't have a meaningful way to surface them to end users.
//
// Use this from request handlers where the user-facing response must not
// block on BrightCRM round-trips.
func (c *Client) SendAsync(event Event) {
	if !c.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		if err := c.Send(ctx, event); err != nil {
			slog.Warn("crmsync: send failed", "event", event.Event, "site_id", event.SiteID, "err", err)
		}
	}()
}
