//go:build !ee

package handlers

// WireMollie is the OSS-build no-op. Cloud-only routes already 503
// when checkoutImpl is nil, so this shim exists purely so
// cmd/server/main.go can call WireMollie(...) unconditionally without
// guarding the call site with build tags.
func WireMollie(h *BillingHandler, apiKey string) {
	// Intentionally empty. The OSS build never wires Mollie.
}
