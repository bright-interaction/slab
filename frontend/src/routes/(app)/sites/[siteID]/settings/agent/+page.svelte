<script lang="ts">
	import * as agentSurfaceApi from '$lib/api/agentSurface';
	import type { AgentSurface, AgentSurfaceResponse } from '$lib/api/agentSurface';
	import { isAvailable } from '$lib/api/agentSurface';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Site } from '$lib/api/types';
	import { Bot, BookOpen, Wrench, Database, MessageSquare, ShieldCheck, Network, Lock, Copy } from 'lucide-svelte';

	let { data }: { data: { site: Site } } = $props();
	const siteID = $derived(data.site.id);

	let loading = $state(true);
	let surface = $state<AgentSurface | null>(null);
	let unavailableReason = $state<string | null>(null);

	async function load() {
		loading = true;
		try {
			const response: AgentSurfaceResponse = await agentSurfaceApi.get(siteID);
			if (!isAvailable(response)) {
				unavailableReason = response.reason;
				surface = null;
				return;
			}
			surface = response;
			unavailableReason = null;
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load agent surface.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	async function copyEndpoint() {
		if (!surface?.endpoint) return;
		try {
			await navigator.clipboard.writeText(surface.endpoint);
			toast.success('Endpoint copied.');
		} catch {
			toast.error('Copy failed; select and copy manually.');
		}
	}

	const stackDocs = $derived(surface?.knowledge.filter((d) => d.category === 'stack').sort((a, b) => a.order - b.order) ?? []);
	const uxDocs = $derived(surface?.knowledge.filter((d) => d.category === 'ux').sort((a, b) => a.order - b.order) ?? []);
	const writeTools = $derived(surface?.tools.filter((t) => t.requires_write) ?? []);
	const readTools = $derived(surface?.tools.filter((t) => !t.requires_write) ?? []);
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Agent</h1>
		<p class="text-[13px] text-text-secondary">
			What an AI agent connecting via MCP can see and do for this site.
			Read-only inventory; the agent itself uses the MCP protocol at <code class="text-text-primary">{surface?.endpoint ?? '/api/agent/mcp'}</code>.
		</p>
	</header>

	{#if loading}
		<div class="mt-12 flex items-center gap-2 text-text-muted">
			<Spinner /> Loading surface...
		</div>
	{:else if unavailableReason}
		<Card padding="md" class="mt-8 border-warning/30 bg-warning/5">
			<div class="flex items-start gap-3">
				<span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-warning/30 bg-warning/5 text-warning">
					<Bot size={16} strokeWidth={1.5} />
				</span>
				<div class="flex flex-col gap-0.5">
					<p class="text-[13px] font-medium text-text-primary">MCP not configured</p>
					<p class="text-[12px] text-text-muted">{unavailableReason}</p>
				</div>
			</div>
		</Card>
	{:else if surface}
		<!-- At-a-glance counts -->
		<section class="mt-8 grid grid-cols-2 gap-3 md:grid-cols-5">
			<Card padding="md">
				<div class="flex flex-col gap-1">
					<span class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Tools</span>
					<span class="text-2xl font-light text-text-primary">{surface.counts.tools}</span>
					<span class="text-[12px] text-text-secondary">{surface.counts.write_tools} write-capable</span>
				</div>
			</Card>
			<Card padding="md">
				<div class="flex flex-col gap-1">
					<span class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Resources</span>
					<span class="text-2xl font-light text-text-primary">{surface.counts.resources}</span>
					<span class="text-[12px] text-text-secondary">read-only URIs</span>
				</div>
			</Card>
			<Card padding="md">
				<div class="flex flex-col gap-1">
					<span class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Prompts</span>
					<span class="text-2xl font-light text-text-primary">{surface.counts.prompts}</span>
					<span class="text-[12px] text-text-secondary">playbook templates</span>
				</div>
			</Card>
			<Card padding="md">
				<div class="flex flex-col gap-1">
					<span class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Curriculum</span>
					<span class="text-2xl font-light text-text-primary">{surface.counts.knowledge_docs}</span>
					<span class="text-[12px] text-text-secondary">stack + UX docs</span>
				</div>
			</Card>
			<Card padding="md">
				<div class="flex flex-col gap-1">
					<span class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Protocol</span>
					<span class="text-base font-mono text-text-primary">{surface.protocol_version}</span>
					<span class="text-[12px] text-text-secondary">MCP v{surface.version}</span>
				</div>
			</Card>
		</section>

		<!-- Endpoint copy -->
		<section class="mt-6">
			<Card padding="md">
				<div class="flex items-center justify-between gap-3">
					<div class="flex items-center gap-3 min-w-0">
						<span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-border-light bg-bg-elevated text-text-secondary">
							<Network size={16} strokeWidth={1.5} />
						</span>
						<div class="flex flex-col gap-0.5 min-w-0">
							<p class="text-[13px] font-medium text-text-primary">JSON-RPC endpoint</p>
							<code class="block truncate text-[12px] text-text-muted">{surface.endpoint}</code>
						</div>
					</div>
					<button onclick={copyEndpoint} class="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-border-light bg-bg-elevated px-2.5 py-1.5 text-[12px] text-text-secondary hover:bg-bg-hover" type="button">
						<Copy size={12} strokeWidth={1.5} /> Copy
					</button>
				</div>
			</Card>
		</section>

		<!-- Tools -->
		<section class="mt-10">
			<header class="mb-3 flex items-center gap-2">
				<Wrench size={14} strokeWidth={1.5} class="text-text-muted" />
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Tools ({surface.counts.tools})
				</h2>
			</header>
			<p class="mb-4 text-[12px] text-text-secondary">
				Callable JSON-RPC functions. Write-capable tools require an agent key minted with the <code>write</code> capability; admin operators can mint keys at <a href="/sites/{siteID}/settings" class="text-accent underline-offset-2 hover:underline">Settings -> Agent keys</a>.
			</p>
			<div class="space-y-3">
				{#if writeTools.length > 0}
					<details class="group" open>
						<summary class="cursor-pointer text-[12px] font-medium text-text-primary hover:text-accent">
							Write tools ({writeTools.length})
						</summary>
						<div class="mt-3 grid grid-cols-1 gap-2 md:grid-cols-2">
							{#each writeTools as t (t.name)}
								<Card padding="sm">
									<div class="flex items-start gap-2">
										<span class="mt-0.5 inline-flex h-5 items-center rounded-full border border-warning/30 bg-warning/5 px-1.5 text-[10px] font-mono uppercase tracking-wide text-warning">w</span>
										<div class="min-w-0">
											<p class="font-mono text-[12px] text-text-primary">{t.name}</p>
											<p class="mt-0.5 text-[11px] text-text-muted line-clamp-3">{t.description}</p>
										</div>
									</div>
								</Card>
							{/each}
						</div>
					</details>
				{/if}
				<details class="group" open>
					<summary class="cursor-pointer text-[12px] font-medium text-text-primary hover:text-accent">
						Read tools ({readTools.length})
					</summary>
					<div class="mt-3 grid grid-cols-1 gap-2 md:grid-cols-2">
						{#each readTools as t (t.name)}
							<Card padding="sm">
								<div class="flex items-start gap-2">
									<span class="mt-0.5 inline-flex h-5 items-center rounded-full border border-border-light bg-bg-elevated px-1.5 text-[10px] font-mono uppercase tracking-wide text-text-secondary">r</span>
									<div class="min-w-0">
										<p class="font-mono text-[12px] text-text-primary">{t.name}</p>
										<p class="mt-0.5 text-[11px] text-text-muted line-clamp-3">{t.description}</p>
									</div>
								</div>
							</Card>
						{/each}
					</div>
				</details>
			</div>
		</section>

		<!-- Resources -->
		<section class="mt-10">
			<header class="mb-3 flex items-center gap-2">
				<Database size={14} strokeWidth={1.5} class="text-text-muted" />
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Resources ({surface.counts.resources})
				</h2>
			</header>
			<p class="mb-4 text-[12px] text-text-secondary">
				Read-only URIs the agent fetches via <code>resources/read</code>. The curriculum is exposed under <code>atomicsite://knowledge/*</code>; the cross-reference graph at <code>atomicsite://meta/knowledge-graph</code> lets the agent navigate the surface in one fetch.
			</p>
			<div class="grid grid-cols-1 gap-2 md:grid-cols-2">
				{#each surface.resources as r (r.uri)}
					<Card padding="sm">
						<p class="font-mono text-[12px] text-text-primary">{r.uri}</p>
						<p class="mt-0.5 text-[11px] text-text-muted line-clamp-3">{r.description}</p>
						<p class="mt-1 text-[10px] font-mono uppercase tracking-wide text-text-muted">{r.mime_type}</p>
					</Card>
				{/each}
			</div>
		</section>

		<!-- Prompts -->
		<section class="mt-10">
			<header class="mb-3 flex items-center gap-2">
				<MessageSquare size={14} strokeWidth={1.5} class="text-text-muted" />
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Prompts ({surface.counts.prompts})
				</h2>
			</header>
			<p class="mb-4 text-[12px] text-text-secondary">
				Playbook templates the user picks from a dropdown in the host UI. Each prompt is a starting frame for a multi-step flow.
			</p>
			<div class="grid grid-cols-1 gap-2 md:grid-cols-2">
				{#each surface.prompts as p (p.name)}
					<Card padding="sm">
						<p class="font-mono text-[12px] text-text-primary">{p.name}</p>
						<p class="mt-0.5 text-[11px] text-text-muted line-clamp-3">{p.description}</p>
					</Card>
				{/each}
			</div>
		</section>

		<!-- Curriculum -->
		<section class="mt-10">
			<header class="mb-3 flex items-center gap-2">
				<BookOpen size={14} strokeWidth={1.5} class="text-text-muted" />
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Curriculum ({surface.counts.knowledge_docs})
				</h2>
			</header>
			<p class="mb-4 text-[12px] text-text-secondary">
				Embedded markdown the agent reads to learn the stack and the design discipline. Stack docs cover Astro, TypeScript, the CSS-variable system, blocks, i18n, security, personalization, cookieproof. UX docs cover typography, color, spacing, motion, accessibility, performance, forms, nav, dark mode, and premium-design principles.
			</p>
			<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
				<div>
					<h3 class="mb-2 text-[12px] font-medium text-text-primary">Stack</h3>
					<ol class="space-y-1.5">
						{#each stackDocs as d (d.slug)}
							<li class="flex items-baseline gap-2">
								<span class="font-mono text-[10px] text-text-muted">{d.order}</span>
								<div class="min-w-0">
									<p class="text-[12px] text-text-primary">{d.title}</p>
									<p class="text-[11px] text-text-muted line-clamp-2">{d.summary}</p>
								</div>
							</li>
						{/each}
					</ol>
				</div>
				<div>
					<h3 class="mb-2 text-[12px] font-medium text-text-primary">UX / UI</h3>
					<ol class="space-y-1.5">
						{#each uxDocs as d (d.slug)}
							<li class="flex items-baseline gap-2">
								<span class="font-mono text-[10px] text-text-muted">{d.order}</span>
								<div class="min-w-0">
									<p class="text-[12px] text-text-primary">{d.title}</p>
									<p class="text-[11px] text-text-muted line-clamp-2">{d.summary}</p>
								</div>
							</li>
						{/each}
					</ol>
				</div>
			</div>
		</section>

		<!-- Privacy invariants -->
		<section class="mt-10">
			<header class="mb-3 flex items-center gap-2">
				<Lock size={14} strokeWidth={1.5} class="text-text-muted" />
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Privacy invariants</h2>
			</header>
			<Card padding="md" class="border-success/20 bg-success/5">
				<ul class="space-y-1.5 text-[12px] text-text-secondary">
					{#each surface.privacy_invariants as inv}
						<li class="flex items-start gap-2">
							<ShieldCheck size={12} strokeWidth={1.5} class="mt-0.5 shrink-0 text-success" />
							<span>{inv}</span>
						</li>
					{/each}
				</ul>
			</Card>
		</section>
	{/if}
</div>
