<script lang="ts">
	import * as mediaApi from '$lib/api/media';
	import { toast } from '$lib/stores/toast.svelte';
	import { Folder, FolderHeart, Layers, Inbox, Plus, Trash2 } from 'lucide-svelte';
	import type { MediaFolder } from '$lib/api/types';

	const FOLDER_ALL = '';
	const FOLDER_UNFILED = '__unfiled';

	let {
		siteID,
		selected = $bindable(FOLDER_ALL),
		folders,
		totalCount,
		unfiledCount,
		onChanged
	}: {
		siteID: string;
		selected?: string;
		folders: MediaFolder[];
		totalCount: number;
		unfiledCount: number;
		onChanged?: () => void;
	} = $props();

	let creating = $state(false);
	let newFolder = $state('');
	let createError = $state<string | null>(null);

	const systemFolders = $derived(folders.filter((f) => f.is_system));
	const userFolders = $derived(folders.filter((f) => !f.is_system));

	function pick(name: string) {
		selected = name;
	}

	async function submitCreate() {
		const name = newFolder.trim().toLowerCase();
		if (!name) return;
		creating = true;
		createError = null;
		try {
			await mediaApi.createFolder(siteID, name);
			newFolder = '';
			onChanged?.();
			selected = name;
		} catch (err) {
			createError = err instanceof Error ? err.message : 'Failed to create folder';
		} finally {
			creating = false;
		}
	}

	async function removeFolder(f: MediaFolder, e: MouseEvent) {
		e.stopPropagation();
		if (f.is_system) return;
		const confirmed = confirm(
			`Delete folder "${f.name}"? Items inside will be moved back to All media.`
		);
		if (!confirmed) return;
		try {
			await mediaApi.deleteFolder(siteID, f.name);
			toast.success(`Folder ${f.name} removed`);
			if (selected === f.name) selected = FOLDER_ALL;
			onChanged?.();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to remove folder');
		}
	}

	function rowClass(active: boolean): string {
		return active
			? 'flex w-full items-center gap-2 rounded-md bg-bg-surface px-2.5 py-1.5 text-[12.5px] font-medium text-text-primary shadow-sm'
			: 'flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-[12.5px] text-text-secondary hover:bg-bg-elevated hover:text-text-primary';
	}
</script>

<nav class="flex flex-col gap-5" aria-label="Media folders">
	<div class="flex flex-col gap-0.5">
		<button type="button" class={rowClass(selected === FOLDER_ALL)} onclick={() => pick(FOLDER_ALL)}>
			<Layers class="h-3.5 w-3.5" strokeWidth={1.75} />
			<span class="flex-1 text-left">All media</span>
			<span class="text-[11px] text-text-muted">{totalCount}</span>
		</button>
		<button
			type="button"
			class={rowClass(selected === FOLDER_UNFILED)}
			onclick={() => pick(FOLDER_UNFILED)}
		>
			<Inbox class="h-3.5 w-3.5" strokeWidth={1.75} />
			<span class="flex-1 text-left">Unfiled</span>
			<span class="text-[11px] text-text-muted">{unfiledCount}</span>
		</button>
	</div>

	{#if systemFolders.length > 0}
		<div class="flex flex-col gap-1.5">
			<h3 class="px-2.5 text-[10px] font-mono uppercase tracking-[0.2em] text-text-muted">
				System
			</h3>
			<div class="flex flex-col gap-0.5">
				{#each systemFolders as f (f.name)}
					<button
						type="button"
						class={rowClass(selected === f.name)}
						onclick={() => pick(f.name)}
						title="Brand assets: favicon, OG image, logos. Always present."
					>
						<FolderHeart class="h-3.5 w-3.5 text-accent" strokeWidth={1.75} />
						<span class="flex-1 text-left capitalize">
							{f.name === 'brand' ? 'Brand assets' : f.name}
						</span>
						<span class="text-[11px] text-text-muted">{f.item_count}</span>
					</button>
				{/each}
			</div>
		</div>
	{/if}

	<div class="flex flex-col gap-1.5">
		<h3 class="px-2.5 text-[10px] font-mono uppercase tracking-[0.2em] text-text-muted">Folders</h3>
		<div class="flex flex-col gap-0.5">
			{#each userFolders as f (f.name)}
				<div class="group relative">
					<button
						type="button"
						class={rowClass(selected === f.name) + ' pr-8'}
						onclick={() => pick(f.name)}
					>
						<Folder class="h-3.5 w-3.5" strokeWidth={1.75} />
						<span class="flex-1 truncate text-left">{f.name}</span>
						<span class="text-[11px] text-text-muted">{f.item_count}</span>
					</button>
					<button
						type="button"
						aria-label="Delete folder {f.name}"
						class="absolute right-1.5 top-1/2 inline-flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded text-text-muted opacity-0 transition-opacity hover:bg-danger/10 hover:text-danger group-hover:opacity-100"
						onclick={(e) => removeFolder(f, e)}
					>
						<Trash2 class="h-3 w-3" strokeWidth={1.75} />
					</button>
				</div>
			{/each}
			{#if userFolders.length === 0}
				<p class="px-2.5 text-[11px] text-text-muted">No custom folders yet.</p>
			{/if}
		</div>

		<form
			class="mt-1 flex items-center gap-1.5 px-2.5"
			onsubmit={(e) => {
				e.preventDefault();
				void submitCreate();
			}}
		>
			<input
				type="text"
				placeholder="new-folder"
				bind:value={newFolder}
				class="h-7 w-full min-w-0 rounded-md border border-border-light bg-bg-elevated px-2 text-[12px] text-text-primary placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
				maxlength={32}
				disabled={creating}
				pattern="[a-z0-9][a-z0-9-]{'{'}0,31{'}'}"
				title="Lowercase letters, digits, and hyphens. Up to 32 chars."
			/>
			<button
				type="submit"
				aria-label="Create folder"
				class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-border-light bg-bg-elevated text-text-secondary hover:bg-bg-hover hover:text-text-primary disabled:opacity-50"
				disabled={creating || !newFolder.trim()}
			>
				<Plus class="h-3.5 w-3.5" strokeWidth={2} />
			</button>
		</form>
		{#if createError}
			<p class="px-2.5 text-[11px] text-danger">{createError}</p>
		{/if}
	</div>
</nav>
