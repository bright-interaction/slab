package domains

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// VerifyHostnameOwnership probes http://<hostname>/.well-known/atomic-verify/<token>
// and reports nil when the response body equals the expected token.
//
// This is the "your DNS points at us" check. We deliberately use plain
// HTTP (not HTTPS): the cert hasn't been issued yet, and the verifier
// inside the same nginx that will eventually serve the live site is
// proof enough that the routing is wired correctly. The verifier
// listens on the global atomicsite-acme.conf block (port 80), reading
// from the atomicsite app's /.well-known/atomic-verify/{token} handler.
//
// Timeout, redirects, and IP family are constrained so a malicious
// hostname can't make us hold connections forever or follow a 302 to
// localhost.
func VerifyHostnameOwnership(ctx context.Context, hostname, expectedToken string) error {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return fmt.Errorf("hostname is empty")
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second, Control: ssrfDialControl}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       10 * time.Second,
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		// Refuse redirects: a verified host should answer the token
		// path directly. Anything else is a misconfig (or worse, an
		// attempt to forward the probe to a different host where the
		// attacker controls the response).
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	url := fmt.Sprintf("http://%s/.well-known/atomic-verify/%s", hostname, expectedToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build verify request: %w", err)
	}
	req.Header.Set("User-Agent", "atomicsite-domain-verify/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify URL returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return fmt.Errorf("read verify body: %w", err)
	}
	got := strings.TrimSpace(string(body))
	if got != expectedToken {
		return fmt.Errorf("verify body mismatch: expected %q, got %q", expectedToken, truncate(got, 40))
	}
	return nil
}

// VerifyHostnameLive checks that the live HTTPS endpoint answers and
// returns 200/301/302/308 from any path. The reconciler uses this to
// flip a 'cert_ready' row to 'live' (and to detect when a 'live' row
// goes 'error' because the cert expired or DNS changed).
func VerifyHostnameLive(ctx context.Context, hostname string) error {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return fmt.Errorf("hostname is empty")
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second, Control: ssrfDialControl}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ResponseHeaderTimeout: 5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       10 * time.Second,
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	url := "https://" + hostname + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "atomicsite-domain-live/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}
	return fmt.Errorf("live probe returned %d", resp.StatusCode)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
