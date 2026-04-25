<script lang="ts">
	import { goto } from '$app/navigation';
	import { page as pageState } from '$app/state';
	import * as pagesApi from '$lib/api/pages';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import PageRow from '$lib/components/pages/PageRow.svelte';
	import type { Page } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);

	let pages = $state<Page[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let draggingId = $state<string | null>(null);

	let newDialogOpen = $state(false);
	let creating = $state(false);
	let newTitle = $state('');
	let newSlug = $state('');
	let newLayout = $state('default');
	let newSlugTouched = $state(false);

	const layoutOptions = [
		{ value: 'default', label: 'Default' },
		{ value: 'landing', label: 'Landing' },
		{ value: 'legal', label: 'Legal' },
		{ value: 'minimal', label: 'Minimal' }
	];

	function slugify(input: string): string {
		return input
			.toLowerCase()
			.normalize('NFKD')
			.replace(/[̀-ͯ]/g, '')
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 60);
	}

	$effect(() => {
		if (!newSlugTouched) {
			newSlug = slugify(newTitle);
		}
	});

	async function loadPages(id: string) {
		loading = true;
		loadError = null;
		try {
			const res = await pagesApi.list(id);
			pages = [...(res.pages ?? [])].sort((a, b) => a.sort_order - b.sort_order);
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load pages';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void loadPages(siteID);
	});

	function openNewDialog() {
		newTitle = '';
		newSlug = '';
		newLayout = 'default';
		newSlugTouched = false;
		newDialogOpen = true;
	}

	async function createPage() {
		const title = newTitle.trim();
		const slug = newSlug.trim();
		if (title.length < 2) {
			toast.error('Title must be at least 2 characters.');
			return;
		}
		if (!/^[a-z0-9-]+$/.test(slug)) {
			toast.error('Slug must use lowercase letters, numbers, and hyphens.');
			return;
		}
		creating = true;
		try {
			const created = await pagesApi.create(siteID, {
				title,
				slug,
				layout: newLayout
			});
			newDialogOpen = false;
			toast.success('Page created.');
			await goto(`/sites/${siteID}/pages/${created.id}`);
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Failed to create page';
			toast.error(msg);
		} finally {
			creating = false;
		}
	}

	function openPage(p: Page) {
		void goto(`/sites/${siteID}/pages/${p.id}`);
	}

	async function deletePage(p: Page) {
		const ok = await confirm({
			title: `Delete "${p.title || p.slug}"?`,
			message: 'This removes the page and its blocks. This cannot be undone.',
			confirmText: 'Delete',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		const previous = pages;
		pages = pages.filter((x) => x.id !== p.id);
		try {
			await pagesApi.remove(siteID, p.id);
			toast.success('Page deleted.');
		} catch (err) {
			pages = previous;
			const msg = err instanceof Error ? err.message : 'Failed to delete page';
			toast.error(msg);
		}
	}

	function indexOf(id: string): number {
		return pages.findIndex((p) => p.id === id);
	}

	function handleDragStart(id: string, e: DragEvent) {
		draggingId = id;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', id);
		}
	}

	function handleDragOver(id: string, e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		if (!draggingId || draggingId === id) return;
		const fromIdx = indexOf(draggingId);
		const toIdx = indexOf(id);
		if (fromIdx === -1 || toIdx === -1 || fromIdx === toIdx) return;
		const next = [...pages];
		const moved = next.splice(fromIdx, 1)[0];
		if (!moved) return;
		next.splice(toIdx, 0, moved);
		pages = next.map((p, i) => ({ ...p, sort_order: i }));
	}

	async function handleDragEnd() {
		const movedId = draggingId;
		draggingId = null;
		if (!movedId) return;
		try {
			await pagesApi.reorder(
				siteID,
				pages.map((p) => p.id)
			);
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Failed to save order';
			toast.error(msg);
			void loadPages(siteID);
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex items-center justify-between">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Pages
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Drag rows to reorder. Click a page to open the editor.
			</p>
		</div>
		<Button variant="primary" onclick={openNewDialog}>New page</Button>
	</header>

	<section class="mt-6">
		{#if loading}
			<div class="flex flex-col gap-2">
				{#each Array(4) as _, i (i)}
					<Card padding="sm">
						<div class="flex items-center gap-3">
							<Skeleton width="40%" height="0.95rem" />
							<Skeleton width="20%" height="0.7rem" />
						</div>
					</Card>
				{/each}
			</div>
		{:else if loadError}
			<Card padding="md">
				<p class="text-[13px] text-danger">{loadError}</p>
			</Card>
		{:else if pages.length === 0}
			<Card padding="none">
				<EmptyState
					title="No pages yet"
					description="Pages are the top-level URLs of your site. Add a homepage, an about page, or a legal page."
				>
					{#snippet action()}
						<Button variant="primary" onclick={openNewDialog}>Create page</Button>
					{/snippet}
				</EmptyState>
			</Card>
		{:else}
			<div class="flex flex-col gap-2" role="list">
				{#each pages as p (p.id)}
					<PageRow
						page={p}
						dragging={draggingId === p.id}
						onOpen={() => openPage(p)}
						onDelete={() => deletePage(p)}
						ondragstart={(e) => handleDragStart(p.id, e)}
						ondragover={(e) => handleDragOver(p.id, e)}
						ondragend={handleDragEnd}
						ondrop={handleDrop}
					/>
				{/each}
			</div>
		{/if}
	</section>
</div>

<Dialog bind:open={newDialogOpen} title="New page" size="sm">
	<div class="flex flex-col gap-3">
		<Input
			label="Title"
			value={newTitle}
			oninput={(e) => {
				newTitle = (e.currentTarget as HTMLInputElement).value;
			}}
			placeholder="About us"
		/>
		<Input
			label="Slug"
			value={newSlug}
			oninput={(e) => {
				newSlug = (e.currentTarget as HTMLInputElement).value;
				newSlugTouched = true;
			}}
			placeholder="about"
			hint="Lowercase letters, numbers, hyphens. Used in the URL."
		/>
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Layout</span>
			<Select options={layoutOptions} bind:value={newLayout} />
		</div>
	</div>
	{#snippet footer()}
		<Button variant="secondary" onclick={() => (newDialogOpen = false)}>Cancel</Button>
		<Button variant="primary" loading={creating} onclick={createPage}>Create</Button>
	{/snippet}
</Dialog>
