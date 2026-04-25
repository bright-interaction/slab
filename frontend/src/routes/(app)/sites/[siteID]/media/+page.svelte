<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import * as mediaApi from '$lib/api/media';
	import Button from '$lib/components/ui/Button.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import MediaGrid from '$lib/components/media/MediaGrid.svelte';
	import MediaUploader from '$lib/components/media/MediaUploader.svelte';
	import MediaDetailDialog from '$lib/components/media/MediaDetailDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Medium, Site } from '$lib/api/types';

	const PAGE_SIZE = 24;

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let items = $state<Medium[]>([]);
	let total = $state(0);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let detailOpen = $state(false);
	let selected = $state<Medium | null>(null);

	let dragDepth = $state(0);
	const isDragging = $derived(dragDepth > 0);

	let uploader: MediaUploader;

	const offset = $derived.by(() => {
		const v = parseInt(page.url.searchParams.get('offset') || '0', 10);
		return Number.isFinite(v) && v >= 0 ? v : 0;
	});

	async function load(off: number) {
		loading = true;
		loadError = null;
		try {
			const res = await mediaApi.list(siteID, { limit: PAGE_SIZE, offset: off });
			items = res.items;
			total = res.total;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load media';
			toast.error(loadError);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load(offset);
	});

	function gotoOffset(off: number) {
		const params = new URLSearchParams(page.url.searchParams);
		if (off === 0) {
			params.delete('offset');
		} else {
			params.set('offset', String(off));
		}
		const qs = params.toString();
		goto(qs ? `?${qs}` : page.url.pathname, { keepFocus: true, noScroll: true });
	}

	function handleUploaded(m: Medium) {
		items = [m, ...items];
		total = total + 1;
	}

	function openDetail(m: Medium) {
		selected = m;
		detailOpen = true;
	}

	function handleUpdated(updated: Medium) {
		items = items.map((it) => (it.id === updated.id ? updated : it));
		if (selected && selected.id === updated.id) selected = updated;
	}

	function handleDeletedFromDialog(m: Medium) {
		items = items.filter((it) => it.id !== m.id);
		total = Math.max(0, total - 1);
	}

	async function handleQuickDelete(m: Medium) {
		const prev = items;
		const prevTotal = total;
		items = items.filter((it) => it.id !== m.id);
		total = Math.max(0, total - 1);
		try {
			await mediaApi.remove(siteID, m.id);
			toast.success('Media deleted');
		} catch (err) {
			items = prev;
			total = prevTotal;
			toast.error(err instanceof Error ? err.message : 'Failed to delete');
		}
	}

	async function handleCopyUrl(m: Medium) {
		const path = mediaApi.mediaUrl(siteID, m.id, 'original');
		const full = typeof window !== 'undefined' ? `${window.location.origin}${path}` : path;
		try {
			await navigator.clipboard.writeText(full);
			toast.success('URL copied');
		} catch {
			toast.error('Could not copy');
		}
	}

	function handleEditAlt(m: Medium) {
		selected = m;
		detailOpen = true;
	}

	function onDragEnter(e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer && Array.from(e.dataTransfer.types).includes('Files')) {
			dragDepth = dragDepth + 1;
		}
	}
	function onDragOver(e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
	}
	function onDragLeave(e: DragEvent) {
		e.preventDefault();
		dragDepth = Math.max(0, dragDepth - 1);
	}
	async function onDrop(e: DragEvent) {
		e.preventDefault();
		dragDepth = 0;
		if (!e.dataTransfer) return;
		const files = Array.from(e.dataTransfer.files).filter((f) => f.type.startsWith('image/'));
		if (files.length === 0) {
			toast.error('Only image files are accepted');
			return;
		}
		await uploader.uploadFiles(files);
	}

	const showingFrom = $derived(total === 0 ? 0 : offset + 1);
	const showingTo = $derived(Math.min(offset + items.length, total));
	const canPrev = $derived(offset > 0);
	const canNext = $derived(offset + items.length < total);
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Media</h1>
			<p class="mt-1 text-[13px] text-text-secondary">Images served at the edge with auto variants.</p>
		</div>
		<Button variant="primary" onclick={() => uploader.open()}>Upload</Button>
	</div>

	<MediaUploader bind:this={uploader} {siteID} onUploaded={handleUploaded} />

	<div
		role="region"
		aria-label="Media library, drop files to upload"
		ondragenter={onDragEnter}
		ondragover={onDragOver}
		ondragleave={onDragLeave}
		ondrop={onDrop}
		class="relative mt-6 rounded-2xl {isDragging
			? 'border-2 border-dashed border-accent bg-bg-surface/50'
			: 'border-2 border-dashed border-transparent'}"
	>
		{#if isDragging}
			<div
				class="pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-2xl bg-bg-surface/70 backdrop-blur-sm"
			>
				<p class="font-display text-xl font-extralight tracking-tight text-text-primary">
					Drop to upload
				</p>
			</div>
		{/if}

		<div class="p-1">
			{#if loading}
				<div class="flex items-center justify-center py-24">
					<Spinner size="lg" />
				</div>
			{:else if loadError}
				<Card padding="md">
					<p class="text-[13px] text-danger">{loadError}</p>
				</Card>
			{:else if items.length === 0}
				<Card padding="none">
					<EmptyState
						title="No media yet"
						description="Drop image files here or click Upload to add your first asset."
					>
						{#snippet action()}
							<Button variant="primary" onclick={() => uploader.open()}>Upload</Button>
						{/snippet}
					</EmptyState>
				</Card>
			{:else}
				<MediaGrid
					media={items}
					{siteID}
					onOpen={openDetail}
					onEdit={handleEditAlt}
					onCopyUrl={handleCopyUrl}
					onDelete={handleQuickDelete}
				/>
			{/if}
		</div>
	</div>

	{#if !loading && total > 0}
		<div class="mt-6 flex items-center justify-between">
			<p class="text-[12px] text-text-muted">
				Showing {showingFrom} to {showingTo} of {total}
			</p>
			<div class="flex items-center gap-2">
				<Button
					variant="secondary"
					size="sm"
					disabled={!canPrev}
					onclick={() => gotoOffset(Math.max(0, offset - PAGE_SIZE))}
				>
					Previous
				</Button>
				<Button
					variant="secondary"
					size="sm"
					disabled={!canNext}
					onclick={() => gotoOffset(offset + PAGE_SIZE)}
				>
					Next
				</Button>
			</div>
		</div>
	{/if}
</div>

<MediaDetailDialog
	bind:open={detailOpen}
	{siteID}
	media={selected}
	onUpdated={handleUpdated}
	onDeleted={handleDeletedFromDialog}
/>
