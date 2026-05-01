package perfectfoundation

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/bright-interaction/slab/internal/store"
)

// Apply seeds the perfect-foundation defaults on a new site: reference KB
// (curated reference-site guidance) and any extra built-in guardrails
// not covered by the in-engine ValidateBlock / ValidatePageSlug paths.
//
// Callers pass a tx-bound *store.Queries (queries.WithTx(tx)) so a failure
// rolls back everything. The starter-kit Apply() runs after this so kits
// can extend or override foundation entries.
func Apply(ctx context.Context, q *store.Queries, siteID string) error {
	if err := SeedReferenceKnowledgebase(ctx, q, siteID); err != nil {
		return err
	}
	return nil
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
