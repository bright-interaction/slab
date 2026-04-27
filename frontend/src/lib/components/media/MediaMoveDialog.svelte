<script lang="ts">
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { FolderHeart, Folder, Inbox } from 'lucide-svelte';
	import type { MediaFolder, Medium } from '$lib/api/types';

	let {
		open = $bindable(false),
		media,
		folders,
		busy = false,
		onMove
	}: {
		open?: boolean;
		media: Medium | null;
		folders: MediaFolder[];
		busy?: boolean;
		onMove: (folder: string) => Promise<void>;
	} = $props();

	const currentFolder = $derived(media?.folder ?? '');

	async function pick(name: string) {
		if (busy) return;
		await onMove(name);
	}

	const systemFolders = $derived(folders.filter((f) => f.is_system));
	const userFolders = $derived(folders.filter((f) => !f.is_system));

	function rowClass(active: boolean): string {
		return active
			? 'flex w-full items-center gap-2 rounded-md border border-border bg-bg-surface px-3 py-2 text-[13px] font-medium text-text-primary'
			: 'flex w-full items-center gap-2 rounded-md border border-transparent px-3 py-2 text-[13px] text-text-secondary hover:border-border-light hover:bg-bg-elevated hover:text-text-primary';
	}
</script>

<Dialog bind:open title="Move to folder" size="sm">
	{#if media}
		<p class="text-[12px] text-text-muted">
			{media.alt_text || media.filename}
		</p>
		<div class="mt-4 flex flex-col gap-1">
			<button
				type="button"
				class={rowClass(currentFolder === '')}
				onclick={() => pick('')}
				disabled={busy}
			>
				<Inbox class="h-4 w-4" strokeWidth={1.75} />
				<span class="flex-1 text-left">Unfiled</span>
				{#if currentFolder === ''}
					<span class="text-[11px] text-text-muted">current</span>
				{/if}
			</button>
			{#each systemFolders as f (f.name)}
				<button
					type="button"
					class={rowClass(currentFolder === f.name)}
					onclick={() => pick(f.name)}
					disabled={busy}
				>
					<FolderHeart class="h-4 w-4 text-accent" strokeWidth={1.75} />
					<span class="flex-1 text-left">
						{f.name === 'brand' ? 'Brand assets' : f.name}
					</span>
					{#if currentFolder === f.name}
						<span class="text-[11px] text-text-muted">current</span>
					{/if}
				</button>
			{/each}
			{#each userFolders as f (f.name)}
				<button
					type="button"
					class={rowClass(currentFolder === f.name)}
					onclick={() => pick(f.name)}
					disabled={busy}
				>
					<Folder class="h-4 w-4" strokeWidth={1.75} />
					<span class="flex-1 text-left">{f.name}</span>
					{#if currentFolder === f.name}
						<span class="text-[11px] text-text-muted">current</span>
					{/if}
				</button>
			{/each}
		</div>
	{/if}

	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>Close</Button>
	{/snippet}
</Dialog>
