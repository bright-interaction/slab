package server

import (
	"database/sql"
	"fmt"

	"github.com/bright-interaction/slab/internal/config"
	"github.com/bright-interaction/slab/internal/shield"
)

// buildShieldStore picks the Shield store backend based on the resolved
// config. ATOMICSITE_SHIELD_REDIS_URL set => RedisStore (multi-node ready,
// every cluster member shares state). Empty => SQLStore (single-node OSS
// default, no extra deps).
//
// Returns the constructed store, a human-readable backend kind (logged at
// boot so the operator can confirm which mode is active), and an error
// for transport-level failures during construction. Callers treat error
// as "shield disabled" and log it loudly.
func buildShieldStore(cfg *config.Config, db *sql.DB) (shield.Store, string, error) {
	if cfg != nil && cfg.ShieldRedisURL != "" {
		rs, err := shield.NewRedisStoreFromURL(cfg.ShieldRedisURL)
		if err != nil {
			return nil, "", fmt.Errorf("redis store: %w", err)
		}
		if cfg.ShieldRedisPrefix != "" {
			rs.Prefix = cfg.ShieldRedisPrefix
		}
		return rs, "redis", nil
	}
	return shield.NewSQLStore(db), "sqlite", nil
}
