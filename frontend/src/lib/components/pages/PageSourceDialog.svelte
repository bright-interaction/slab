<script lang="ts">
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import * as pagesApi from '$lib/api/pages';
	import { toast } from '$lib/stores/toast.svelte';
	import { Copy, Check, RefreshCw } from 'lucide-svelte';

	let {
		open = $bindable(false),
		siteID,
		pageID
	}: {
		open?: boolean;
		siteID: string;
		pageID: string;
	} = $props();

	let loading = $state(false);
	let error = $state<string | null>(null);
	let astro = $state<string | null>(null);
	let slug = $state<string>('');
	let copied = $state(false);

	async function load(): Promise<void> {
		loading = true;
		error = null;
		try {
			const res = await pagesApi.previewSource(siteID, pageID);
			astro = res.astro;
			slug = res.slug;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to render page';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (open && astro === null && !loading) {
			void load();
		}
		if (!open) {
			// Drop the cached source on close so the next open re-fetches.
			astro = null;
			error = null;
			copied = false;
		}
	});

	async function copySource(): Promise<void> {
		if (!astro) return;
		try {
			await navigator.clipboard.writeText(astro);
			copied = true;
			toast.success('Source copied');
			setTimeout(() => (copied = false), 1500);
		} catch {
			toast.error('Could not copy');
		}
	}
</script>

<Dialog bind:open title="Page source" size="lg">
	<div class="flex items-center justify-between">
		<p class="text-[12px] text-text-muted">
			{slug ? `/${slug}.astro` : 'Loading…'} · what bun build will see for this page
		</p>
		<div class="flex items-center gap-1">
			<button
				type="button"
				aria-label="Refresh"
				title="Refresh"
				class="inline-flex h-7 w-7 items-center justify-center rounded text-text-muted hover:bg-bg-hover hover:text-text-primary disabled:opacity-50"
				disabled={loading}
				onclick={() => {
					astro = null;
					void load();
				}}
			>
				<RefreshCw class="h-3.5 w-3.5 {loading ? 'animate-spin' : ''}" />
			</button>
			<button
				type="button"
				aria-label="Copy source"
				title="Copy"
				class="inline-flex h-7 w-7 items-center justify-center rounded text-text-muted hover:bg-bg-hover hover:text-text-primary disabled:opacity-50"
				disabled={!astro}
				onclick={copySource}
			>
				{#if copied}
					<Check class="h-3.5 w-3.5 text-accent" />
				{:else}
					<Copy class="h-3.5 w-3.5" />
				{/if}
			</button>
		</div>
	</div>

	<div class="mt-3 max-h-[60vh] overflow-auto rounded-lg border border-border-light bg-bg-elevated/40">
		{#if loading}
			<p class="px-4 py-8 text-center text-[12px] text-text-muted">Rendering…</p>
		{:else if error}
			<p class="px-4 py-3 text-[12px] text-danger">{error}</p>
		{:else if astro !== null}
			<pre class="px-4 py-3 font-mono text-[11.5px] leading-relaxed text-text-primary">{astro}</pre>
		{/if}
	</div>

	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>Close</Button>
	{/snippet}
</Dialog>
