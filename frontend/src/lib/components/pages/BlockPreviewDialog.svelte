<script lang="ts">
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import BlockPreview from '$lib/components/blocks/BlockPreview.svelte';

	let {
		open = $bindable(false),
		siteID,
		pageID,
		blockID,
		blockType,
		dataJson
	}: {
		open?: boolean;
		siteID: string;
		pageID: string;
		blockID: string;
		blockType: string;
		dataJson: string;
	} = $props();

	// We only want the iframe + fetch loop running while the modal is on
	// screen. Mount on open, unmount on close so closing the dialog is
	// equivalent to "stop previewing this block."
	let mounted = $state(false);
	$effect(() => {
		if (open) mounted = true;
		else mounted = false;
	});
</script>

<Dialog bind:open title="Live preview" description="Block-only render. Header and footer appear on the full page." size="lg">
	<div class="min-h-[200px]">
		{#if mounted}
			<BlockPreview {siteID} {pageID} {blockID} {blockType} {dataJson} />
		{/if}
	</div>
</Dialog>
