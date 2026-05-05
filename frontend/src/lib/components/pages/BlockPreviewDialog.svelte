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

<Dialog bind:open title="Live preview" description="Block-only render at desktop width. Header and footer render on the full page." size="xl">
	<div class="overflow-hidden rounded-lg border border-border-light bg-white">
		{#if mounted}
			<!-- 720px is a generous floor (covers 70vh hero on a 1024-tall window
			     with room for safe area), 4000px ceiling lets full process_steps
			     and pricing blocks render without internal iframe scroll. The
			     dialog body itself scrolls if total content exceeds viewport. -->
			<BlockPreview
				{siteID}
				{pageID}
				{blockID}
				{blockType}
				{dataJson}
				minHeight={720}
				maxHeight={4000}
				hideChrome
			/>
		{/if}
	</div>
</Dialog>
