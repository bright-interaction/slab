package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	authmw "github.com/brightinteraction/atomicsite/internal/middleware"
	"github.com/brightinteraction/atomicsite/internal/store"
)

// registerExtraTools wires the operator-mutable surfaces beyond pages
// and blocks: components, css_classes, knowledgebase, guardrails,
// allowed_scripts, fonts (read+delete; upload still goes through the
// REST endpoint because it is multipart), design_references, plus
// get_capabilities for self-introspection.
//
// Each tool is a thin pass-through to the existing sqlc query the REST
// handler also calls. Same validation, same constraints, same audit
// shape. The only thing MCP adds is the auth + write-cap gating that
// every tool already runs through.
func (s *Server) registerExtraTools() {
	register := func(t Tool) { s.tools[t.Name] = t }

	// --- components ------------------------------------------------------

	register(Tool{
		Name:        "list_components",
		Description: "Lists every reusable component registered for the site (Astro snippets, Svelte fragments, etc.). Components are template blocks the agent can compose into pages without re-authoring CSS.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, _ json.RawMessage) (string, error) {
			rows, err := s.queries.ListComponentsBySite(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			return mustJSON(rows), nil
		},
	})

	register(Tool{
		Name:        "create_component",
		Description: "Registers a new reusable component. Required: name, template (Astro/Svelte source). Optional: category, props_schema (JSON), css_classes (space-separated names), usage_note.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"name":{"type":"string"},
				"template":{"type":"string"},
				"category":{"type":"string"},
				"props_schema":{"type":"string"},
				"css_classes":{"type":"string"},
				"usage_note":{"type":"string"}
			},
			"required":["name","template"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				Name        string `json:"name"`
				Template    string `json:"template"`
				Category    string `json:"category"`
				PropsSchema string `json:"props_schema"`
				CssClasses  string `json:"css_classes"`
				UsageNote   string `json:"usage_note"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if strings.TrimSpace(args.Name) == "" || strings.TrimSpace(args.Template) == "" {
				return "", errors.New("name and template are required")
			}
			id := newID()
			if err := s.queries.CreateComponent(ctx, store.CreateComponentParams{
				ID:          id,
				SiteID:      agent.SiteID,
				Name:        args.Name,
				Category:    args.Category,
				Template:    args.Template,
				PropsSchema: args.PropsSchema,
				CssClasses:  args.CssClasses,
				UsageNote:   args.UsageNote,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": id, "status": "created"}), nil
		},
	})

	register(Tool{
		Name:        "update_component",
		Description: "Updates a registered component. Required: id. All other fields are optional but the API replaces the whole row, so pass the existing values for fields you do not want to change.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"id":{"type":"string"},
				"name":{"type":"string"},
				"template":{"type":"string"},
				"category":{"type":"string"},
				"props_schema":{"type":"string"},
				"css_classes":{"type":"string"},
				"usage_note":{"type":"string"}
			},
			"required":["id","name","template"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Template    string `json:"template"`
				Category    string `json:"category"`
				PropsSchema string `json:"props_schema"`
				CssClasses  string `json:"css_classes"`
				UsageNote   string `json:"usage_note"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			existing, err := s.queries.GetComponentByID(ctx, args.ID)
			if err != nil {
				return "", fmt.Errorf("component %s not found", args.ID)
			}
			if existing.SiteID != agent.SiteID {
				return "", errors.New("component belongs to another site")
			}
			if err := s.queries.UpdateComponent(ctx, store.UpdateComponentParams{
				ID:          args.ID,
				Name:        args.Name,
				Category:    args.Category,
				Template:    args.Template,
				PropsSchema: args.PropsSchema,
				CssClasses:  args.CssClasses,
				UsageNote:   args.UsageNote,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "updated"}), nil
		},
	})

	register(Tool{
		Name:        "delete_component",
		Description: "Deletes a registered component by id. Cross-tenant guarded.",
		InputSchema: schema(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			existing, err := s.queries.GetComponentByID(ctx, args.ID)
			if err != nil {
				return "", fmt.Errorf("component %s not found", args.ID)
			}
			if existing.SiteID != agent.SiteID {
				return "", errors.New("component belongs to another site")
			}
			if err := s.queries.DeleteComponent(ctx, args.ID); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "deleted"}), nil
		},
	})

	// --- css_classes -----------------------------------------------------

	register(Tool{
		Name:        "list_css_classes",
		Description: "Lists every CSS class registered for the site, ordered by category and sort_order. The agent reads this before composing pages so it can reuse existing classes instead of pasting one-off styles.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, _ json.RawMessage) (string, error) {
			rows, err := s.queries.ListCSSClassesBySite(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			return mustJSON(rows), nil
		},
	})

	register(Tool{
		Name:        "upsert_css_class",
		Description: "Creates or updates a CSS class. If id is provided, updates that class. Otherwise creates a new class with the given name (must be unique per site). Required: name, css. Optional: id (for update), category, usage_note, sort_order.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"id":{"type":"string"},
				"name":{"type":"string"},
				"css":{"type":"string"},
				"category":{"type":"string"},
				"usage_note":{"type":"string"},
				"sort_order":{"type":"number"}
			},
			"required":["name","css"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Css       string `json:"css"`
				Category  string `json:"category"`
				UsageNote string `json:"usage_note"`
				SortOrder int64  `json:"sort_order"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if strings.TrimSpace(args.Name) == "" || strings.TrimSpace(args.Css) == "" {
				return "", errors.New("name and css are required")
			}
			if args.ID != "" {
				existing, err := s.queries.GetCSSClassByID(ctx, args.ID)
				if err != nil {
					return "", fmt.Errorf("css class %s not found", args.ID)
				}
				if existing.SiteID != agent.SiteID {
					return "", errors.New("css class belongs to another site")
				}
				if err := s.queries.UpdateCSSClass(ctx, store.UpdateCSSClassParams{
					ID:        args.ID,
					Name:      args.Name,
					Category:  args.Category,
					Css:       args.Css,
					UsageNote: args.UsageNote,
					SortOrder: args.SortOrder,
				}); err != nil {
					return "", err
				}
				return mustJSON(map[string]string{"id": args.ID, "status": "updated"}), nil
			}
			id := newID()
			if err := s.queries.CreateCSSClass(ctx, store.CreateCSSClassParams{
				ID:        id,
				SiteID:    agent.SiteID,
				Name:      args.Name,
				Category:  args.Category,
				Css:       args.Css,
				UsageNote: args.UsageNote,
				SortOrder: args.SortOrder,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": id, "status": "created"}), nil
		},
	})

	register(Tool{
		Name:        "delete_css_class",
		Description: "Deletes a CSS class by id. Cross-tenant guarded.",
		InputSchema: schema(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			existing, err := s.queries.GetCSSClassByID(ctx, args.ID)
			if err != nil {
				return "", fmt.Errorf("css class %s not found", args.ID)
			}
			if existing.SiteID != agent.SiteID {
				return "", errors.New("css class belongs to another site")
			}
			if err := s.queries.DeleteCSSClass(ctx, args.ID); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "deleted"}), nil
		},
	})

	// --- knowledgebase ---------------------------------------------------

	register(Tool{
		Name:        "list_knowledgebase",
		Description: "Lists every knowledgebase entry for the site (brand voice, product positioning, FAQ source, anything operators want the agent to remember). The agent reads this before authoring copy so the voice matches.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, _ json.RawMessage) (string, error) {
			rows, err := s.queries.ListKnowledgebaseBySite(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			return mustJSON(rows), nil
		},
	})

	register(Tool{
		Name:        "create_knowledgebase_entry",
		Description: "Adds a new knowledgebase entry. Required: title, content. Optional: category (e.g. brand-voice, product, faq), sort_order.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"title":{"type":"string"},
				"content":{"type":"string"},
				"category":{"type":"string"},
				"sort_order":{"type":"number"}
			},
			"required":["title","content"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				Title     string `json:"title"`
				Content   string `json:"content"`
				Category  string `json:"category"`
				SortOrder int64  `json:"sort_order"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if strings.TrimSpace(args.Title) == "" || strings.TrimSpace(args.Content) == "" {
				return "", errors.New("title and content are required")
			}
			id := newID()
			if err := s.queries.CreateKnowledgebaseEntry(ctx, store.CreateKnowledgebaseEntryParams{
				ID:        id,
				SiteID:    agent.SiteID,
				Category:  args.Category,
				Title:     args.Title,
				Content:   args.Content,
				SortOrder: args.SortOrder,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": id, "status": "created"}), nil
		},
	})

	register(Tool{
		Name:        "update_knowledgebase_entry",
		Description: "Updates an existing knowledgebase entry. Required: id, title, content. Optional: category, is_active (0 or 1), sort_order.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"id":{"type":"string"},
				"title":{"type":"string"},
				"content":{"type":"string"},
				"category":{"type":"string"},
				"is_active":{"type":"number"},
				"sort_order":{"type":"number"}
			},
			"required":["id","title","content"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Content   string `json:"content"`
				Category  string `json:"category"`
				IsActive  int64  `json:"is_active"`
				SortOrder int64  `json:"sort_order"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			existing, err := s.queries.GetKnowledgebaseByID(ctx, args.ID)
			if err != nil {
				return "", fmt.Errorf("knowledgebase entry %s not found", args.ID)
			}
			if existing.SiteID != agent.SiteID {
				return "", errors.New("knowledgebase entry belongs to another site")
			}
			if err := s.queries.UpdateKnowledgebaseEntry(ctx, store.UpdateKnowledgebaseEntryParams{
				ID:        args.ID,
				Category:  args.Category,
				Title:     args.Title,
				Content:   args.Content,
				IsActive:  args.IsActive,
				SortOrder: args.SortOrder,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "updated"}), nil
		},
	})

	register(Tool{
		Name:        "delete_knowledgebase_entry",
		Description: "Deletes a knowledgebase entry by id. Cross-tenant guarded.",
		InputSchema: schema(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			existing, err := s.queries.GetKnowledgebaseByID(ctx, args.ID)
			if err != nil {
				return "", fmt.Errorf("knowledgebase entry %s not found", args.ID)
			}
			if existing.SiteID != agent.SiteID {
				return "", errors.New("knowledgebase entry belongs to another site")
			}
			if err := s.queries.DeleteKnowledgebaseEntry(ctx, args.ID); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "deleted"}), nil
		},
	})

	// --- guardrails ------------------------------------------------------

	register(Tool{
		Name:        "list_guardrails",
		Description: "Lists every guardrail rule for the site. Guardrails are content-validation rules the build pipeline enforces (e.g. minimum H1 length, banned phrases, required CTA per page).",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, _ json.RawMessage) (string, error) {
			rows, err := s.queries.ListGuardrailsBySite(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			return mustJSON(rows), nil
		},
	})

	register(Tool{
		Name:        "create_guardrail",
		Description: "Adds a new guardrail rule. Required: rule_type, target, value. Optional: severity (warn|error, defaults to warn).",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"rule_type":{"type":"string"},
				"target":{"type":"string"},
				"value":{"type":"string"},
				"severity":{"type":"string","enum":["warn","error"]}
			},
			"required":["rule_type","target","value"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				RuleType string `json:"rule_type"`
				Target   string `json:"target"`
				Value    string `json:"value"`
				Severity string `json:"severity"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if args.Severity == "" {
				args.Severity = "warn"
			}
			if args.Severity != "warn" && args.Severity != "error" {
				return "", errors.New("severity must be warn or error")
			}
			id := newID()
			if err := s.queries.CreateGuardrail(ctx, store.CreateGuardrailParams{
				ID:       id,
				SiteID:   agent.SiteID,
				RuleType: args.RuleType,
				Target:   args.Target,
				Value:    args.Value,
				Severity: args.Severity,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": id, "status": "created"}), nil
		},
	})

	register(Tool{
		Name:        "delete_guardrail",
		Description: "Deletes a guardrail rule by id. Cross-tenant guarded.",
		InputSchema: schema(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			existing, err := s.queries.GetGuardrailByID(ctx, args.ID)
			if err != nil {
				return "", fmt.Errorf("guardrail %s not found", args.ID)
			}
			if existing.SiteID != agent.SiteID {
				return "", errors.New("guardrail belongs to another site")
			}
			if err := s.queries.DeleteGuardrail(ctx, args.ID); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "deleted"}), nil
		},
	})

	// --- allowed_scripts -------------------------------------------------

	register(Tool{
		Name:        "list_allowed_scripts",
		Description: "Lists every whitelisted external origin for the site. Each entry has kind (script/frame/image/media/connect/all) which routes it to the corresponding CSP directive when the builder regenerates security headers.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, _ json.RawMessage) (string, error) {
			rows, err := s.queries.ListAllowedScriptsBySite(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			return mustJSON(rows), nil
		},
	})

	register(Tool{
		Name:        "register_allowed_script",
		Description: "Whitelists an external origin so the builder includes it in the CSP. Required: domain (hostname only, no scheme), kind (script|frame|image|media|connect|all), purpose (one-line description). The next build picks up the change.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"domain":{"type":"string"},
				"kind":{"type":"string","enum":["script","frame","image","media","connect","all"]},
				"purpose":{"type":"string"}
			},
			"required":["domain","kind","purpose"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				Domain  string `json:"domain"`
				Kind    string `json:"kind"`
				Purpose string `json:"purpose"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			args.Domain = strings.TrimSpace(args.Domain)
			if args.Domain == "" {
				return "", errors.New("domain is required")
			}
			if strings.Contains(args.Domain, "://") || strings.Contains(args.Domain, "/") {
				return "", errors.New("domain must be a hostname only, no scheme or path")
			}
			switch args.Kind {
			case "script", "frame", "image", "media", "connect", "all":
			default:
				return "", errors.New("kind must be one of: script, frame, image, media, connect, all")
			}
			id := newID()
			if err := s.queries.CreateAllowedScript(ctx, store.CreateAllowedScriptParams{
				ID:      id,
				SiteID:  agent.SiteID,
				Domain:  args.Domain,
				Purpose: args.Purpose,
				Kind:    args.Kind,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": id, "status": "registered"}), nil
		},
	})

	register(Tool{
		Name:        "revoke_allowed_script",
		Description: "Revokes a whitelisted origin by id. The next build regenerates CSP directives without it.",
		InputSchema: schema(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			scripts, err := s.queries.ListAllowedScriptsBySite(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			found := false
			for _, sc := range scripts {
				if sc.ID == args.ID {
					found = true
					break
				}
			}
			if !found {
				return "", fmt.Errorf("allowed script %s not found in this site", args.ID)
			}
			if err := s.queries.DeleteAllowedScript(ctx, args.ID); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "revoked"}), nil
		},
	})

	// --- fonts (read + delete; upload stays on REST due to multipart) ----

	register(Tool{
		Name:        "list_fonts",
		Description: "Lists every uploaded font (woff2) for the site. Each row carries family_name, weight, style, file_path. Read this before authoring typography to know which faces and weights are available.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, _ json.RawMessage) (string, error) {
			rows, err := s.queries.ListSiteFonts(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			return mustJSON(rows), nil
		},
	})

	register(Tool{
		Name:        "delete_font",
		Description: "Removes an uploaded font by id. The id comes from list_fonts. Cross-tenant guarded.",
		InputSchema: schema(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if err := s.queries.DeleteSiteFont(ctx, store.DeleteSiteFontParams{
				ID:     args.ID,
				SiteID: agent.SiteID,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "deleted"}), nil
		},
	})

	// --- design references ----------------------------------------------

	register(Tool{
		Name:        "list_design_references",
		Description: "Lists every external GitHub repo registered as a design reference for this site (taste-skill, high-end-visual-design, custom). Each row carries the fetched bundle (README, tailwind config, sample components) under fetched_json.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, _ json.RawMessage) (string, error) {
			rows, err := s.queries.ListDesignReferences(ctx, agent.SiteID)
			if err != nil {
				return "", err
			}
			return mustJSON(rows), nil
		},
	})

	register(Tool{
		Name:        "add_design_reference",
		Description: "Registers a public GitHub repo as a design reference. Required: url (https://github.com/owner/repo). Optional: label (display name), ref_type (design-system|component-library|reference-site, defaults to design-system). The handler fetches a representative bundle (package.json, README, tailwind config, sample components) and stores it for the agent to read.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"url":{"type":"string"},
				"label":{"type":"string"},
				"ref_type":{"type":"string","enum":["design-system","component-library","reference-site"]}
			},
			"required":["url"]
		}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				URL     string `json:"url"`
				Label   string `json:"label"`
				RefType string `json:"ref_type"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			parsed, err := url.Parse(strings.TrimSpace(args.URL))
			if err != nil || parsed.Host != "github.com" {
				return "", errors.New("url must be a public github.com repo URL")
			}
			if args.RefType == "" {
				args.RefType = "design-system"
			}
			switch args.RefType {
			case "design-system", "component-library", "reference-site":
			default:
				return "", errors.New("ref_type must be design-system, component-library, or reference-site")
			}
			// The MCP tool does not run the fetch synchronously (the
			// handler does that with a 10s timeout). Instead, we record the
			// reference with empty fetched_json and tell the caller to
			// invoke refresh_design_reference to populate the bundle.
			// This keeps the MCP request fast and avoids blocking the
			// JSON-RPC connection on a slow GitHub fetch.
			id := newID()
			if err := s.queries.CreateDesignReference(ctx, store.CreateDesignReferenceParams{
				ID:          id,
				SiteID:      agent.SiteID,
				Url:         args.URL,
				Label:       strings.TrimSpace(args.Label),
				RefType:     args.RefType,
				FetchedJson: "",
				FetchedAt:   "",
			}); err != nil {
				if strings.Contains(err.Error(), "UNIQUE") {
					return "", errors.New("this repo is already registered as a design reference")
				}
				return "", err
			}
			return mustJSON(map[string]string{
				"id":     id,
				"status": "registered",
				"hint":   "Call refresh_design_reference with this id to fetch the bundle from GitHub.",
			}), nil
		},
	})

	register(Tool{
		Name:        "delete_design_reference",
		Description: "Removes a design reference by id. Cross-tenant guarded.",
		InputSchema: schema(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`),
		RequiresWrite: true,
		Handler: func(ctx context.Context, agent *authmw.AgentIdentity, raw json.RawMessage) (string, error) {
			var args struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return "", err
			}
			if err := s.queries.DeleteDesignReference(ctx, store.DeleteDesignReferenceParams{
				ID:     args.ID,
				SiteID: agent.SiteID,
			}); err != nil {
				return "", err
			}
			return mustJSON(map[string]string{"id": args.ID, "status": "deleted"}), nil
		},
	})

	// --- self-introspection ---------------------------------------------

	register(Tool{
		Name:        "get_capabilities",
		Description: "Returns the same self-describing snapshot as atomicsite://meta/capabilities resource: agent identity, write capability, full tools/resources/prompts surface, eval thresholds, privacy invariants. Use this when a tools/call surface is more convenient than resources/read.",
		InputSchema: schema(`{"type":"object","properties":{}}`),
		Handler: func(_ context.Context, agent *authmw.AgentIdentity, _ json.RawMessage) (string, error) {
			return mustJSON(s.capabilitiesSnapshot(agent)), nil
		},
	})
}
