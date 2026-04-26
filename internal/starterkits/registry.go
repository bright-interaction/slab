// Package starterkits provides a registry for site starter kits that can be
// applied during the seed handler to populate a freshly created site with
// industry-specific defaults (pages, blocks, components, copy, etc).
//
// Concrete kits register themselves at init() time via Default.Register.
package starterkits

import (
	"context"
	"sort"
	"sync"

	"github.com/bright-interaction/slab/internal/store"
)

// Kit is the interface every starter kit must implement.
type Kit interface {
	// ID is the stable identifier used by API callers (e.g. "b2b-services").
	ID() string
	// Name is the human-readable name shown in the wizard.
	Name() string
	// Description is a short blurb shown in the wizard.
	Description() string
	// TargetSiteTypes lists the site types this kit is recommended for
	// (e.g. ["b2b"]). Empty means any.
	TargetSiteTypes() []string
	// Apply runs the kit against an already-seeded site. It must be
	// transaction-safe: callers pass a queries handle bound to an open
	// tx (queries.WithTx(tx)) and roll back on error.
	Apply(ctx context.Context, q *store.Queries, siteID string) error
}

// HiddenKit is implemented by kits that should not appear in the default
// wizard list but stay registered for backwards compatibility (sites already
// created with their ID, tests, programmatic use). The wizard's
// /api/starter-kits endpoint filters these out unless include=hidden.
type HiddenKit interface {
	Hidden() bool
}

// IsHidden returns true if k implements HiddenKit and reports hidden.
func IsHidden(k Kit) bool {
	if hk, ok := k.(HiddenKit); ok {
		return hk.Hidden()
	}
	return false
}

// Registry holds the registered kits.
type Registry struct {
	mu   sync.RWMutex
	kits map[string]Kit
}

// Default is the package-level registry that init()-time kits register against.
var Default = NewRegistry()

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{kits: map[string]Kit{}}
}

// Register adds a kit to the registry. Later registrations with the same ID
// overwrite earlier ones, which is intentional for tests.
func (r *Registry) Register(k Kit) {
	if k == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.kits[k.ID()] = k
}

// Get returns the kit registered under id, if any.
func (r *Registry) Get(id string) (Kit, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	k, ok := r.kits[id]
	return k, ok
}

// List returns all registered kits sorted by ID.
func (r *Registry) List() []Kit {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Kit, 0, len(r.kits))
	for _, k := range r.kits {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}
