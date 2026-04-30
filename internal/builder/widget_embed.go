package builder

import (
	"crypto/sha256"
	"encoding/hex"
	_ "embed"
)

// CookieProofWidget is the inline-config widget bundle, sourced from the
// CookieProof repo (`automations/CookieProof/`) and built into a same-origin
// distribution by `bun run build` (rollup target `cookieproof.embed.esm.js`).
//
// The bundle is auto-registering .  once the script runs, `<cookie-consent>`
// is defined and the embed entry reads `window.__CCB__` to configure it.
// No runtime fetch to consent.example.com.
//
// Updated in lockstep by the atomicsite Dockerfile's "widget" stage so a
// production build always carries the freshest widget code.
//
//go:embed assets/cookieproof.embed.esm.js
var CookieProofWidget []byte

// CookieProofWidgetHash returns the first 8 hex chars of SHA-256 over the
// embedded widget bytes. Used as a cache-busting filename suffix when the
// builder writes the asset into a tenant workspace's public/ directory.
func CookieProofWidgetHash() string {
	sum := sha256.Sum256(CookieProofWidget)
	return hex.EncodeToString(sum[:])[:8]
}

// CookieProofWidgetFilename returns the filename used for the per-tenant
// widget asset. Same hash for every tenant on a given build, but the asset
// is written into each workspace separately so each tenant serves it
// same-origin.
func CookieProofWidgetFilename() string {
	return "_ccb." + CookieProofWidgetHash() + ".js"
}

// CircuitBgScript is the animated circuit-board background JS, embedded so
// any block with data-bg="circuit" can opt into a ~3.5KB animated PCB
// pattern in the site's primary color. Reads --color-primary from CSS so
// it auto-tints. Inline-rendered next to the canvas element by the
// renderer; no separate asset URL.
//
//go:embed assets/circuit-bg.js
var CircuitBgScript []byte

