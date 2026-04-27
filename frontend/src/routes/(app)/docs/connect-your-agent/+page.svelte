<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import { ArrowLeft, Check, Copy } from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';
	import { toast } from '$lib/stores/toast.svelte';

	const siteID = $derived(currentSite.value?.id ?? null);
	let copied = $state(false);

	const claudeMd = `# Working on this Atomic Site

You are an AI agent connected to an Atomic Site instance. Your job is to build
and edit a website by calling the agent API.

## Setup
- Base URL: $ATOMICSITE_API
- Auth: X-Agent-Key: $ATOMICSITE_KEY
- Site ID: read from /api/agent/context (your key is scoped to one site)

## First-call workflow (do this BEFORE anything else)
1. Call \`GET /api/agent/context\`.
2. Inspect the \`pending_setup\` array in the response.
3. For every item in pending_setup, walk the user through it before
   touching content:
   - Read the \`why\` to the user; ask only what you need.
   - Call the listed \`endpoint\` with their answer.
   - Re-fetch /api/agent/context and verify the item is gone.
4. Only after pending_setup is empty (or the user explicitly defers an
   item) move on to content work.

## Editing workflow
1. Make edits via the relevant CRUD endpoints. Every write goes through the
   guardrails engine. Read the validation errors and fix them; do not work
   around them.
2. After a meaningful set of edits, trigger a build:
   \`POST /api/agent/build\`
3. Poll status: \`GET /api/agent/build/{buildID}/status\` until done.
4. Fetch the evaluation: \`GET /api/agent/evaluation/{buildID}\`. The evals
   are the source of truth for quality. Fix every failing check before
   declaring the task done.

## Endpoints you will use most
- \`GET /api/agent/context\` -> full site state + pending_setup
- \`PATCH /api/agent/profile\` -> business name, address, contact emails
  (drives Organization JSON-LD, security.txt, legal pages)
- \`PATCH /api/agent/branding\` -> colours, fonts, meta_title,
  meta_description, og_image_id, favicon_id
- \`PATCH /api/agent/settings\` -> analytics + seo + general categories
  (writes to security/allowed-scripts/nginx are admin-only)
- \`POST /api/agent/pages\` and \`PATCH /api/agent/pages/{slug}\`
- \`POST /api/agent/pages/{slug}/blocks\`
- \`POST /api/agent/build\` and \`GET /api/agent/evaluation/{buildID}\`

## Capabilities
Your key carries: ["read", "write"]

## Guardrails (these will reject your writes if you violate them)
- Pages: title 30-60 chars, description 120-160 chars
- Blocks: image blocks need alt + dimensions; CTAs need a label;
  no generic anchor text ("click here", "read more"); no mixed-content
  http:// URLs in https sites
- URL depth: max 3 levels
- Media: alt text required; SVG rejected for safety; SSRF-guarded
  from-url ingestion

## Bring-your-own integrations
- CRM: any HTTPS webhook URL + shared secret. Set via
  \`PATCH /api/agent/settings\` with category=analytics, keys
  crm_webhook_url and crm_webhook_secret. Payloads HMAC-signed
  (X-Atomicsite-Signature, SHA-256 hex).
- Cookie banner: paste any HTML/JS into analytics.cookie_banner_snippet
  (Cookiebot, OneTrust, Termly, Iubenda, your own). Or flip
  analytics.cookieproof_enabled=1 for the bundled CookieProof.
`;

	async function copyClaudeMd() {
		try {
			await navigator.clipboard.writeText(claudeMd);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			toast.error('Could not copy. Select the text and copy manually.');
		}
	}
</script>

<div class="mx-auto max-w-5xl px-6 py-8">
	<a
		href="/docs"
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Documentation
	</a>
	<header class="mt-3 flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Connect your agent
		</h1>
		<p class="text-[13px] text-text-secondary">
			Wire Claude CLI, Cursor, or any HTTP-capable agent to a site in five steps.
		</p>
	</header>

	<div class="mt-8 flex flex-col gap-5">
		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Step 1: Get an agent key
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Open the site you want the agent to work on. Open Agent keys. Generate a key, copy the
				raw value (you only see it once), give it a recognisable name.
			</p>
			<a
				href={siteID ? `/sites/${siteID}/agent-keys` : '/sites'}
				class="mt-4 inline-flex items-center gap-1.5 text-[12px] text-text-primary transition-colors hover:text-text-secondary"
			>
				Open Agent keys
			</a>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Step 2: Export environment variables
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Drop these into your shell profile (zshrc / bashrc) or your agent's env. The agent reads
				both.
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`export ATOMICSITE_API="https://app.slab.example.com"
export ATOMICSITE_KEY="ask_<the key you just generated>"`}</pre>
		</Card>

		<Card padding="md">
			<div class="flex items-start justify-between gap-3">
				<div>
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Step 3: Drop a CLAUDE.md into your project
					</h2>
					<p class="mt-3 text-[13px] text-text-secondary">
						Paste this file at the root of the project the agent will work in. It gives the
						agent the workflow, the capability list, and the guardrails it must respect.
					</p>
				</div>
				<button
					type="button"
					onclick={copyClaudeMd}
					class="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-md border border-border-light bg-bg-elevated px-2.5 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover"
					aria-label="Copy CLAUDE.md template"
				>
					{#if copied}
						<Check size={12} strokeWidth={2} />
						Copied
					{:else}
						<Copy size={12} strokeWidth={1.75} />
						Copy
					{/if}
				</button>
			</div>
			<pre
				class="mt-4 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[11px] leading-relaxed text-text-primary"
			>{claudeMd}</pre>
			<p class="mt-3 text-[12px] text-text-muted">
				Same file works as <span class="font-mono">.cursorrules</span> for Cursor or as a
				system-prompt fragment for any agent runtime.
			</p>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Step 4: Smoke test the connection
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Bootstrap context. If this returns a JSON object with site details, you are wired up.
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`curl -sH "X-Agent-Key: $ATOMICSITE_KEY" \\
  "$ATOMICSITE_API/api/agent/context" | jq .site`}</pre>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Step 5: Hand it to your agent
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Give the agent a real task. The CLAUDE.md tells it where to look. It will read context,
				make edits with curl, trigger a build, read the evaluation, and self-correct.
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`# Example prompt
"Read /api/agent/context. Add a 'Pricing' page with three plan blocks
(Starter, Pro, Enterprise). Trigger a build. Read the evaluation.
Fix any failing checks. Don't stop until everything is green."`}</pre>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Other agent runtimes
			</h2>
			<dl class="mt-4 space-y-4">
				<div>
					<dt class="text-[13px] font-medium text-text-primary">Cursor / VS Code agents</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Same env vars. Save the CLAUDE.md template above as
						<span class="font-mono">.cursorrules</span> in your project root, or paste it into a
						workspace-level system prompt. The agent uses curl directly from its bash tool.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">n8n / generic HTTP</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Webhook trigger → HTTP Request node. Set the
						<span class="font-mono">X-Agent-Key</span> header. Hit any agent endpoint. Right
						for scheduled rebuilds, content sync from a CMS, or pre-flight evaluation runs.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">CI / GitHub Actions</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Store the key as a repo secret. Run a curl-based job on every push to trigger a
						build and gate the merge on the evaluation grade.
					</dd>
				</div>
			</dl>
		</Card>
	</div>
</div>
