// Package apps holds the curated catalogue of installable third-party
// MCP integrations + the storage shape backing it. Sprint 4 of the
// WP/Webflow replacement roadmap.
//
// Slice A (2026-05-24) ships the marketplace + per-site install
// flow + credential storage. Slice B will wire the upstream MCP
// proxy + namespaced tool exposure so an agent connected to
// atomicsite can fan out to every installed app's MCP server.
package apps

// CredentialField describes one entry in an app's credentials_schema_json.
// The install dialog renders one input per field; the values get stored
// under credentials_json keyed by `Key`.
//
//	{kind: "secret"} -> masked password input (api_key, webhook_secret)
//	{kind: "string"} -> plain text input (account_id, region)
//	{kind: "url"}    -> URL-validated input (workspace_url)
//
// Required fields fail the install POST if missing; optional may stay
// empty.
type CredentialField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Kind        string `json:"kind"`
	Required    bool   `json:"required"`
	Placeholder string `json:"placeholder,omitempty"`
	Help        string `json:"help,omitempty"`
}

// CuratedApp mirrors the apps table shape for the seed loader at
// boot. UpsertApp idempotently writes these on every start so
// updates to descriptions / new credential fields land without a
// migration. publisher_url + docs_url default to the same string
// for brevity.
type CuratedApp struct {
	ID          string
	Slug        string
	Name        string
	Description string
	Category    string
	Publisher   string
	IconURL     string
	MCPURL      string
	DocsURL     string
	Version     string
	Credentials []CredentialField
}

// Curated is the v1 catalogue of supported MCP integrations. Eight
// apps cover the categories tenants ask about most often (commerce,
// scheduling, email, crm, dev, analytics, ai). Each app's MCPURL
// points at the publisher-operated endpoint Slice B's proxy will
// dial.
var Curated = []CuratedApp{
	{
		ID:          "app_stripe_v1",
		Slug:        "stripe",
		Name:        "Stripe",
		Description: "Hosted checkout, subscriptions, refunds. Use when Mollie's not enough (US-first markets, Apple Pay, complex billing). Atomicsite's own checkout already covers Mollie EU one-shot; install Stripe for everything else.",
		Category:    "commerce",
		Publisher:   "Stripe Inc.",
		IconURL:     "https://stripe.com/img/v3/home/social.png",
		MCPURL:      "https://mcp.stripe.com/",
		DocsURL:     "https://docs.stripe.com/agents",
		Version:     "1.0.0",
		Credentials: []CredentialField{
			{Key: "secret_key", Label: "Secret API key", Kind: "secret", Required: true, Placeholder: "sk_test_...", Help: "Find under Developers -> API keys on the Stripe dashboard."},
			{Key: "webhook_secret", Label: "Webhook signing secret", Kind: "secret", Placeholder: "whsec_..."},
		},
	},
	{
		ID:          "app_calcom_v1",
		Slug:        "calcom",
		Name:        "Cal.com",
		Description: "Scheduling. Embed event-type booking forms on the public site; the agent can list / create events via MCP. Self-hosted at cal.example.com for BI tenants.",
		Category:    "scheduling",
		Publisher:   "Cal.com Inc.",
		IconURL:     "https://cal.com/api/logo?type=icon",
		MCPURL:      "https://cal.example.com/api/mcp",
		DocsURL:     "https://cal.com/docs",
		Version:     "1.0.0",
		Credentials: []CredentialField{
			{Key: "api_key", Label: "Personal access token", Kind: "secret", Required: true, Placeholder: "cal_live_...", Help: "Settings -> Developer -> API keys."},
		},
	},
	{
		ID:          "app_brevo_v1",
		Slug:        "brevo",
		Name:        "Brevo",
		Description: "Transactional + marketing email. Capture form submissions into a list; agent can send broadcasts via MCP. EU-hosted alternative to Mailchimp / SendGrid.",
		Category:    "marketing",
		Publisher:   "Brevo (ex-Sendinblue)",
		IconURL:     "https://www.brevo.com/wp-content/uploads/2023/06/cropped-favicon-32x32.png",
		MCPURL:      "https://api.brevo.com/v3/mcp",
		DocsURL:     "https://developers.brevo.com",
		Version:     "1.0.0",
		Credentials: []CredentialField{
			{Key: "api_key", Label: "API key (v3)", Kind: "secret", Required: true, Placeholder: "xkeysib-...", Help: "SMTP & API -> API keys -> Generate new API key."},
		},
	},
	{
		ID:          "app_hubspot_v1",
		Slug:        "hubspot",
		Name:        "HubSpot",
		Description: "CRM + marketing automation. Sync site form submissions to contacts; agent can read / update deals via MCP. Free CRM tier covers most small sites.",
		Category:    "crm",
		Publisher:   "HubSpot Inc.",
		IconURL:     "https://www.hubspot.com/favicon.ico",
		MCPURL:      "https://api.hubspot.com/mcp/v1",
		DocsURL:     "https://developers.hubspot.com",
		Version:     "1.0.0",
		Credentials: []CredentialField{
			{Key: "private_app_token", Label: "Private app token", Kind: "secret", Required: true, Placeholder: "pat-...", Help: "Settings -> Integrations -> Private Apps."},
		},
	},
	{
		ID:          "app_slack_v1",
		Slug:        "slack",
		Name:        "Slack",
		Description: "Send notifications + read channel messages. Wire form submissions to a #leads channel; agent can announce deploys + answer team questions via MCP.",
		Category:    "messaging",
		Publisher:   "Slack Technologies",
		IconURL:     "https://a.slack-edge.com/80588/marketing/img/meta/favicon-32.png",
		MCPURL:      "https://slack.com/api/mcp",
		DocsURL:     "https://api.slack.com/docs/mcp",
		Version:     "1.0.0",
		Credentials: []CredentialField{
			{Key: "bot_token", Label: "Bot user OAuth token", Kind: "secret", Required: true, Placeholder: "xoxb-...", Help: "Create a Slack app -> OAuth & Permissions -> Bot User OAuth Token."},
		},
	},
	{
		ID:          "app_discord_v1",
		Slug:        "discord",
		Name:        "Discord",
		Description: "Community channel notifications. Post site events to a Discord channel; agent can moderate + announce via MCP.",
		Category:    "messaging",
		Publisher:   "Discord Inc.",
		IconURL:     "https://discord.com/assets/favicon.ico",
		MCPURL:      "https://discord.com/api/v10/mcp",
		DocsURL:     "https://discord.com/developers/docs",
		Version:     "1.0.0",
		Credentials: []CredentialField{
			{Key: "bot_token", Label: "Bot token", Kind: "secret", Required: true, Placeholder: "MTk4Nj...", Help: "Developer portal -> your application -> Bot -> Reset Token."},
		},
	},
	{
		ID:          "app_algolia_v1",
		Slug:        "algolia",
		Name:        "Algolia",
		Description: "Hosted search. Use when the built-in Pagefind index isn't enough (typo tolerance, faceted search, personalised ranking). Agent can manage indexes + records via MCP.",
		Category:    "dev",
		Publisher:   "Algolia Inc.",
		IconURL:     "https://www.algolia.com/favicon.ico",
		MCPURL:      "https://mcp.algolia.com/v1",
		DocsURL:     "https://www.algolia.com/doc/mcp",
		Version:     "1.0.0",
		Credentials: []CredentialField{
			{Key: "app_id", Label: "Application ID", Kind: "string", Required: true, Placeholder: "ABCDE12345"},
			{Key: "admin_api_key", Label: "Admin API key", Kind: "secret", Required: true, Placeholder: "Required for write operations."},
		},
	},
	{
		ID:          "app_openai_v1",
		Slug:        "openai",
		Name:        "OpenAI",
		Description: "Optional secondary model surface (DALL-E, Whisper, etc). Atomicsite's BYOAI agent flow is already Claude-native; install OpenAI when you need an image or speech model under the same MCP namespace.",
		Category:    "ai",
		Publisher:   "OpenAI",
		IconURL:     "https://openai.com/favicon.ico",
		MCPURL:      "https://api.openai.com/v1/mcp",
		DocsURL:     "https://platform.openai.com/docs",
		Version:     "1.0.0",
		Credentials: []CredentialField{
			{Key: "api_key", Label: "API key", Kind: "secret", Required: true, Placeholder: "sk-..."},
		},
	},
}
