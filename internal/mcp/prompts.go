package mcp

import (
	"context"

	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
)

// registerPrompts wires reusable prompt templates the user picks from a
// dropdown in the host UI. Each prompt is a starting frame the model
// rides into the conversation; users still drive the actual work.
func (s *Server) registerPrompts() {
	register := func(p Prompt) { s.prompts[p.Name] = p }

	register(Prompt{
		Name:        "walk_through_pending_setup",
		Description: "Walk through every pending_setup item with the user, one at a time, and resolve each. Use this on first session to bring the site from wizard-defaults to launch-ready.",
		Render: func(ctx context.Context, agent *authmw.AgentIdentity, _ map[string]string) ([]PromptMessage, error) {
			return []PromptMessage{
				{Role: "user", Content: Content{Type: "text", Text: "Read atomicsite://site/pending_setup. For each item in the list, ask me what value to use, then call bulk_upsert_settings (or update_profile / update_branding) to apply it. After each one, recompute the list and continue with the next remaining item. Stop when pending_setup is empty."}},
			}, nil
		},
	})

	register(Prompt{
		Name:        "audit_seo",
		Description: "Run an SEO audit using the latest evaluation results + settings_catalog. Surfaces failing checks and suggests fixes the agent can apply directly.",
		Render: func(ctx context.Context, agent *authmw.AgentIdentity, _ map[string]string) ([]PromptMessage, error) {
			return []PromptMessage{
				{Role: "user", Content: Content{Type: "text", Text: "Trigger a build, wait for it to complete, then read the evaluation results. For every failing SEO check, propose a concrete fix (settings change, page metadata edit, block edit) and ask me to confirm before applying. Read atomicsite://site/settings_catalog when you need to know which settings keys to write."}},
			}, nil
		},
	})

	register(Prompt{
		Name:        "connect_analytics",
		Description: "Walk through analytics setup: CookieProof / GA4 / Umami / personalization. The agent configures the toggles but does not touch any visitor data.",
		Render: func(ctx context.Context, agent *authmw.AgentIdentity, _ map[string]string) ([]PromptMessage, error) {
			return []PromptMessage{
				{Role: "user", Content: Content{Type: "text", Text: "Help me wire analytics. Read atomicsite://site/settings_catalog for the analytics category. Ask which providers I want (CookieProof / GA4 / Umami / personalization), collect the IDs, and call bulk_upsert_settings to flip the toggles. Important: you have NO access to visitor data or PII; you can only configure the providers."}},
			}, nil
		},
	})

	register(Prompt{
		Name:        "add_iframe_integration",
		Description: "Playbook for embedding a third-party iframe (cal.com, YouTube, Stripe Checkout). Reads trusted_domains to detect whether the host is already whitelisted; if not, asks the human to add it via the admin URL.",
		Render: func(ctx context.Context, agent *authmw.AgentIdentity, _ map[string]string) ([]PromptMessage, error) {
			return []PromptMessage{
				{Role: "user", Content: Content{Type: "text", Text: "I want to embed an iframe. Read atomicsite://site/security_posture. Ask which host I want to embed. Check trusted_domains.frame for that host. If present, insert the iframe block. If absent, tell me the exact admin URL to visit and the kind to pick (frame), then wait for me to confirm I've added it before inserting the block."}},
			}, nil
		},
	})

	register(Prompt{
		Name:        "create_landing_page",
		Description: "Create a high-conversion landing page that suppresses the global nav + footer (hide_global_blocks=true). Walks through the standard sections (hero, problem, solution, social proof, CTA).",
		Render: func(ctx context.Context, agent *authmw.AgentIdentity, _ map[string]string) ([]PromptMessage, error) {
			return []PromptMessage{
				{Role: "user", Content: Content{Type: "text", Text: "Create a landing page. Ask for the slug + headline + the offer. Create the page with hide_global_blocks=1. Then create blocks in order: hero, value-prop, social-proof, cta. Read atomicsite://site/structure first to see what blocks already exist on similar pages so the new page matches the site's voice."}},
			}, nil
		},
	})
}
