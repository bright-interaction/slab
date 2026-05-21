package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bright-interaction/slab/internal/handlers"
	authmw "github.com/bright-interaction/slab/internal/middleware"
)

// registerClarificationTools wires request_clarification +
// get_clarification + list_pending_clarifications. Closes the loop where
// the agent has to guess on ambiguous decisions (which hero variant,
// formal vs casual voice, Stockholm office in hero or footer) and ships
// the wrong answer. With these tools the agent opens a question, the
// human resolves via the dashboard inbox, the agent polls or waits
// inline, then keeps building with the answer.
//
// All three tools route to the same *handlers.ClarificationsHandler so
// the validation + audit semantics match the REST endpoints exactly.
// Nil handler disables the tools with a clean "not configured" error.
func (s *Server) registerClarificationTools() {
	register := func(t Tool) { s.tools[t.Name] = t }

	register(Tool{
		Name:        "request_clarification",
		Description: "Opens a clarification: a row in the clarifications table the human resolves via the dashboard inbox. Use when you reach a decision point that would otherwise force you to guess (which hero variant fits the brand, formal vs casual voice, whether to include the Stockholm office in hero vs footer, etc). Required: question (1-2000 chars). Optional: options (array of up to 12 pre-baked choices, each <=200 chars; human can pick an index OR write free-text), context (up to 8000 chars; surrounding info the human needs to make the call: page slug, block id, design corpus references, screenshot summary). Optional: wait_seconds (0-60; when set, the tool blocks and polls until resolved or wait_seconds elapses, returning the latest status either way; default 0 = async, returns immediately with status=pending so the agent can do other work and poll get_clarification later). Returns the full clarification record so the agent has the id for follow-up polling.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"question":{"type":"string"},
				"options":{"type":"array","items":{"type":"string"},"maxItems":12},
				"context":{"type":"string"},
				"wait_seconds":{"type":"integer","minimum":0,"maximum":60}
			},
			"required":["question"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.clarifications == nil {
				return "", errors.New("request_clarification not configured on this server (clarifications handler unset)")
			}
			var args struct {
				Question    string   `json:"question"`
				Options     []string `json:"options"`
				Context     string   `json:"context"`
				WaitSeconds int      `json:"wait_seconds"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			row, err := s.clarifications.CreateClarification(ctx, agent.SiteID, "agent:"+agent.KeyID, handlers.CreateClarificationParams{
				Question: args.Question,
				Options:  args.Options,
				Context:  args.Context,
			})
			if err != nil {
				return "", err
			}

			out, err := s.clarifications.GetClarification(ctx, agent.SiteID, row.ID)
			if err != nil {
				return "", err
			}

			// Optional inline wait: short-poll the row until status flips
			// or wait_seconds elapses. Bounded by 60s so the JSON-RPC call
			// can't hang the agent indefinitely.
			if args.WaitSeconds > 0 {
				deadline := time.Now().Add(time.Duration(args.WaitSeconds) * time.Second)
				for time.Now().Before(deadline) {
					select {
					case <-ctx.Done():
						return mustJSON(buildClarificationResult(out, "ctx cancelled")), nil
					case <-time.After(1 * time.Second):
					}
					latest, err := s.clarifications.GetClarification(ctx, agent.SiteID, row.ID)
					if err == nil {
						out = latest
						if latest.Status != "pending" {
							break
						}
					}
				}
			}

			return mustJSON(buildClarificationResult(out, "")), nil
		},
	})

	register(Tool{
		Name:        "get_clarification",
		Description: "Returns a single clarification by id. Use to poll for resolution after request_clarification(wait_seconds=0). Status flips from pending to resolved or cancelled when the human acts via the dashboard inbox. resolution_option is the index into options[] the human picked (-1 = free-text only). resolution_text is the human's elaboration.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"id":{"type":"string"}
			},
			"required":["id"]
		}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.clarifications == nil {
				return "", errors.New("get_clarification not configured on this server (clarifications handler unset)")
			}
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			id := strings.TrimSpace(args.ID)
			if id == "" {
				return "", errors.New("id required")
			}
			view, err := s.clarifications.GetClarification(ctx, agent.SiteID, id)
			if err != nil {
				return "", fmt.Errorf("clarification not found: %w", err)
			}
			return mustJSON(buildClarificationResult(view, "")), nil
		},
	})

	register(Tool{
		Name:        "list_pending_clarifications",
		Description: "Lists clarifications the agent has opened that are still pending. Useful on session start to see what asks are still outstanding before opening new ones. Optional: limit (1-200, default 50).",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"limit":{"type":"integer","minimum":1,"maximum":200}
			}
		}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.clarifications == nil {
				return "", errors.New("list_pending_clarifications not configured")
			}
			var args struct {
				Limit int `json:"limit"`
			}
			_ = json.Unmarshal(raw, &args)
			limit := args.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			rows, err := s.clarifications.ListPendingByAgent(ctx, agent.SiteID, "agent:"+agent.KeyID, limit)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{
				"clarifications": rows,
				"count":          len(rows),
				"hint":           "Status of each row is pending. Poll get_clarification(id) or call request_clarification again with wait_seconds to wait inline.",
			}), nil
		},
	})
}

// buildClarificationResult wraps the handler's view with MCP-friendly
// hints so the agent knows what to do next. Same shape for every
// clarification tool so polling code can branch on status uniformly.
func buildClarificationResult(view any, note string) map[string]any {
	out := map[string]any{
		"clarification": view,
		"hint":          "Status=pending: poll get_clarification(id) or call request_clarification with wait_seconds. Status=resolved: read resolution_option (index into options[]) AND resolution_text (human elaboration). Status=cancelled: the human closed the question without answering, pick a sensible default and continue.",
	}
	if note != "" {
		out["note"] = note
	}
	return out
}
