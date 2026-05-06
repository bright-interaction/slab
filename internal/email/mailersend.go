// Package email wraps transactional-email senders. MailerSend is the
// EU-sovereign default (servers in Frankfurt + Amsterdam, GDPR-friendly,
// generous free tier). Other providers can be added behind the same
// MailSender interface; the existing handlers/auth_reset.go already
// declares the interface so /forgot-password works against any backend.
//
// Phase 30.5, Cloud Tier MVP, 2026-05-06.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MailerSendClient implements the MailSender interface from
// internal/handlers/auth_reset.go (and the welcome / invite emails
// added in Phase 30.5).
type MailerSendClient struct {
	apiToken  string
	fromEmail string
	fromName  string
	httpc     *http.Client
	baseURL   string
}

// NewMailerSendClient returns a configured client. apiToken empty
// turns every send into a no-op (the existing forgot-password path
// degrades to "see logs" in that case, preserving local-dev UX).
func NewMailerSendClient(apiToken, fromEmail, fromName string) *MailerSendClient {
	return &MailerSendClient{
		apiToken:  strings.TrimSpace(apiToken),
		fromEmail: strings.TrimSpace(fromEmail),
		fromName:  strings.TrimSpace(fromName),
		httpc:     &http.Client{Timeout: 15 * time.Second},
		baseURL:   "https://api.mailersend.com/v1",
	}
}

// IsConfigured reports whether the client has the credentials it
// needs to send. Handlers should treat false as "skip; log instead".
func (c *MailerSendClient) IsConfigured() bool {
	return c.apiToken != "" && c.fromEmail != ""
}

// SendPasswordResetLink implements the MailSender interface used by
// auth_reset.go. The link goes to /reset-password/{token} on the
// admin host.
func (c *MailerSendClient) SendPasswordResetLink(ctx context.Context, email, link string) error {
	subject := "Reset your Atomic Site password"
	htmlBody := fmt.Sprintf(`<p>Hi,</p>
<p>Click below to reset your Atomic Site password. The link expires in 30 minutes.</p>
<p><a href=%q style="display:inline-block;padding:10px 20px;background:#0a0a0a;color:#fff;text-decoration:none;border-radius:8px">Reset password</a></p>
<p style="color:#666;font-size:12px">If the button doesn't work, paste this into your browser:<br/>%s</p>`, link, link)
	textBody := fmt.Sprintf("Hi,\n\nReset your Atomic Site password (link expires in 30 minutes):\n%s\n", link)
	return c.send(ctx, []recipient{{Email: email}}, subject, htmlBody, textBody)
}

// SendWelcome is the post-signup welcome email new cloud customers
// get when they redeem a waitlist invite. Phase 30.5.
func (c *MailerSendClient) SendWelcome(ctx context.Context, email, name, signinURL string) error {
	if !c.IsConfigured() {
		return nil
	}
	subject := "Welcome to Atomic Site"
	htmlBody := fmt.Sprintf(`<p>Hi %s,</p>
<p>Welcome to Atomic Site. Your workspace is ready.</p>
<p><a href=%q style="display:inline-block;padding:10px 20px;background:#0a0a0a;color:#fff;text-decoration:none;border-radius:8px">Open your workspace</a></p>
<p>You're on the 14-day Solo trial. Upgrade or cancel any time from Account, Billing.</p>
<p>Reply to this email if anything's unclear; a human reads every message.</p>
<p>, Tom</p>`, escapeHTMLBasic(name), signinURL)
	textBody := fmt.Sprintf("Hi %s,\n\nWelcome to Atomic Site. Open your workspace: %s\n\nReply if anything's unclear.\n\n- Tom\n", name, signinURL)
	return c.send(ctx, []recipient{{Email: email, Name: name}}, subject, htmlBody, textBody)
}

// SendWaitlistInvite is the email recipients get when an admin
// approves their waitlist entry and mints a signup token.
func (c *MailerSendClient) SendWaitlistInvite(ctx context.Context, email, name, signupURL string) error {
	if !c.IsConfigured() {
		return nil
	}
	subject := "Your Atomic Site invite is ready"
	htmlBody := fmt.Sprintf(`<p>Hi %s,</p>
<p>Your Atomic Site beta invite is ready. Click below to set a password and create your workspace. The link expires in 7 days.</p>
<p><a href=%q style="display:inline-block;padding:10px 20px;background:#0a0a0a;color:#fff;text-decoration:none;border-radius:8px">Activate your account</a></p>
<p>Atomic Site is EU-sovereign by design. Your data, your visitors, your DPA, all stay inside the EU. We use Hetzner for origin, Mollie for payments, MailerSend for transactional email; all are EU-incorporated.</p>
<p>Reply to this email if anything's unclear.</p>
<p>, Tom</p>`, escapeHTMLBasic(name), signupURL)
	textBody := fmt.Sprintf("Hi %s,\n\nYour Atomic Site beta invite is ready (expires in 7 days):\n%s\n\nReply if anything's unclear.\n\n- Tom\n", name, signupURL)
	return c.send(ctx, []recipient{{Email: email, Name: name}}, subject, htmlBody, textBody)
}

// recipient is one to/from row in the MailerSend API.
type recipient struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// send is the inner HTTP driver. POST /v1/email with bearer token.
// 202 Accepted is the success status; anything else is an error.
func (c *MailerSendClient) send(ctx context.Context, to []recipient, subject, htmlBody, textBody string) error {
	if !c.IsConfigured() {
		return errors.New("mailersend: not configured")
	}
	from := recipient{Email: c.fromEmail, Name: c.fromName}
	body := map[string]any{
		"from":    from,
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
		"text":    textBody,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("mailersend: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/email", bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("mailersend: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("mailersend: do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("mailersend: HTTP %d: %s", resp.StatusCode, string(respBody))
}

// escapeHTMLBasic is a tiny helper for the personalised name in HTML
// bodies. We don't pull in html/template just for this; the only
// untrusted input is the name field. No fancy encoding, just the
// 5 metacharacters that matter for an HTML attribute / text node.
func escapeHTMLBasic(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}
