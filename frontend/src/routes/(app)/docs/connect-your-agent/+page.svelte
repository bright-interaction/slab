<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import {
		ArrowLeft,
		Check,
		Copy,
		Download,
		FileCode,
		Key,
		Plug,
		RefreshCw,
		Sparkles
	} from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as agentKeysApi from '$lib/api/agentKeys';
	import { ApiError } from '$lib/api/client';

	const siteID = $derived(currentSite.value?.id ?? null);
	let copiedKey = $state(false);
	let copiedSmoke = $state(false);
	let copiedEnv = $state(false);
	let copiedFetch = $state(false);
	let copiedMcp = $state(false);
	let copiedMcpCli = $state(false);

	let mcpClient =
		$state<'claude-desktop' | 'claude-code' | 'cursor' | 'cline'>('claude-desktop');

	let bootstrapping = $state(false);
	let bootstrap = $state<agentKeysApi.BootstrapResponse | null>(null);

	// Step 1: key state.
	// keyAcknowledged becomes true after the user clicks "I've saved my key".
	// We don't store the raw key (only its SHA-256 hash), so once the user
	// dismisses the box the raw value is gone forever. Regenerate mints a
	// new key and resets keyAcknowledged so the new value is shown once.
	let keyAcknowledged = $state(false);

	// Step 2: which connection path the user picked. null until they click
	// one of the two cards. The other path is hidden so they only see the
	// instructions that apply to them.
	let chosenPath = $state<'mcp' | 'bundle' | null>(null);

	async function runBootstrap() {
		if (bootstrapping) return;
		if (!siteID) {
			toast.error('Open a site first, then come back to this page.');
			return;
		}
		bootstrapping = true;
		try {
			bootstrap = await agentKeysApi.bootstrap(siteID, 'Quick start key');
			keyAcknowledged = false;
			toast.success('Key created. Copy it now; you can only see it once.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to create key.');
		} finally {
			bootstrapping = false;
		}
	}

	function regenerateKey() {
		// Mints a new key. The previous key still works (the user manages
		// active keys at /sites/{id}/agent-keys). Resets keyAcknowledged so
		// the new raw value is shown once.
		bootstrap = null;
		keyAcknowledged = false;
		void runBootstrap();
	}

	// Pretty mask for the post-acknowledged state. Shows the prefix +
	// last four characters so the user can recognise their key without us
	// re-exposing the secret.
	function maskKey(k: string): string {
		if (!k) return '';
		const dot = '•';
		if (k.length <= 12) return k.replace(/./g, dot);
		const prefix = k.split('_')[0] + '_';
		const tail = k.slice(-4);
		return `${prefix}${dot.repeat(8)}${tail}`;
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

	const manualEnv = `export ATOMICSITE_API="https://app.atomicsite.example.com"
export ATOMICSITE_KEY="atomic_<your key>"`;

	// MCP card. Default tab is Claude Desktop because that is the most
	// common host today; Cursor and Cline cover the rest. The snippet
	// fills in the bootstrap key automatically once the user clicks
	// Generate further down the page; until then it shows a placeholder.
	const baseURL = $derived(
		typeof window !== 'undefined'
			? window.location.origin
			: 'https://app.atomicsite.example.com'
	);
	const mcpURL = $derived(`${baseURL}/mcp`);
	const mcpKey = $derived(bootstrap?.key ?? 'atomic_<your key>');
	const mcpSnippet = $derived.by(() => {
		// Same JSON shape works for Claude Desktop, Cursor, and Claude Code:
		// HTTP transport is inferred from the url scheme (no "transport"
		// field needed). Cline / Continue / Windsurf still want the
		// explicit transport key per their own MCP docs.
		const httpInferred = {
			mcpServers: {
				atomicsite: {
					url: mcpURL,
					headers: { 'X-Agent-Key': mcpKey }
				}
			}
		};
		const cline = {
			mcpServers: {
				atomicsite: {
					transport: 'http',
					url: mcpURL,
					headers: { 'X-Agent-Key': mcpKey }
				}
			}
		};
		switch (mcpClient) {
			case 'claude-desktop':
			case 'claude-code':
			case 'cursor':
				return JSON.stringify(httpInferred, null, 2);
			case 'cline':
				return JSON.stringify(cline, null, 2);
		}
	});
	const mcpConfigPath = $derived.by(() => {
		switch (mcpClient) {
			case 'claude-desktop':
				return '~/Library/Application Support/Claude/claude_desktop_config.json (macOS) or %APPDATA%\\Claude\\claude_desktop_config.json (Windows)';
			case 'claude-code':
				return 'Run the CLI command below (recommended) OR edit .mcp.json (project, git-tracked) or ~/.claude.json (user-global)';
			case 'cursor':
				return '.cursor/mcp.json (in your project) or ~/.cursor/mcp.json (global)';
			case 'cline':
				return 'Cline -> MCP Servers panel -> Edit JSON';
		}
	});
	// Claude Code can be wired via a single CLI command. Recommended over
	// hand-editing the JSON because it picks the right scope file + handles
	// quoting. --scope project tracks .mcp.json in git so the whole team
	// gets the connection without each person setting it up.
	// Positional args (name + url) come BEFORE the flags. Putting flags
	// first and positional args last fails with "missing required
	// argument 'name'" because the parser consumes the next token after
	// --header as the header value, not as the name.
	const mcpCli = $derived(
		`claude mcp add atomicsite ${mcpURL} --transport http --scope project --header "X-Agent-Key: ${mcpKey}"`
	);
	async function copyMcp() {
		try {
			await navigator.clipboard.writeText(mcpSnippet);
			copiedMcp = true;
			setTimeout(() => (copiedMcp = false), 1500);
		} catch {
			toast.error('Could not copy. Select the text and copy manually.');
		}
	}
	async function copyMcpCli() {
		try {
			await navigator.clipboard.writeText(mcpCli);
			copiedMcpCli = true;
			setTimeout(() => (copiedMcpCli = false), 1500);
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
			Two steps. Create your key, then pick how you want to connect.
		</p>
	</header>

	<!-- ===================================================================
		STEP 1: Create your key.
		Everyone needs a key. We mint it server-side, return it once,
		store only the SHA-256 hash. After the user clicks "I've saved
		my key", the raw value is hidden behind a mask. Regenerate
		creates a new key and shows it once; the old key keeps working
		until the user revokes it from /sites/{id}/agent-keys.
	==================================================================== -->
	<section class="mt-8">
		<div class="mb-3 flex items-baseline gap-2">
			<span class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Step 1
			</span>
			<h2 class="text-[14px] font-medium text-text-primary">Create your key</h2>
		</div>
		<Card padding="md">
			<div class="flex items-start gap-3">
				<div
					class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent/10 text-accent"
				>
					<Key size={14} strokeWidth={1.75} />
				</div>
				<div class="min-w-0 flex-1">
					{#if !bootstrap}
						<p class="text-[13px] text-text-secondary">
							A key authenticates your agent against this site. We store only
							the SHA-256 hash; you see the raw key once. Save it in a password
							manager.
						</p>
						{#if currentSite.value}
							<div class="mt-3 rounded-lg border border-border-light bg-bg-elevated px-3 py-2 text-[12px]">
								<span class="text-text-muted">Will create a key for </span>
								<strong class="text-text-primary">{currentSite.value.name}</strong>
								<span class="text-text-muted"> ({currentSite.value.slug}).</span>
							</div>
						{/if}
						<div class="mt-4">
							<Button
								variant="primary"
								onclick={runBootstrap}
								loading={bootstrapping}
								disabled={bootstrapping || !siteID}
							>
								<Key size={14} strokeWidth={1.75} class="mr-1.5" />
								{siteID ? 'Create key' : 'Open a site first'}
							</Button>
						</div>
					{:else if !keyAcknowledged}
						<p class="text-[13px] text-text-secondary">
							Key created for <strong class="text-text-primary">{bootstrap.site_name}</strong>.
							Copy it now. After you click "I've saved my key" it will be hidden
							and you cannot see it again.
						</p>
						<div class="mt-3 flex items-center gap-2">
							<code class="flex-1 truncate rounded-lg border border-border-light bg-bg-elevated px-3 py-2 font-mono text-[11px] text-text-primary">
								{bootstrap.key}
							</code>
							<button
								type="button"
								onclick={() => copyText(bootstrap!.key, 'key')}
								class="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-md border border-border-light bg-bg-elevated px-3 text-[11px] text-text-secondary transition-colors hover:bg-bg-hover"
								aria-label="Copy key"
							>
								{#if copiedKey}
									<Check size={12} strokeWidth={2} />
									Copied
								{:else}
									<Copy size={12} strokeWidth={1.75} />
									Copy
								{/if}
							</button>
						</div>
						<div class="mt-3 flex flex-wrap items-center gap-3">
							<Button variant="primary" onclick={() => (keyAcknowledged = true)}>
								<Check size={14} strokeWidth={1.75} class="mr-1.5" />
								I've saved my key
							</Button>
							<span class="text-[11px] text-text-muted">
								Once dismissed, the raw key is gone.
							</span>
						</div>
					{:else}
						<p class="text-[13px] text-text-secondary">
							Your key is saved. We can no longer show the raw value.
						</p>
						<div class="mt-3 flex items-center gap-2">
							<code class="flex-1 truncate rounded-lg border border-border-light bg-bg-elevated px-3 py-2 font-mono text-[11px] text-text-muted">
								{maskKey(bootstrap.key)}
							</code>
							<Button variant="secondary" onclick={regenerateKey} loading={bootstrapping}>
								<RefreshCw size={13} strokeWidth={1.75} class="mr-1.5" />
								Regenerate
							</Button>
						</div>
						<p class="mt-2 text-[11px] text-text-muted">
							Regenerate creates a new key and shows it once. Your old key keeps
							working until you revoke it on the
							<a
								href={siteID ? `/sites/${siteID}/agent-keys` : '/sites'}
								class="text-text-primary hover:underline"
							>
								Agent keys
							</a>
							page.
						</p>
					{/if}
				</div>
			</div>
		</Card>
	</section>

	<!-- ===================================================================
		STEP 2: Pick a connection path.
		Locked until Step 1 produces a key. Two cards: MCP (recommended)
		and CLAUDE.md bundle (fallback). Only the chosen path renders
		below so users only read instructions that apply to them.
	==================================================================== -->
	<section class="mt-10">
		<div class="mb-3 flex items-baseline gap-2">
			<span class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Step 2
			</span>
			<h2 class="text-[14px] font-medium text-text-primary">Pick how to connect</h2>
		</div>
		{#if !bootstrap}
			<Card padding="md">
				<p class="text-[12px] text-text-muted">
					Create a key in Step 1 first. Step 2 unlocks once you have one.
				</p>
			</Card>
		{:else}
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
				<button
					type="button"
					onclick={() => (chosenPath = 'mcp')}
					class="text-left transition-colors {chosenPath === 'mcp'
						? 'ring-2 ring-accent'
						: ''}"
				>
					<Card padding="md" hoverable>
						<div class="flex items-start gap-3">
							<div
								class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent/10 text-accent"
							>
								<Plug size={14} strokeWidth={1.75} />
							</div>
							<div class="min-w-0 flex-1">
								<p class="text-[13px] font-medium text-text-primary">
									MCP (recommended)
								</p>
								<p class="mt-1 text-[12px] text-text-muted">
									For Claude Desktop, Claude Code, Cursor, Cline. One config
									entry per machine. No files in your project.
								</p>
							</div>
						</div>
					</Card>
				</button>
				<button
					type="button"
					onclick={() => (chosenPath = 'bundle')}
					class="text-left transition-colors {chosenPath === 'bundle'
						? 'ring-2 ring-accent'
						: ''}"
				>
					<Card padding="md" hoverable>
						<div class="flex items-start gap-3">
							<div
								class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent/10 text-accent"
							>
								<FileCode size={14} strokeWidth={1.75} />
							</div>
							<div class="min-w-0 flex-1">
								<p class="text-[13px] font-medium text-text-primary">
									CLAUDE.md bundle (fallback)
								</p>
								<p class="mt-1 text-[12px] text-text-muted">
									For agents that don't speak MCP yet (older Cursor, n8n,
									generic HTTP, CI scripts). Drop two files into your project.
								</p>
							</div>
						</div>
					</Card>
				</button>
			</div>
		{/if}
	</section>

	<!-- ===================================================================
		MCP path. Only renders when chosenPath === 'mcp'.
		Four client tabs. Claude Code tab leads with the CLI command +
		first-time terminal walkthrough so a new user can copy-paste their
		way to a working connection without reading any other docs.
	==================================================================== -->
	{#if bootstrap && chosenPath === 'mcp'}
		<section class="mt-6">
			<Card padding="md">
				<h3 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Pick your client
				</h3>
				<div class="mt-3 flex flex-wrap gap-1 rounded-lg border border-border-light bg-bg-elevated p-0.5">
					{#each [{ id: 'claude-desktop' as const, label: 'Claude Desktop' }, { id: 'claude-code' as const, label: 'Claude Code' }, { id: 'cursor' as const, label: 'Cursor' }, { id: 'cline' as const, label: 'Cline / Continue / Windsurf' }] as tab (tab.id)}
						<button
							type="button"
							onclick={() => (mcpClient = tab.id)}
							class="rounded-md px-3 py-1.5 text-[12px] transition-colors {mcpClient ===
							tab.id
								? 'bg-bg-surface text-text-primary shadow-sm'
								: 'text-text-muted hover:text-text-primary'}"
						>
							{tab.label}
						</button>
					{/each}
				</div>

				<p class="mt-4 text-[12px] text-text-muted">
					Paste this into:
					<code class="font-mono text-[11px] text-text-secondary">{mcpConfigPath}</code>
				</p>

				{#if mcpClient === 'claude-code'}
					<!-- Claude Code: CLI is the recommended path. The command writes
						.mcp.json (project scope) so the connection rides along in git
						and any teammate gets it on pull. -->
					<div class="mt-3">
						<p class="text-[12px] text-text-secondary">
							In your terminal, from the folder you want Claude Code to work in:
						</p>
						<div class="relative mt-2">
							<pre
								class="overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-3 font-mono text-[11.5px] text-text-secondary"><code>{mcpCli}</code></pre>
							<button
								type="button"
								onclick={copyMcpCli}
								class="absolute right-2 top-2 inline-flex h-7 items-center gap-1 rounded-md border border-border-light bg-bg-surface px-2 text-[11px] text-text-secondary hover:text-text-primary"
								aria-label="Copy CLI command"
							>
								{#if copiedMcpCli}
									<Check size={12} strokeWidth={2} />
									Copied
								{:else}
									<Copy size={12} strokeWidth={1.75} />
									Copy
								{/if}
							</button>
						</div>

						<!-- First-time walkthrough. Plain step-by-step for a user who has
							never run claude / mcp / a terminal command for an agent. -->
						<details class="mt-3 rounded-lg border border-border-light bg-bg-elevated/50">
							<summary class="cursor-pointer px-3 py-2 text-[12px] text-text-secondary">
								First time? Full walkthrough
							</summary>
							<ol class="space-y-3 px-3 pb-3 pt-2 text-[12px] text-text-secondary">
								<li>
									<p class="font-medium text-text-primary">1. Open Terminal.</p>
									<p class="mt-0.5 text-text-muted">
										On macOS: Cmd+Space, type "Terminal", hit enter. On Windows:
										Win+R, type "wt" or "cmd", hit enter.
									</p>
								</li>
								<li>
									<p class="font-medium text-text-primary">
										2. Make a project folder and go into it.
									</p>
									<pre class="mt-1 overflow-x-auto rounded-md bg-bg-surface px-2 py-1.5 font-mono text-[11px] text-text-primary"><code>mkdir ~/Desktop/my-atomicsite-project
cd ~/Desktop/my-atomicsite-project</code></pre>
								</li>
								<li>
									<p class="font-medium text-text-primary">
										3. Paste the command above and hit enter.
									</p>
									<p class="mt-0.5 text-text-muted">
										It writes a small <code class="font-mono">.mcp.json</code> file
										inside the folder. If you see "missing required argument",
										make sure the URL comes right after "atomicsite" and before
										the flags.
									</p>
								</li>
								<li>
									<p class="font-medium text-text-primary">4. Start Claude Code.</p>
									<pre class="mt-1 overflow-x-auto rounded-md bg-bg-surface px-2 py-1.5 font-mono text-[11px] text-text-primary"><code>claude</code></pre>
									<p class="mt-0.5 text-text-muted">
										Don't have it? Install with
										<code class="font-mono">npm install -g @anthropic-ai/claude-code</code>.
									</p>
								</li>
								<li>
									<p class="font-medium text-text-primary">
										5. Inside Claude Code, type
										<code class="font-mono">/mcp</code>
										and hit enter.
									</p>
									<p class="mt-0.5 text-text-muted">
										You should see "atomicsite" listed as connected. Now ask the
										agent something: "Walk me through pending_setup."
									</p>
								</li>
							</ol>
						</details>

						<p class="mt-3 text-[12px] text-text-muted">
							Or paste this JSON into <code class="font-mono">.mcp.json</code>
							or <code class="font-mono">~/.claude.json</code> by hand:
						</p>
					</div>
				{/if}

				<div class="relative mt-2">
					<pre
						class="overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-3 font-mono text-[11.5px] text-text-secondary"><code>{mcpSnippet}</code></pre>
					<button
						type="button"
						onclick={copyMcp}
						class="absolute right-2 top-2 inline-flex h-7 items-center gap-1 rounded-md border border-border-light bg-bg-surface px-2 text-[11px] text-text-secondary hover:text-text-primary"
						aria-label="Copy MCP snippet"
					>
						{#if copiedMcp}
							<Check size={12} strokeWidth={2} />
							Copied
						{:else}
							<Copy size={12} strokeWidth={1.75} />
							Copy
						{/if}
					</button>
				</div>

				<p class="mt-3 text-[12px] text-text-muted">
					Restart your client after saving the file. Atomicsite tools should
					appear in its tool picker.
				</p>

				<div
					class="mt-4 rounded-lg border border-border-light bg-bg-elevated/40 p-3 text-[12px] text-text-secondary"
				>
					<p class="font-medium text-text-primary">What MCP exposes</p>
					<ul class="mt-1.5 list-disc space-y-0.5 pl-4 text-text-muted">
						<li>22 tools for content + config (pages, blocks, branding, profile, settings, media, build, evaluation)</li>
						<li>6 read-only resources (settings_catalog, security_posture, i18n, pending_setup, structure, full context)</li>
						<li>5 prompt templates (walk_through_pending_setup, audit_seo, connect_analytics, add_iframe_integration, create_landing_page)</li>
					</ul>
					<p class="mt-2 font-medium text-text-primary">What MCP does not expose</p>
					<ul class="mt-1.5 list-disc space-y-0.5 pl-4 text-text-muted">
						<li>Visitor metadata (name, email, lifecycle, last_topic): visitor PII never reaches an AI host.</li>
						<li>Security-header writes, allowed-scripts writes: admin-only because they widen attack surface.</li>
					</ul>
				</div>
			</Card>
		</section>
	{/if}

	<!-- ===================================================================
		Bundle path. Only renders when chosenPath === 'bundle'.
		Three pieces: download CLAUDE.md, download .env, copy a smoke-
		test curl. Plus instructions for non-Claude-Code agents (Cursor
		old, n8n, CI). The same bootstrap response carries all the file
		bodies.
	==================================================================== -->
	{#if bootstrap && chosenPath === 'bundle'}
		<section class="mt-6 flex flex-col gap-5">
			<Card padding="md">
				<h3 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Download the bundle
				</h3>
				<p class="mt-2 text-[12px] text-text-muted">
					Three pieces: a personalised <code class="font-mono">CLAUDE.md</code>, an
					<code class="font-mono">atomicsite.env</code> with your API URL and key,
					and a smoke-test command. Drop them at the root of the project the
					agent will work in.
				</p>
				<div class="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-3">
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
				<details class="mt-4 rounded-lg border border-border-light bg-bg-elevated px-3 py-2">
					<summary class="cursor-pointer text-[12px] text-text-secondary">
						Preview CLAUDE.md
					</summary>
					<pre class="mt-3 max-h-[400px] overflow-auto rounded-md bg-bg-surface p-3 font-mono text-[11px] leading-relaxed text-text-primary">{bootstrap.claude_md}</pre>
				</details>
			</Card>

			<Card padding="md">
				<h3 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Use it with Claude Code
				</h3>
				<ol class="mt-3 list-decimal space-y-2 pl-5 text-[13px] text-text-secondary">
					<li>
						Save the two files at your project root:
						<code class="font-mono">CLAUDE.md</code> and
						<code class="font-mono">atomicsite.env</code>.
					</li>
					<li>
						In Terminal: <code class="font-mono">source ./atomicsite.env</code>.
					</li>
					<li>
						Run <code class="font-mono">claude</code>. Claude Code reads
						<code class="font-mono">CLAUDE.md</code> automatically and inherits the
						env vars.
					</li>
					<li>
						Ask the agent to walk through pending_setup, then build the site, then
						read the evaluation, then fix any failing checks.
					</li>
				</ol>
			</Card>

			<Card padding="md">
				<h3 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Other runtimes
				</h3>
				<dl class="mt-3 space-y-3 text-[13px]">
					<div>
						<dt class="font-medium text-text-primary">Cursor / VS Code agents (no MCP)</dt>
						<dd class="mt-1 text-[12px] text-text-secondary">
							Same bundle. Rename <code class="font-mono">CLAUDE.md</code> to
							<code class="font-mono">.cursorrules</code> at the project root.
							Set the env vars in the agent's runtime config. The agent uses curl
							from its shell tool.
						</dd>
					</div>
					<div>
						<dt class="font-medium text-text-primary">n8n / generic HTTP</dt>
						<dd class="mt-1 text-[12px] text-text-secondary">
							Webhook trigger to HTTP Request node. Set the
							<code class="font-mono">X-Agent-Key</code> header. Hit any agent
							endpoint. Works for scheduled rebuilds, content sync from a CMS,
							pre-flight evaluation runs.
						</dd>
					</div>
					<div>
						<dt class="font-medium text-text-primary">CI / GitHub Actions</dt>
						<dd class="mt-1 text-[12px] text-text-secondary">
							Store the key as a repo secret. Run a curl-based job on every push
							to trigger a build and gate the merge on the evaluation grade.
						</dd>
					</div>
				</dl>
			</Card>

			<Card padding="md">
				<h3 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Re-fetch CLAUDE.md any time
				</h3>
				<p class="mt-2 text-[12px] text-text-secondary">
					The agent endpoint regenerates the personalised CLAUDE.md on every
					call, so you always get the freshest pending_setup snapshot.
				</p>
				<div class="mt-3 flex items-start gap-2">
					<pre
						class="flex-1 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-3 font-mono text-[11.5px] text-text-primary"><code>{fetchClaudeMd}</code></pre>
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
			</Card>
		</section>
	{/if}
</div>
