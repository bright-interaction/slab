// Package ee defines the build-tag boundary between Atomic Site OSS Core and
// the Cloud (Enterprise Edition) distribution. See ee/README.md for the
// canonical list of what belongs in this package.
//
// The Provider interface is the single seam between OSS Core and Cloud. Core
// constructs a Provider on startup and stores it on the dependency struct;
// every cloud-shaped operation (multi-tenant edge orchestration, billing,
// cross-tenant rollups) routes through it.
//
// The OSS build (default, no `-tags ee`) links cloud_oss.go and gets the
// no-op Provider that returns ErrNotAvailable for every operation. The
// Cloud build (`-tags ee`) links cloud_ee.go and gets the real Provider.
package ee

import "errors"

// ErrNotAvailable is returned by every Provider method in the OSS build.
// Callers in core check `errors.Is(err, ee.ErrNotAvailable)` to decide
// between "fail loudly because cloud was expected" and "fall through to
// the OSS path because cloud is optional".
var ErrNotAvailable = errors.New("atomicsite: cloud feature not available in OSS build")

// Provider is the cloud control-plane seam. The OSS build returns a stub
// that no-ops every method. The Cloud build wires real implementations
// (Vercel API client, Caddy admin client, Stripe, etc.).
//
// Methods are added here as cloud features land. Today the interface is
// intentionally minimal: the build-tag boundary itself is the deliverable,
// not any specific cloud feature.
type Provider interface {
	// Mode reports the build flavour. Returns "oss" or "cloud". Used by
	// internal/config Validate() to refuse-to-start when
	// ATOMICSITE_DEPLOYMENT_MODE=cloud is set on a binary that was not
	// built with -tags ee.
	Mode() string
}
