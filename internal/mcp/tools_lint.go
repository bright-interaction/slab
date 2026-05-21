package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/agent"
	"github.com/brightinteraction/atomicsite/internal/critique"
	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
)

// runBlockLint is the in-process bridge create_block + update_block
// use to run the synchronous design lint (gap 4 of 6, 2026-05-21).
// Takes whatever shape json.Unmarshal handed back (map[string]any |
// nil | scalar) and normalises to the map the linter expects.
//
// Why this lives in the MCP package: keeps the cross-package
// boundary tight (critique.LintBlockData only deals in map[string]any
// + DesignPlaybookInfo, no MCP framework types) and lets future
// linters fan out from a single hook.
func runBlockLint(blockType string, raw any) []critique.LintFinding {
	if raw == nil {
		return nil
	}
	data, ok := raw.(map[string]any)
	if !ok {
		// data was authored as a non-object (string, array). Skip lint;
		// the renderer will surface the type error on build.
		return nil
	}
	return critique.LintBlockData(blockType, data, agent.DefaultDesignPlaybook())
}

// runBlockLintFromJSON unmarshals a stored DataJson string and lints
// the result. Used by update_block where the post-mutation state comes
// from the DB as a JSON string rather than the just-decoded map.
func runBlockLintFromJSON(blockType, dataJSON string) []critique.LintFinding {
	dataJSON = strings.TrimSpace(dataJSON)
	if dataJSON == "" || dataJSON == "{}" || dataJSON == "null" {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return nil
	}
	return critique.LintBlockData(blockType, data, agent.DefaultDesignPlaybook())
}

// registerLintTools wires the standalone lint_block tool. Lets the
// agent run the same lint create_block embeds without actually
// committing the block: useful for the "try a few variants, pick the
// cleanest one" workflow before any DB write.
func (s *Server) registerLintTools() {
	register := func(t Tool) { s.tools[t.Name] = t }

	register(Tool{
		Name:        "lint_block",
		Description: "Runs the synchronous block-level design lint without persisting. Same rule set create_block + update_block embed in their responses: hero_quality (plain text hero with no hero_graphic / bg=circuit / image_id), headline_length (>12 words), custom_block_duplicate_eyebrow, slop_term / slop_name / slop_company / slop_number against the playbook's banned-phrase list. Input: block_type + data (the JSON the agent is about to commit). Output: design_warnings array. Use this to vet a variant before calling create_block, especially when authoring multiple options for request_clarification.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"block_type":{"type":"string"},
				"data":{"type":"object"}
			},
			"required":["block_type","data"]
		}`),
		Handler: func(_ context.Context, _ *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				BlockType string         `json:"block_type"`
				Data      map[string]any `json:"data"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if strings.TrimSpace(args.BlockType) == "" {
				return "", errors.New("block_type required")
			}
			findings := critique.LintBlockData(args.BlockType, args.Data, agent.DefaultDesignPlaybook())
			return mustJSON(map[string]any{
				"design_warnings":    findings,
				"count":              len(findings),
				"design_inspiration": critique.InspirationsFor(args.BlockType),
				"hint":               "Each finding has name, severity (warning|info), field, message, fix. Zero findings = the block clears the synchronous design lint; the Inspector still grades the rendered HTML after the next build. design_inspiration is the curated 2-3 design-corpus references for this block_type (gap 5).",
			}), nil
		},
	})
}
