// Package handlers - webhook_hook.go.
//
// Package-level indirection so content-mutation handlers (pages,
// collection items, builds, forms) can emit webhook events without
// every handler's constructor signature growing a *webhook.Emitter
// field. Boot wires the singleton via SetWebhookEmitter; every
// emit call is a nil-check + delegation, zero allocations on the
// hot path when no subscriptions exist.

package handlers

import (
	"context"

	"github.com/brightinteraction/atomicsite/internal/webhook"
)

// activeEmitter is set once at boot. nil = webhook delivery is
// disabled (single-binary deploys without subscriptions configured).
// Reads happen on every content write but mutate is once at startup,
// so a plain pointer (not atomic) is fine - the value lives for the
// process lifetime.
var activeEmitter *webhook.Emitter

// SetWebhookEmitter wires the singleton. Called from cmd/server/main.go
// after the *webhook.Emitter is constructed.
func SetWebhookEmitter(e *webhook.Emitter) {
	activeEmitter = e
}

// emitWebhook is the convenience wrapper handlers use. nil-safe so
// the call site doesn't need to guard each invocation. Best-effort:
// a webhook delivery row that fails to land is logged inside the
// Emitter, never bubbles up to the caller.
func emitWebhook(ctx context.Context, siteID, eventType string, payload any) {
	if activeEmitter == nil {
		return
	}
	activeEmitter.Emit(ctx, siteID, eventType, payload)
}

// EmitWebhook is the public re-export so other packages (e.g. internal/mcp)
// can fire events through the same singleton without re-importing the
// webhook package and re-wiring an Emitter. Same nil-safe contract.
func EmitWebhook(ctx context.Context, siteID, eventType string, payload any) {
	emitWebhook(ctx, siteID, eventType, payload)
}
