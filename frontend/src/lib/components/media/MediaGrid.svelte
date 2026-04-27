<script lang="ts">
	import MediaCard from './MediaCard.svelte';
	import type { Medium } from '$lib/api/types';

	let {
		media,
		siteID,
		showFolder = false,
		onOpen,
		onSelect,
		onEdit,
		onMove,
		onCopyUrl,
		onDelete
	}: {
		media: Medium[];
		siteID: string;
		showFolder?: boolean;
		onOpen?: (m: Medium) => void;
		onSelect?: (m: Medium) => void;
		onEdit?: (m: Medium) => void;
		onMove?: (m: Medium) => void;
		onCopyUrl?: (m: Medium) => void;
		onDelete?: (m: Medium) => void;
	} = $props();
</script>

<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
	{#each media as item (item.id)}
		<MediaCard
			media={item}
			{siteID}
			{showFolder}
			{onOpen}
			{onSelect}
			{onEdit}
			{onMove}
			{onCopyUrl}
			{onDelete}
		/>
	{/each}
</div>
