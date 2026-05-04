package mcp

import (
	"context"
	"fmt"

	authmw "github.com/bright-interaction/slab/internal/middleware"
	"github.com/bright-interaction/slab/internal/knowledge"
)

// registerKnowledgeResources exposes the curriculum in
// internal/knowledge/docs/ as MCP resources. The catalog lives at
// atomicsite://knowledge/index; each doc is reachable at
// atomicsite://knowledge/<slug> with mimeType text/markdown.
//
// The resources are agent-readable regardless of capability bit; teaching
// the agent costs nothing and serves any session shape.
func (s *Server) registerKnowledgeResources() {
	register := func(r Resource) { s.resources[r.URI] = r }

	register(Resource{
		URI:         "atomicsite://knowledge/index",
		Name:        "Knowledge catalog",
		Description: "Catalog of every curriculum doc the agent can read: stack mastery (Astro, TypeScript, the CSS-variable system, blocks, i18n, security, personalization, cookieproof) and UX mastery (typography, color, spacing, motion, accessibility, performance, forms, nav, dark mode, premium-design principles). Bodies are stripped here; fetch atomicsite://knowledge/<slug> for the full text.",
		MimeType:    "application/json",
		Reader: func(ctx context.Context, agent *authmw.AgentIdentity) (string, error) {
			return mustJSON(map[string]any{
				"docs": knowledge.Catalog(),
				"hint": "Read atomicsite://knowledge/<slug> for full body. Use the master_the_stack prompt to walk the curriculum in reading order.",
			}), nil
		},
	})

	for _, slug := range knowledge.Slugs() {
		slug := slug
		doc, _ := knowledge.Get(slug)
		register(Resource{
			URI:         fmt.Sprintf("atomicsite://knowledge/%s", slug),
			Name:        doc.Title,
			Description: doc.Summary,
			MimeType:    "text/markdown",
			Reader: func(ctx context.Context, _ *authmw.AgentIdentity) (string, error) {
				d, ok := knowledge.Get(slug)
				if !ok {
					return "", fmt.Errorf("knowledge doc %q not found", slug)
				}
				return d.Body, nil
			},
		})
	}
}
