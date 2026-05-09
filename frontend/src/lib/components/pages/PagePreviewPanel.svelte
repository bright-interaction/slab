<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { RefreshCw, Smartphone, Monitor, Tablet, ExternalLink } from 'lucide-svelte';

	interface Props {
		siteID: string;
		pageID: string;
		// Increment this from the parent (e.g. on block save) to force a refetch.
		refreshNonce?: number;
	}

	let { siteID, pageID, refreshNonce = 0 }: Props = $props();

	type Device = 'desktop' | 'tablet' | 'mobile';
	const DEVICES: { id: Device; label: string; width: number; icon: typeof Monitor }[] = [
		{ id: 'desktop', label: 'Desktop', width: 1280, icon: Monitor },
		{ id: 'tablet', label: 'Tablet', width: 820, icon: Tablet },
		{ id: 'mobile', label: 'Mobile', width: 390, icon: Smartphone }
	];

	let device = $state<Device>('desktop');
	let html = $state<string>('');
	let loading = $state<boolean>(true);
	let err = $state<string | null>(null);
	let lastFetched = $state<number>(0);
	let abortCtl: AbortController | null = null;

	async function load(force = false) {
		if (loading && !force) {
			abortCtl?.abort();
		}
		abortCtl = new AbortController();
		loading = true;
		err = null;
		try {
			const resp = await fetch(
				`/api/sites/${encodeURIComponent(siteID)}/pages/${encodeURIComponent(pageID)}/draft-preview`,
				{ credentials: 'include', signal: abortCtl.signal, cache: 'no-store' }
			);
			if (!resp.ok) {
				throw new Error(`HTTP ${resp.status}`);
			}
			const ct = resp.headers.get('content-type') ?? '';
			if (!ct.startsWith('text/html')) {
				throw new Error(`unexpected content-type ${ct}`);
			}
			html = await resp.text();
			lastFetched = Date.now();
		} catch (e: unknown) {
			if ((e as { name?: string })?.name === 'AbortError') return;
			err = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		void load(true);
	});

	onDestroy(() => {
		abortCtl?.abort();
	});

	// Refetch when the parent signals a change (e.g. block saved).
	$effect(() => {
		if (refreshNonce > 0) {
			void load(true);
		}
	});

	// Refetch when site or page changes (defensive: the parent route
	// usually remounts the panel, but in case it stays mounted while a
	// prop swaps, the effect keeps the iframe in sync).
	$effect(() => {
		if (siteID && pageID) {
			void load(true);
		}
	});

	const currentDevice = $derived(DEVICES.find((d) => d.id === device) ?? DEVICES[0]);
	const lastFetchedLabel = $derived.by(() => {
		if (!lastFetched) return '';
		const secs = Math.round((Date.now() - lastFetched) / 1000);
		if (secs < 5) return 'just now';
		if (secs < 60) return `${secs}s ago`;
		return `${Math.round(secs / 60)}m ago`;
	});

	// Refresh tick re-evaluates the lastFetchedLabel display.
	let _tick = $state(0);
	const _interval = setInterval(() => (_tick = _tick + 1), 5000);
	onDestroy(() => clearInterval(_interval));
	$effect(() => {
		// Touch _tick so the derived label re-runs.
		void _tick;
	});

	function openInNewTab() {
		const url = `/api/sites/${encodeURIComponent(siteID)}/pages/${encodeURIComponent(pageID)}/draft-preview`;
		window.open(url, '_blank', 'noopener,noreferrer');
	}
</script>

<div class="flex flex-col gap-3">
	<div
		class="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border-light bg-bg-elevated px-3 py-2"
	>
		<div role="group" aria-label="Device size" class="inline-flex items-center gap-0.5">
			{#each DEVICES as d (d.id)}
				{@const Icon = d.icon}
				<button
					type="button"
					aria-pressed={device === d.id}
					title={`${d.label} (${d.width}px)`}
					onclick={() => (device = d.id)}
					class="inline-flex h-7 items-center gap-1.5 rounded px-2.5 text-[11.5px] font-medium transition-colors {device ===
					d.id
						? 'bg-bg-surface text-text-primary shadow-sm'
						: 'text-text-muted hover:text-text-primary'}"
				>
					<Icon class="h-3.5 w-3.5" />
					{d.label}
				</button>
			{/each}
		</div>
		<div class="flex items-center gap-2">
			<span class="text-[11px] text-text-muted">
				{#if loading}
					rendering…
				{:else if err}
					failed
				{:else if lastFetchedLabel}
					updated {lastFetchedLabel}
				{/if}
			</span>
			<Button variant="ghost" size="sm" onclick={() => load(true)} disabled={loading}>
				<RefreshCw class="h-3.5 w-3.5 {loading ? 'animate-spin' : ''}" />
				Refresh
			</Button>
			<Button variant="ghost" size="sm" onclick={openInNewTab}>
				<ExternalLink class="h-3.5 w-3.5" />
				Open
			</Button>
		</div>
	</div>

	{#if err}
		<div
			class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-[12px] text-red-800"
			role="alert"
		>
			Preview failed: {err}
		</div>
	{/if}

	<div
		class="flex justify-center overflow-hidden rounded-md border border-border-light bg-[#fafaf9]"
		style="min-height: 60vh;"
	>
		<iframe
			title="Page preview"
			srcdoc={html}
			class="block bg-white shadow-sm transition-[width] duration-200"
			style="width: {Math.min(currentDevice?.width ?? 1280, 1280)}px; height: 80vh; max-width: 100%;"
			sandbox="allow-same-origin allow-scripts allow-forms allow-popups"
			loading="eager"
			referrerpolicy="no-referrer"
		></iframe>
	</div>
</div>
