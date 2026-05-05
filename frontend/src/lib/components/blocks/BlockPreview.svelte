<script lang="ts">
	import { onMount } from 'svelte';
	import * as blocksApi from '$lib/api/blocks';

	let {
		siteID,
		pageID,
		blockID,
		blockType,
		dataJson,
		minHeight = 0,
		maxHeight = 1600,
		hideChrome = false
	}: {
		siteID: string;
		pageID: string;
		blockID: string;
		blockType: string;
		dataJson: string;
		minHeight?: number;
		maxHeight?: number;
		hideChrome?: boolean;
	} = $props();

	let iframeEl: HTMLIFrameElement | null = $state(null);
	let height = $state(Math.max(160, minHeight));
	let loading = $state(true);
	let error = $state<string | null>(null);
	let ready = $state(false);

	let timer: ReturnType<typeof setTimeout> | null = null;
	let abort: AbortController | null = null;

	function buildDoc(html: string, css: string, fontFace: string): string {
		const fontStyle = fontFace ? `<style>${fontFace}</style>` : '';
		return `<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<style>${css}</style>
${fontStyle}
<style>
html, body { padding: 0; margin: 0; background: var(--color-bg, #fff); }
main { display: block; }
</style>
</head>
<body>
<main id="main">${html}</main>
<script>
(function () {
  function report() {
    var h = document.body.scrollHeight || document.documentElement.scrollHeight || 0;
    parent.postMessage({ source: 'atomicsite-preview', height: h }, '*');
  }
  new ResizeObserver(report).observe(document.body);
  window.addEventListener('load', report);
  setTimeout(report, 50);
})();
${'<'}/script>
</body>
</html>`;
	}

	async function refresh(): Promise<void> {
		if (!blockID) {
			error = 'Save the block once before previewing.';
			loading = false;
			return;
		}
		abort?.abort();
		abort = new AbortController();
		loading = true;
		error = null;
		try {
			const data = parseDataJson(dataJson);
			const res = await blocksApi.previewHTML(siteID, pageID, blockID, {
				block_type: blockType,
				data
			});
			if (iframeEl) {
				iframeEl.srcdoc = buildDoc(res.html, res.css, res.font_face);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Preview failed';
		} finally {
			loading = false;
		}
	}

	function parseDataJson(s: string): unknown {
		try {
			return JSON.parse(s || '{}');
		} catch {
			return {};
		}
	}

	function schedule(): void {
		if (!ready) return;
		if (timer) clearTimeout(timer);
		timer = setTimeout(refresh, 300);
	}

	$effect(() => {
		// Re-fetch on input changes once we're mounted.
		void dataJson;
		void blockType;
		schedule();
	});

	onMount(() => {
		ready = true;
		void refresh();
		const onMsg = (ev: MessageEvent) => {
			const data = ev.data as { source?: string; height?: number } | null;
			if (data && data.source === 'atomicsite-preview' && typeof data.height === 'number') {
				const h = Math.max(Math.max(120, minHeight), Math.min(maxHeight, Math.ceil(data.height)));
				if (Math.abs(h - height) > 4) height = h;
			}
		};
		window.addEventListener('message', onMsg);
		return () => {
			window.removeEventListener('message', onMsg);
			if (timer) clearTimeout(timer);
			abort?.abort();
		};
	});
</script>

<div class="relative {hideChrome ? '' : 'rounded-lg border border-border-light bg-bg-surface'} overflow-hidden">
	{#if !hideChrome}
		<div class="flex items-center justify-between gap-2 border-b border-border-light bg-bg-elevated/40 px-3 py-1.5">
			<span class="text-[10.5px] font-mono uppercase tracking-[0.18em] text-text-muted">Live preview</span>
			{#if loading}
				<span class="text-[11px] text-text-muted">Rendering…</span>
			{/if}
		</div>
	{/if}
	{#if error}
		<p class="px-3 py-3 text-[12px] text-danger">{error}</p>
	{/if}
	<iframe
		bind:this={iframeEl}
		title="Block preview"
		sandbox="allow-same-origin"
		style="height: {height}px;"
		class="block w-full bg-white"
	></iframe>
	{#if !hideChrome}
		<p class="border-t border-border-light bg-bg-elevated/40 px-3 py-1.5 text-[10.5px] text-text-muted">
			Block-only preview. Header and footer render on the full page.
		</p>
	{/if}
</div>
