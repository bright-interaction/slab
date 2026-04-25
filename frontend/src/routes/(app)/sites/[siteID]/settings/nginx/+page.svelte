<script lang="ts">
	import * as settingsApi from '$lib/api/settings';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { buildNginxPreview, categoryMap } from '$lib/settings/nginxPreview';
	import type { Site } from '$lib/api/types';
	import { Copy, Download, RefreshCw } from 'lucide-svelte';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let loading = $state(true);
	let preview = $state('');

	async function load() {
		loading = true;
		try {
			const [security, server] = await Promise.all([
				settingsApi.listByCategory(siteID, 'security'),
				settingsApi.listByCategory(siteID, 'server')
			]);
			preview = buildNginxPreview({
				domain: data.site.domain,
				security: categoryMap(security),
				server: categoryMap(server)
			});
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load preview.');
			preview = '';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	async function copyToClipboard() {
		if (!preview) return;
		try {
			await navigator.clipboard.writeText(preview);
			toast.success('Copied to clipboard.');
		} catch {
			toast.error('Copy failed. Select and copy manually.');
		}
	}

	function download() {
		if (!preview) return;
		const blob = new Blob([preview], { type: 'text/plain' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		const safeName = (data.site.domain || data.site.slug || 'site').replace(/[^a-z0-9.-]/gi, '_');
		a.download = `${safeName}.nginx.conf`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}
</script>

<div class="mx-auto max-w-4xl px-6 py-8">
	<header class="flex items-start justify-between gap-4">
		<div class="flex flex-col gap-1.5">
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Nginx preview
			</h1>
			<p class="text-[13px] text-text-secondary">
				Read-only preview of the server config rendered from current settings.
			</p>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="ghost" onclick={() => load()} disabled={loading}>
				{#snippet icon()}
					<RefreshCw size={14} strokeWidth={1.75} />
				{/snippet}
				Refresh
			</Button>
			<Button variant="secondary" onclick={copyToClipboard} disabled={!preview || loading}>
				{#snippet icon()}
					<Copy size={14} strokeWidth={1.75} />
				{/snippet}
				Copy
			</Button>
			<Button variant="primary" onclick={download} disabled={!preview || loading}>
				{#snippet icon()}
					<Download size={14} strokeWidth={1.75} />
				{/snippet}
				Download
			</Button>
		</div>
	</header>

	<div class="mt-6">
		<Card padding="none">
			{#if loading}
				<div class="flex items-center justify-center py-12">
					<Spinner />
				</div>
			{:else}
				<pre
					class="overflow-auto rounded-xl bg-bg-elevated p-4 font-mono text-xs whitespace-pre text-text-primary max-h-[70vh]"
				>{preview || '# No preview available.'}</pre>
			{/if}
		</Card>
	</div>

	<p class="mt-3 text-[12px] text-text-muted">
		Preview is generated client-side. The actual config applied at deploy may differ slightly
		based on platform-specific overrides.
	</p>
</div>
