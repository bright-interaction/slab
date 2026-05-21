package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	authmw "github.com/bright-interaction/slab/internal/middleware"
)

// registerRevisionTools wires list_revisions + get_revision +
// restore_revision. Closes the loop where the agent (or the human via
// the dashboard) can inspect prior states of a page/block and roll
// back without losing the current state (restore appends a new
// revision; history is append-only).
//
// All three tools route to the same *handlers.RevisionsHandler so the
// cross-tenant defence + entity-belongs-to-site check are written
// exactly once. Nil handler disables the tools with a clean error.
//
// Sprint 1 of the WP/Webflow replacement roadmap (see
// atomicsite/docs/ROADMAP.md).
func (s *Server) registerRevisionTools() {
	register := func(t Tool) { s.tools[t.Name] = t }

	register(Tool{
		Name:        "list_revisions",
		Description: "Lists prior revisions of an entity (page or block) for this site. Each revision is a snapshot written automatically before the entity was updated or deleted. Use to see what changed when, who made the change, and to pick a version to restore. Required: entity_type ('page' or 'block'), entity_id. Optional: limit (1-200, default 50). Returns revisions newest-first with version_number, change_summary, created_by ('user:{id}' or 'agent:{keyID}'), created_at, and snapshot_json. Use get_revision(entity_type, entity_id, version) to fetch a single snapshot, or restore_revision to roll back.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"entity_type":{"type":"string","enum":["page","block"]},
				"entity_id":{"type":"string"},
				"limit":{"type":"integer","minimum":1,"maximum":200}
			},
			"required":["entity_type","entity_id"]
		}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.revisions == nil {
				return "", errors.New("list_revisions not configured on this server (revisions handler unset)")
			}
			var args struct {
				EntityType string `json:"entity_type"`
				EntityID   string `json:"entity_id"`
				Limit      int    `json:"limit"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			args.EntityType = strings.ToLower(strings.TrimSpace(args.EntityType))
			if args.EntityID == "" {
				return "", errors.New("entity_id required")
			}
			rows, err := s.revisions.ListLatestForEntity(ctx, agent.SiteID, args.EntityType, args.EntityID, args.Limit)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{
				"revisions": rows,
				"count":     len(rows),
				"hint":      "Use restore_revision(entity_type, entity_id, version) to roll back. Restore is non-destructive: the current state is snapshotted first, so you can undo the restore itself.",
			}), nil
		},
	})

	register(Tool{
		Name:        "get_revision",
		Description: "Returns a single revision snapshot by version number for an entity (page or block). Use after list_revisions to inspect the exact prior state before deciding to restore. Returns snapshot_json (the full row at that version), change_summary, created_by, created_at. Required: entity_type ('page' or 'block'), entity_id, version (positive integer).",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"entity_type":{"type":"string","enum":["page","block"]},
				"entity_id":{"type":"string"},
				"version":{"type":"integer","minimum":1}
			},
			"required":["entity_type","entity_id","version"]
		}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.revisions == nil {
				return "", errors.New("get_revision not configured on this server (revisions handler unset)")
			}
			var args struct {
				EntityType string `json:"entity_type"`
				EntityID   string `json:"entity_id"`
				Version    int64  `json:"version"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			args.EntityType = strings.ToLower(strings.TrimSpace(args.EntityType))
			row, err := s.revisions.GetByVersion(ctx, agent.SiteID, args.EntityType, args.EntityID, args.Version)
			if err != nil {
				return "", err
			}
			return mustJSON(map[string]any{"revision": row}), nil
		},
	})

	register(Tool{
		Name:        "restore_revision",
		Description: "Restores an entity (page or block) to a prior version. NON-DESTRUCTIVE: the current state is snapshotted first so you can undo the restore itself. After this call the live entity has the data from the given version_number; a new revision row is appended representing the restored state. Use after list_revisions + get_revision to pick the right version. Required: entity_type ('page' or 'block'), entity_id, version (positive integer).",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"entity_type":{"type":"string","enum":["page","block"]},
				"entity_id":{"type":"string"},
				"version":{"type":"integer","minimum":1}
			},
			"required":["entity_type","entity_id","version"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			if s.revisions == nil {
				return "", errors.New("restore_revision not configured on this server (revisions handler unset)")
			}
			var args struct {
				EntityType string `json:"entity_type"`
				EntityID   string `json:"entity_id"`
				Version    int64  `json:"version"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			args.EntityType = strings.ToLower(strings.TrimSpace(args.EntityType))
			result, err := s.revisions.RestoreForAgent(ctx, agent.SiteID, args.EntityType, args.EntityID, args.Version, "agent:"+agent.KeyID)
			if err != nil {
				return "", err
			}
			return mustJSON(result), nil
		},
	})
}
