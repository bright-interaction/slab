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
	import MediaFolderSidebar from '$lib/components/media/MediaFolderSidebar.svelte';
	import MediaMoveDialog from '$lib/components/media/MediaMoveDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { MediaFolder, Medium, Site } from '$lib/api/types';

	const PAGE_SIZE = 24;
	const FOLDER_ALL = '';
	const FOLDER_UNFILED = '__unfiled';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let items = $state<Medium[]>([]);
	let total = $state(0);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let folders = $state<MediaFolder[]>([]);
	let totalCount = $state(0);
	let unfiledCount = $state(0);

	let detailOpen = $state(false);
	let selected = $state<Medium | null>(null);

	let moveOpen = $state(false);
	let moveTarget = $state<Medium | null>(null);
	let moveBusy = $state(false);

	let dragDepth = $state(0);
	const isDragging = $derived(dragDepth > 0);

	let uploader: MediaUploader;

	const offset = $derived.by(() => {
		const v = parseInt(page.url.searchParams.get('offset') || '0', 10);
		return Number.isFinite(v) && v >= 0 ? v : 0;
	});

	const folderParam = $derived(page.url.searchParams.get('folder') ?? FOLDER_ALL);

	const folderLabel = $derived.by(() => {
		if (folderParam === FOLDER_ALL) return 'All media';
		if (folderParam === FOLDER_UNFILED) return 'Unfiled';
		if (folderParam === 'brand') return 'Brand assets';
		return folderParam;
	});

	const folderHelper = $derived.by(() => {
		if (folderParam === FOLDER_ALL) return 'Every image you have uploaded.';
		if (folderParam === FOLDER_UNFILED) return 'Images that have not been filed into a folder yet.';
		if (folderParam === 'brand')
			return 'Favicon, OG image, logos, and other brand assets that ship with every build.';
		return `Items in the ${folderParam} folder.`;
	});

	// Upload destination: if viewing a real folder, default uploads to it.
	const uploadFolder = $derived(
		folderParam === FOLDER_ALL || folderParam === FOLDER_UNFILED ? '' : folderParam
	);

	async function loadItems(off: number, folder: string) {
		loading = true;
		loadError = null;
		try {
			const res = await mediaApi.list(siteID, { limit: PAGE_SIZE, offset: off, folder });
			items = res.items;
			total = res.total;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load media';
			toast.error(loadError);
		} finally {
			loading = false;
		}
	}

	async function loadFolders() {
		try {
			const res = await mediaApi.listFolders(siteID);
			folders = res.folders;
			totalCount = res.total_count;
			unfiledCount = res.unfiled_count;
		} catch {
			folders = [];
		}
	}

	$effect(() => {
		void loadItems(offset, folderParam);
	});

	$effect(() => {
		void loadFolders();
	});

	function setFolder(name: string) {
		const params = new URLSearchParams(page.url.searchParams);
		if (name === FOLDER_ALL) params.delete('folder');
		else params.set('folder', name);
		params.delete('offset');
		const qs = params.toString();
		goto(qs ? `?${qs}` : page.url.pathname, { keepFocus: true, noScroll: true });
	}

	function gotoOffset(off: number) {
		const params = new URLSearchParams(page.url.searchParams);
		if (off === 0) params.delete('offset');
		else params.set('offset', String(off));
		const qs = params.toString();
		goto(qs ? `?${qs}` : page.url.pathname, { keepFocus: true, noScroll: true });
	}

	function handleUploaded(m: Medium) {
		// Only prepend to the visible grid if it belongs to the current view.
		const visibleHere =
			folderParam === FOLDER_ALL ||
			(folderParam === FOLDER_UNFILED && (!m.folder || m.folder === '')) ||
			folderParam === m.folder;
		if (visibleHere) {
			items = [m, ...items];
			total = total + 1;
		}
		void loadFolders();
	}

	function openDetail(m: Medium) {
		selected = m;
		detailOpen = true;
	}

	function handleUpdated(updated: Medium) {
		items = items.map((it) => (it.id === updated.id ? updated : it));
		if (selected && selected.id === updated.id) selected = updated;
		void loadFolders();
	}

	function handleDeletedFromDialog(m: Medium) {
		items = items.filter((it) => it.id !== m.id);
		total = Math.max(0, total - 1);
		void loadFolders();
	}

	async function handleQuickDelete(m: Medium) {
		const prev = items;
		const prevTotal = total;
		items = items.filter((it) => it.id !== m.id);
		total = Math.max(0, total - 1);
		try {
			await mediaApi.remove(siteID, m.id);
			toast.success('Media deleted');
			void loadFolders();
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

	function handleMove(m: Medium) {
		moveTarget = m;
		moveOpen = true;
	}

	async function applyMove(folder: string) {
		if (!moveTarget) return;
		moveBusy = true;
		try {
			const updated = await mediaApi.update(siteID, moveTarget.id, { folder });
			toast.success(folder ? `Moved to ${folder}` : 'Moved to Unfiled');
			// If the new folder doesn't match the current view, drop it from the grid.
			const stillVisible =
				folderParam === FOLDER_ALL ||
				(folderParam === FOLDER_UNFILED && (!updated.folder || updated.folder === '')) ||
				folderParam === updated.folder;
			if (stillVisible) {
				items = items.map((it) => (it.id === updated.id ? updated : it));
			} else {
				items = items.filter((it) => it.id !== updated.id);
				total = Math.max(0, total - 1);
			}
			void loadFolders();
			moveOpen = false;
			moveTarget = null;
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to move');
		} finally {
			moveBusy = false;
		}
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

	const showFolderBadge = $derived(folderParam === FOLDER_ALL);
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-wrap items-start justify-between gap-4">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Media</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Images served at the edge with auto variants.
			</p>
		</div>
		<Button variant="primary" onclick={() => uploader.open()}>
			Upload{uploadFolder ? ` to ${uploadFolder}` : ''}
		</Button>
	</header>

	<MediaUploader bind:this={uploader} {siteID} folder={uploadFolder} onUploaded={handleUploaded} />

	<div class="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-[14rem_1fr]">
		<aside class="lg:sticky lg:top-24 lg:self-start">
			<MediaFolderSidebar
				{siteID}
				selected={folderParam}
				{folders}
				{totalCount}
				{unfiledCount}
				onChanged={loadFolders}
			/>
		</aside>

		<div>
			<div class="mb-3 flex items-baseline justify-between">
				<div>
					<h2 class="text-[13px] font-medium text-text-primary">{folderLabel}</h2>
					<p class="mt-0.5 text-[11.5px] text-text-muted">{folderHelper}</p>
				</div>
				<span class="font-mono text-[11px] uppercase tracking-[0.12em] text-text-muted">
					{total}
					{total === 1 ? 'item' : 'items'}
				</span>
			</div>

			<div
				role="region"
				aria-label="Media library, drop files to upload"
				ondragenter={onDragEnter}
				ondragover={onDragOver}
				ondragleave={onDragLeave}
				ondrop={onDrop}
				class="relative rounded-2xl {isDragging
					? 'border-2 border-dashed border-accent bg-bg-surface/50'
					: 'border-2 border-dashed border-transparent'}"
			>
				{#if isDragging}
					<div
						class="pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-2xl bg-bg-surface/70 backdrop-blur-sm"
					>
						<p class="font-display text-xl font-extralight tracking-tight text-text-primary">
							{uploadFolder ? `Drop to upload to ${uploadFolder}` : 'Drop to upload'}
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
								title={folderParam === FOLDER_ALL
									? 'No media yet'
									: `Nothing in ${folderLabel} yet`}
								description="Drop image files here or click Upload to add an asset."
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
							showFolder={showFolderBadge}
							onOpen={openDetail}
							onEdit={handleEditAlt}
							onMove={handleMove}
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
	</div>
</div>

<MediaDetailDialog
	bind:open={detailOpen}
	{siteID}
	media={selected}
	onUpdated={handleUpdated}
	onDeleted={handleDeletedFromDialog}
/>

<MediaMoveDialog
	bind:open={moveOpen}
	media={moveTarget}
	{folders}
	busy={moveBusy}
	onMove={applyMove}
/>
