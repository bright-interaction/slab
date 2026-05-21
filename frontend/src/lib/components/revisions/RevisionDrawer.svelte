<script lang="ts">
	import { onMount } from 'svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as revisionsApi from '$lib/api/revisions';
	import { ApiError } from '$lib/api/client';
	import type { Revision, EntityType } from '$lib/api/revisions';

	let {
		open = $bindable(false),
		siteID,
		entityType,
		entityID,
		entityLabel = '',
		onRestored
	}: {
		open?: boolean;
		siteID: string;
		entityType: EntityType;
		entityID: string;
		entityLabel?: string;
		onRestored?: (version: number) => void;
	} = $props();

	let rows = $state<Revision[]>([]);
	let loading = $state(false);
	let error = $state('');
	let restoring = $state<number | null>(null);
	let confirmOpen = $state(false);
	let pickedVersion = $state<number | null>(null);

	$effect(() => {
		if (open) {
			void load();
		}
	});

	async function load() {
		loading = true;
		error = '';
		try {
			rows = await revisionsApi.listRevisions(siteID, entityType, entityID, 50);
		} catch (err) {
			rows = [];
			error = err instanceof ApiError ? err.message : 'Failed to load revisions';
		} finally {
			loading = false;
		}
	}

	function openRestoreConfirm(version: number) {
		pickedVersion = version;
		confirmOpen = true;
	}

	async function doRestore() {
		if (pickedVersion == null) return;
		const version = pickedVersion;
		restoring = version;
		try {
			await revisionsApi.restoreRevision(siteID, entityType, entityID, version);
			toast.success(`Restored to v${version}. Current state was snapshotted; undo the restore by picking the new latest revision.`);
			onRestored?.(version);
			await load();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Restore failed');
		} finally {
			restoring = null;
			pickedVersion = null;
		}
	}

	function shortActor(s: string): string {
		if (!s) return 'system';
		if (s.startsWith('user:')) return 'you / teammate';
		if (s.startsWith('agent:')) return 'agent ' + s.slice(6, 14);
		return s;
	}
</script>

<Dialog
	bind:open
	size="lg"
	title="Version history"
	description={entityLabel
		? `${entityLabel} (${entityType})`
		: `${entityType} history`}
>
	{#if loading}
		<div class="space-y-2">
			<Skeleton class="h-16" />
			<Skeleton class="h-16" />
			<Skeleton class="h-16" />
		</div>
	{:else if error}
		<div class="rounded-lg border border-danger/30 bg-danger/5 p-3 text-xs text-danger">
			{error}
		</div>
	{:else if rows.length === 0}
		<EmptyState
			title="No revisions yet"
			description={`The first edit to this ${entityType} writes a snapshot here. Restoring any prior revision creates a new revision representing the restored state, so history is always append-only.`}
		/>
	{:else}
		<ul class="divide-y divide-border-light">
			{#each rows as row, i (row.id)}
				<li class="py-3">
					<div class="flex items-start justify-between gap-4">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2">
								<Badge variant={i === 0 ? 'success' : 'default'}>
									v{row.version_number}{i === 0 ? ' (current)' : ''}
								</Badge>
								<span class="text-[11px] text-text-muted">{row.created_at}</span>
								<span class="text-[11px] font-mono text-text-muted truncate max-w-[20ch]">
									{shortActor(row.created_by)}
								</span>
							</div>
							<div class="mt-1 text-sm text-text-primary">
								{row.change_summary || '(no summary)'}
							</div>
						</div>
						{#if i !== 0}
							<button
								type="button"
								class="text-accent text-xs hover:underline shrink-0"
								onclick={() => openRestoreConfirm(row.version_number)}
								disabled={restoring === row.version_number}
							>
								{restoring === row.version_number ? 'Restoring…' : 'Restore'}
							</button>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end">
			<Button variant="ghost" onclick={() => (open = false)}>Close</Button>
		</div>
	{/snippet}
</Dialog>

<ConfirmDialog
	bind:open={confirmOpen}
	title="Restore to v{pickedVersion}?"
	message={`The current state will be snapshotted first (so this restore is itself undoable), then the ${entityType} will be reverted to v${pickedVersion}. The change is non-destructive: history stays append-only.`}
	confirmText="Restore"
	variant="primary"
	onconfirm={doRestore}
/>
