// Package retention - billing_shim.go.
//
// init() wires the package-level `billingLimit` indirection in
// retention.go to the real `billing.Limit` lookup. The shim exists
// so retention.go itself doesn't have to import internal/billing
// directly (keeps the file's import set small and stable; the
// indirection costs one func call per workspace per sweep, well
// inside the noise floor).

package retention

import "github.com/brightinteraction/atomicsite/internal/billing"

func init() {
	billingLimit = billing.Limit
}
