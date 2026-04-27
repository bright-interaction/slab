<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { ArrowLeft, Check, Copy, Download, Sparkles } from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as agentKeysApi from '$lib/api/agentKeys';
	import { ApiError } from '$lib/api/client';

	const siteID = $derived(currentSite.value?.id ?? null);
	let copiedKey = $state(false);
	let copiedSmoke = $state(false);
	let copiedEnv = $state(false);
	let copiedFetch = $state(false);

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

	async function copyText(text: string, marker: 'key' | 'smoke' | 'env' | 'fetch') {
		try {
			await navigator.clipboard.writeText(text);
			if (marker === 'key') {
				copiedKey = true;
				setTimeout(() => (copiedKey = false), 1500);
			} else if (marker === 'smoke') {
				copiedSmoke = true;
				setTimeout(() => (copiedSmoke = false), 1500);
			} else if (marker === 'env') {
				copiedEnv = true;
				setTimeout(() => (copiedEnv = false), 1500);
			} else {
				copiedFetch = true;
				setTimeout(() => (copiedFetch = false), 1500);
			}
		} catch {
			toast.error('Could not copy. Select the text and copy manually.');
		}
	}

	const fetchClaudeMd = `curl -sH "X-Agent-Key: $ATOMICSITE_KEY" \\
  "$ATOMICSITE_API/api/agent/bootstrap" -o CLAUDE.md`;

	const manualEnv = `export ATOMICSITE_API="https://app.slab.example.com"
export ATOMICSITE_KEY="ask_<your key>"`;
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
			Click the button. We mint a key, render a personalised CLAUDE.md, bundle it with an
			.env and a smoke-test curl. Drop the files into your project. Your agent takes it from
			there.
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
						{#if currentSite.value}
							<div class="mt-4 rounded-lg border border-border-light bg-bg-elevated px-3 py-2.5">
								<p class="text-[11px] font-mono uppercase tracking-[0.18em] text-text-muted">
									Will bind to
								</p>
								<div class="mt-1 flex items-baseline gap-2">
									<p class="text-[14px] font-medium text-text-primary">
										{currentSite.value.name}
									</p>
									<code class="font-mono text-[11px] text-text-muted">
										{currentSite.value.slug}
									</code>
								</div>
								<p class="mt-1 font-mono text-[10px] text-text-muted">
									site_id: {currentSite.value.id}
								</p>
								<p class="mt-1 text-[11px] text-text-muted">
									Switch sites in the top-bar dropdown if this isn't the right one. The bundle
									keys, CLAUDE.md, and pending_setup snapshot are all tied to this site.
								</p>
							</div>
						{/if}
						<div class="mt-4 flex flex-wrap items-center gap-3">
							<Button
								variant="primary"
								onclick={runBootstrap}
								loading={bootstrapping}
								disabled={bootstrapping || !siteID}
							>
								<Sparkles size={14} strokeWidth={1.75} class="mr-1.5" />
								{siteID
									? `Generate bundle for ${currentSite.value?.name ?? 'this site'}`
									: 'Open a site first'}
							</Button>
							<span class="text-[11px] text-text-muted">
								{siteID
									? 'Adds a fresh key with read+write capabilities.'
									: 'Pick a site from the top-bar site switcher.'}
							</span>
						</div>
					{:else}
						<div class="mt-4 flex flex-col gap-3">
							<div class="rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-2 text-[12px] text-emerald-700 dark:text-emerald-300">
								Bundle generated for <strong>{bootstrap.site_name}</strong>
								<code class="ml-1 font-mono text-[10px] opacity-80">
									(site_id: {bootstrap.site_id})
								</code>
							</div>
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
				What the bundle contains
			</h2>
			<dl class="mt-4 space-y-4">
				<div>
					<dt class="text-[13px] font-medium text-text-primary">
						<code class="font-mono text-[12px]">CLAUDE.md</code>
					</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Personalised for this site: name, domain, base URL, your fresh key, and a
						snapshot of the current pending_setup array as a starting checklist. Drop it
						at the root of the project the agent will work in. Cursor users: rename to
						<code class="font-mono">.cursorrules</code>.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">
						<code class="font-mono text-[12px]">atomicsite.env</code>
					</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Two exports: <code class="font-mono">ATOMICSITE_API</code> and
						<code class="font-mono">ATOMICSITE_KEY</code>. Source the file (<code
							class="font-mono">source ./atomicsite.env</code
						>) before invoking the agent, or feed the values into your runtime's secret
						store.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">Smoke-test curl</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						One-liner that hits <code class="font-mono">/api/agent/context</code> and
						pretty-prints the site object. Run it in your terminal first; if you see your
						site name back, you are wired up.
					</dd>
				</div>
			</dl>
			<p class="mt-4 text-[12px] text-text-muted">
				The raw key only appears once. Stash it in a password manager or commit
				<code class="font-mono">atomicsite.env</code> to your secret store now. If you lose
				it, generate another bundle and the previous key still works (use
				<a href={siteID ? `/sites/${siteID}/agent-keys` : '/sites'} class="text-text-primary hover:underline">Agent keys</a>
				to revoke any you don't want).
			</p>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Use it with Claude Code
			</h2>
			<ol class="mt-4 list-decimal space-y-2 pl-5 text-[13px] text-text-secondary">
				<li>
					Save the downloaded files at your project root: <code class="font-mono"
						>CLAUDE.md</code
					>
					and <code class="font-mono">atomicsite.env</code>.
				</li>
				<li>
					In your shell: <code class="font-mono">source ./atomicsite.env</code>.
				</li>
				<li>
					Run <code class="font-mono">claude</code>. Claude Code auto-loads
					<code class="font-mono">CLAUDE.md</code> and inherits the env vars. The agent now
					knows your site, its base URL, the key, and the pending_setup checklist.
				</li>
				<li>
					Give it a task. The CLAUDE.md tells the agent to walk pending_setup first, then
					trigger a build, then read the evaluation, then self-correct.
				</li>
			</ol>
			<pre
				class="mt-4 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`# Example first prompt
"Walk me through pending_setup. Then add a Pricing page with three plan
blocks (Starter, Pro, Enterprise). Build, read the evaluation, and fix
any failing checks. Don't stop until everything is green."`}</pre>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Other agent runtimes
			</h2>
			<dl class="mt-4 space-y-4">
				<div>
					<dt class="text-[13px] font-medium text-text-primary">Cursor / VS Code agents</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Same bundle. Rename <code class="font-mono">CLAUDE.md</code> to
						<code class="font-mono">.cursorrules</code> at the project root, or paste its
						contents into a workspace-level system prompt. Set the env vars in the agent's
						runtime config. The agent uses curl directly from its shell tool.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">n8n / generic HTTP</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Webhook trigger -> HTTP Request node. Set the
						<code class="font-mono">X-Agent-Key</code> header. Hit any agent endpoint. Right
						for scheduled rebuilds, content sync from a CMS, or pre-flight evaluation runs.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">CI / GitHub Actions</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Store the key as a repo secret. Run a curl-based job on every push to trigger
						a build and gate the merge on the evaluation grade.
					</dd>
				</div>
			</dl>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Already have a key? Skip the bundle.
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				If you already issued a key from
				<a
					href={siteID ? `/sites/${siteID}/agent-keys` : '/sites'}
					class="text-text-primary hover:underline">Agent keys</a
				>, set the two env vars manually:
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{manualEnv}</pre>
			<p class="mt-4 text-[13px] text-text-secondary">
				Then fetch the personalised CLAUDE.md straight from the agent endpoint:
			</p>
			<div class="mt-3 flex items-start gap-2">
				<pre
					class="flex-1 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
				>{fetchClaudeMd}</pre>
				<button
					type="button"
					onclick={() => copyText(fetchClaudeMd, 'fetch')}
					class="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-md border border-border-light bg-bg-elevated px-3 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover"
					aria-label="Copy fetch command"
				>
					{#if copiedFetch}
						<Check size={12} strokeWidth={2} />
						Copied
					{:else}
						<Copy size={12} strokeWidth={1.75} />
						Copy
					{/if}
				</button>
			</div>
			<p class="mt-3 text-[12px] text-text-muted">
				<code class="font-mono">GET /api/agent/bootstrap</code> regenerates the file every
				call, so the agent always reads the freshest pending_setup. Right for re-syncing
				after the human has edited the site in the admin UI.
			</p>
		</Card>
	</div>
</div>
