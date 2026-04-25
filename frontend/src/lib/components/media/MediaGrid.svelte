<script lang="ts">
	import MediaCard from './MediaCard.svelte';
	import type { Medium } from '$lib/api/types';

	let {
		media,
		siteID,
		onOpen,
		onSelect,
		onEdit,
		onCopyUrl,
		onDelete
	}: {
		media: Medium[];
		siteID: string;
		onOpen?: (m: Medium) => void;
		onSelect?: (m: Medium) => void;
		onEdit?: (m: Medium) => void;
		onCopyUrl?: (m: Medium) => void;
		onDelete?: (m: Medium) => void;
	} = $props();
</script>

<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-3">
	{#each media as item (item.id)}
		<MediaCard
			media={item}
			{siteID}
			{onOpen}
			{onSelect}
			{onEdit}
			{onCopyUrl}
			{onDelete}
		/>
	{/each}
</div>
