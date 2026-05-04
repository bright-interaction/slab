package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

const AgentContextKey contextKey = "agent_key"

// AgentIdentity is the request-scoped agent after successful API key auth.
type AgentIdentity struct {
	KeyID        string
	SiteID       string
	Name         string
	Capabilities []string
}

// GetAgent retrieves the authenticated agent from the request context.
func GetAgent(r *http.Request) *AgentIdentity {
	a, _ := r.Context().Value(AgentContextKey).(*AgentIdentity)
	return a
}

// HashAgentKey produces a SHA-256 hash of the raw API key for storage.
func HashAgentKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

// AgentAuthMiddleware authenticates requests via X-Agent-Key header.
type AgentAuthMiddleware struct {
	queries *store.Queries
}

func NewAgentAuthMiddleware(queries *store.Queries) *AgentAuthMiddleware {
	return &AgentAuthMiddleware{queries: queries}
}

func (m *AgentAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agent, err := m.authenticate(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Update last_used_at in background
		go func() {
			_ = m.queries.UpdateAgentKeyLastUsed(context.Background(), agent.KeyID)
		}()

		ctx := context.WithValue(r.Context(), AgentContextKey, agent)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AgentAuthMiddleware) authenticate(r *http.Request) (*AgentIdentity, error) {
	raw := r.Header.Get("X-Agent-Key")
	if raw == "" {
		// Also accept Bearer token in Authorization header
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			raw = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if raw == "" {
		return nil, errAgentNoKey
	}

	keyHash := HashAgentKey(raw)
	// Revocation enforcement: GetAgentKeyByHash already filters
	// WHERE is_active = 1 at the SQL level, so a revoked key (set
	// is_active = 0 by RevokeAgentKey) returns sql.ErrNoRows here
	// and the request is rejected with errAgentInvalidKey. The test
	// in agent_auth_test.go pins this contract.
	row, err := m.queries.GetAgentKeyByHash(r.Context(), keyHash)
	if err != nil {
		return nil, errAgentInvalidKey
	}

	var caps []string
	if err := json.Unmarshal([]byte(row.Capabilities), &caps); err != nil {
		caps = []string{"read"}
	}

	return &AgentIdentity{
		KeyID:        row.ID,
		SiteID:       row.SiteID,
		Name:         row.Name,
		Capabilities: caps,
	}, nil
}

const (
	errAgentNoKey     authError = "API key required. Set X-Agent-Key header or Bearer token."
	errAgentInvalidKey authError = "Invalid or deactivated API key."
)
