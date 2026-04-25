<script lang="ts">
	import * as mediaApi from '$lib/api/media';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import MediaGrid from './MediaGrid.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Medium } from '$lib/api/types';

	const PAGE_SIZE = 18;

	let {
		open = $bindable(false),
		siteID,
		onSelect
	}: {
		open?: boolean;
		siteID: string;
		onSelect: (m: Medium) => void;
	} = $props();

	let items = $state<Medium[]>([]);
	let total = $state(0);
	let offset = $state(0);
	let query = $state('');
	let loading = $state(false);
	let loadError = $state<string | null>(null);

	let searchTimer: ReturnType<typeof setTimeout> | null = null;

	async function load() {
		loading = true;
		loadError = null;
		try {
			const res = await mediaApi.list(siteID, {
				limit: PAGE_SIZE,
				offset,
				q: query || undefined
			});
			items = res.items;
			total = res.total;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load media';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (open) {
			void load();
		} else {
			items = [];
			offset = 0;
			query = '';
		}
	});

	function onQueryInput(e: Event) {
		query = (e.currentTarget as HTMLInputElement).value;
		offset = 0;
		if (searchTimer) clearTimeout(searchTimer);
		searchTimer = setTimeout(() => {
			void load();
		}, 200);
	}

	function pick(m: Medium) {
		onSelect(m);
		open = false;
	}

	const showingFrom = $derived(total === 0 ? 0 : offset + 1);
	const showingTo = $derived(Math.min(offset + items.length, total));
	const canPrev = $derived(offset > 0);
	const canNext = $derived(offset + items.length < total);

	function prev() {
		offset = Math.max(0, offset - PAGE_SIZE);
		void load();
	}
	function next() {
		offset = offset + PAGE_SIZE;
		void load();
	}

	$effect(() => {
		if (loadError) toast.error(loadError);
	});
</script>

<Dialog bind:open size="lg" title="Choose media">
	<div class="flex flex-col gap-4">
		<Input
			placeholder="Search by filename or alt text"
			value={query}
			oninput={onQueryInput}
		/>

		{#if loading}
			<div class="flex items-center justify-center py-12">
				<Spinner />
			</div>
		{:else if items.length === 0}
			<EmptyState
				title="No media found"
				description={query ? 'Try a different search.' : 'Upload media from the Media page first.'}
			/>
		{:else}
			<MediaGrid media={items} {siteID} onSelect={pick} />
		{/if}

		<div class="flex items-center justify-between border-t border-border-light pt-3">
			<p class="text-[12px] text-text-muted">
				{#if total > 0}
					Showing {showingFrom} to {showingTo} of {total}
				{:else}
					No items
				{/if}
			</p>
			<div class="flex items-center gap-2">
				<Button variant="secondary" size="sm" disabled={!canPrev} onclick={prev}>Previous</Button>
				<Button variant="secondary" size="sm" disabled={!canNext} onclick={next}>Next</Button>
			</div>
		</div>
	</div>
</Dialog>
