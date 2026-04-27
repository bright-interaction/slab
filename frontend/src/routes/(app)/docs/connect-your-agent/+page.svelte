<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { ArrowLeft, Check, Copy, Download, Sparkles } from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as agentKeysApi from '$lib/api/agentKeys';
	import { ApiError } from '$lib/api/client';

	const siteID = $derived(currentSite.value?.id ?? null);
	let copied = $state(false);
	let copiedKey = $state(false);
	let copiedSmoke = $state(false);
	let copiedEnv = $state(false);

	let bootstrapping = $state(false);
	let bootstrap = $state<agentKeysApi.BootstrapResponse | null>(null);

	async function runBootstrap() {
		if (bootstrapping) return;
		if (!siteID) {
			toast.error('Open a site first, then come back to this page.');
			return;
		}
		bootstrapping = true;
		try {
			bootstrap = await agentKeysApi.bootstrap(siteID, 'Quick start key');
			toast.success('Key generated. Save it now: it cannot be retrieved later.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to generate setup bundle.');
		} finally {
			bootstrapping = false;
		}
	}

	function downloadFile(name: string, content: string, type = 'text/markdown') {
		const blob = new Blob([content], { type });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = name;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}

	async function copyText(text: string, marker: 'key' | 'smoke' | 'env') {
		try {
			await navigator.clipboard.writeText(text);
			if (marker === 'key') {
				copiedKey = true;
				setTimeout(() => (copiedKey = false), 1500);
			} else if (marker === 'smoke') {
				copiedSmoke = true;
				setTimeout(() => (copiedSmoke = false), 1500);
			} else {
				copiedEnv = true;
				setTimeout(() => (copiedEnv = false), 1500);
			}
		} catch {
			toast.error('Could not copy. Select the text and copy manually.');
		}
	}

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
			One-click setup below. The manual steps further down are still here in case you want to
			wire it by hand.
		</p>
	</header>

	<div class="mt-8 flex flex-col gap-5">
		<Card padding="md">
			<div class="flex items-start gap-3">
				<div
					class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent/10 text-accent"
				>
					<Sparkles size={14} strokeWidth={1.75} />
				</div>
				<div class="min-w-0 flex-1">
					<h2 class="text-[14px] font-medium text-text-primary">
						One-click setup
					</h2>
					<p class="mt-1 text-[12px] text-text-muted">
						Click the button. We mint a fresh agent key, render a personalised CLAUDE.md
						(with your key, base URL, site name, and current pending_setup snapshot), plus an
						.env file and a smoke-test command. Drop the files into your project and your
						agent is wired in 30 seconds.
					</p>
					{#if !bootstrap}
						<div class="mt-4 flex flex-wrap items-center gap-3">
							<Button
								variant="primary"
								onclick={runBootstrap}
								loading={bootstrapping}
								disabled={bootstrapping || !siteID}
							>
								<Sparkles size={14} strokeWidth={1.75} class="mr-1.5" />
								{siteID ? 'Generate key & download CLAUDE.md' : 'Open a site first'}
							</Button>
							<span class="text-[11px] text-text-muted">
								{siteID
									? 'Adds a fresh key with read+write capabilities to this site.'
									: 'Pick a site from the top-bar site switcher.'}
							</span>
						</div>
					{:else}
						<div class="mt-4 flex flex-col gap-3">
							<div class="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-[12px] text-amber-700 dark:text-amber-300">
								<strong>{bootstrap.note}</strong> The raw key below is shown once; we store
								only its SHA-256 hash.
							</div>
							<div class="flex items-center gap-2">
								<code class="flex-1 truncate rounded-lg border border-border-light bg-bg-elevated px-3 py-2 font-mono text-[11px] text-text-primary">
									{bootstrap.key}
								</code>
								<button
									type="button"
									onclick={() => copyText(bootstrap!.key, 'key')}
									class="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-md border border-border-light bg-bg-elevated px-3 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover"
									aria-label="Copy agent key"
								>
									{#if copiedKey}
										<Check size={12} strokeWidth={2} />
										Copied
									{:else}
										<Copy size={12} strokeWidth={1.75} />
										Copy key
									{/if}
								</button>
							</div>
							<div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
								<button
									type="button"
									onclick={() => downloadFile('CLAUDE.md', bootstrap!.claude_md, 'text/markdown')}
									class="inline-flex items-center justify-center gap-1.5 rounded-md border border-border-light bg-bg-elevated px-3 py-2 text-[12px] text-text-primary transition-colors hover:bg-bg-hover"
								>
									<Download size={13} strokeWidth={1.75} />
									Download CLAUDE.md
								</button>
								<button
									type="button"
									onclick={() => downloadFile('atomicsite.env', bootstrap!.env_file, 'text/plain')}
									class="inline-flex items-center justify-center gap-1.5 rounded-md border border-border-light bg-bg-elevated px-3 py-2 text-[12px] text-text-primary transition-colors hover:bg-bg-hover"
								>
									<Download size={13} strokeWidth={1.75} />
									Download .env
								</button>
								<button
									type="button"
									onclick={() => copyText(bootstrap!.smoke_test, 'smoke')}
									class="inline-flex items-center justify-center gap-1.5 rounded-md border border-border-light bg-bg-elevated px-3 py-2 text-[12px] text-text-primary transition-colors hover:bg-bg-hover"
								>
									{#if copiedSmoke}
										<Check size={13} strokeWidth={2} />
										Smoke test copied
									{:else}
										<Copy size={13} strokeWidth={1.75} />
										Copy smoke-test curl
									{/if}
								</button>
							</div>
							<details class="rounded-lg border border-border-light bg-bg-elevated px-3 py-2">
								<summary class="cursor-pointer text-[12px] text-text-secondary">
									Preview CLAUDE.md
								</summary>
								<pre class="mt-3 max-h-[400px] overflow-auto rounded-md bg-bg-surface p-3 font-mono text-[11px] leading-relaxed text-text-primary">{bootstrap.claude_md}</pre>
							</details>
							<p class="text-[11px] text-text-muted">
								Once you've saved the bundle, the agent can re-fetch this file any time by
								calling <code class="font-mono">GET /api/agent/bootstrap</code>.
							</p>
						</div>
					{/if}
				</div>
			</div>
		</Card>

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
			>{`export ATOMICSITE_API="https://app.atomicsite.example.com"
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
