<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';
	import { ExternalLink } from 'lucide-svelte';

	interface Props {
		siteID: string;
		pageID: string;
		// Kept for prop-compat with callers; ignored now that this panel
		// no longer renders an iframe (the new-tab button always fetches fresh).
		refreshNonce?: number;
	}

	let { siteID, pageID }: Props = $props();

	const previewURL = $derived(
		`/api/sites/${encodeURIComponent(siteID)}/pages/${encodeURIComponent(pageID)}/draft-preview`
	);

	function openInNewTab() {
		window.open(previewURL, '_blank', 'noopener,noreferrer');
	}
</script>

<div class="flex flex-col gap-3">
	<div
		class="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border-light bg-bg-elevated px-3 py-2"
	>
		<div class="text-[12px] text-text-muted">
			Preview opens in a new tab. The iframe preview was removed because rendering it inline froze
			the editor on heavy pages.
		</div>
		<Button variant="primary" size="sm" onclick={openInNewTab}>
			<ExternalLink class="h-3.5 w-3.5" />
			Open preview
		</Button>
	</div>
</div>
