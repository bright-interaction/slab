<script lang="ts">
	import * as deployApi from '$lib/api/deploy';
	import { ApiError } from '$lib/api/client';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import TargetForm from '$lib/components/deploy/TargetForm.svelte';
	import type { DeployTarget, Site } from '$lib/api/types';
	import { Plus, Trash2, Pencil, Star } from 'lucide-svelte';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let targets = $state<DeployTarget[]>([]);

	let dialogOpen = $state(false);
	let dialogMode = $state<'create' | 'edit'>('create');
	let formValue = $state<Partial<DeployTarget>>({});
	let formError = $state<string | null>(null);
	let submitting = $state(false);
	let editingId = $state<string | null>(null);

	async function load() {
		loading = true;
		loadError = null;
		try {
			targets = await deployApi.listTargets(siteID);
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load deploy targets.';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const hasDefault = $derived(targets.some((t) => t.is_default));

	function openCreate() {
		dialogMode = 'create';
		editingId = null;
		formValue = { name: '', kind: 'local', config: {}, is_default: !hasDefault };
		formError = null;
		dialogOpen = true;
	}

	function openEdit(t: DeployTarget) {
		dialogMode = 'edit';
		editingId = t.id;
		formValue = {
			id: t.id,
			site_id: t.site_id,
			name: t.name,
			kind: t.kind,
			config: { ...t.config },
			is_default: t.is_default
		};
		formError = null;
		dialogOpen = true;
	}

	async function submit() {
		if (!formValue.name || formValue.name.trim() === '') {
			formError = 'Name is required.';
			return;
		}
		if (!formValue.kind) {
			formError = 'Kind is required.';
			return;
		}
		submitting = true;
		formError = null;
		try {
			if (dialogMode === 'create') {
				await deployApi.createTarget(siteID, {
					name: formValue.name.trim(),
					kind: formValue.kind,
					config: formValue.config ?? {},
					is_default: formValue.is_default ?? false
				});
				toast.success('Deploy target created.');
			} else if (editingId) {
				await deployApi.updateTarget(siteID, editingId, {
					name: formValue.name.trim(),
					config: formValue.config ?? {}
				});
				// Default flag is updated through SetDefault, not the patch endpoint.
				if (formValue.is_default) {
					const existing = targets.find((t) => t.id === editingId);
					if (existing && !existing.is_default) {
						await deployApi.setDefault(siteID, editingId);
					}
				}
				toast.success('Deploy target updated.');
			}
			dialogOpen = false;
			await load();
		} catch (e) {
			formError = e instanceof ApiError ? e.message : 'Failed to save deploy target.';
		} finally {
			submitting = false;
		}
	}

	async function makeDefault(t: DeployTarget) {
		try {
			await deployApi.setDefault(siteID, t.id);
			toast.success(`${t.name} is now the default target.`);
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to set default.');
		}
	}

	async function remove(t: DeployTarget) {
		const ok = await confirm({
			title: 'Delete deploy target',
			message: `Remove "${t.name}"? Past deployments stay in the build history but no new deploys can use it.`,
			confirmText: 'Delete',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await deployApi.removeTarget(siteID, t.id);
			toast.success('Deploy target deleted.');
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete deploy target.');
		}
	}

	function publicURL(t: DeployTarget): string | null {
		const cfg = t.config as Record<string, unknown> | null;
		if (!cfg) return null;
		const v = cfg.public_url;
		if (typeof v === 'string' && v.trim() !== '') return v;
		const d = cfg.domain;
		if (typeof d === 'string' && d.trim() !== '') return `https://${d}`;
		return null;
	}

	function kindLabel(k: string): string {
		if (k === 'local') return 'Local';
		if (k === 'rsync') return 'Rsync';
		if (k === 'dockyard') return 'Dockyard';
		return k;
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex items-start justify-between gap-4">
		<div class="flex flex-col gap-1.5">
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Deployment
			</h1>
			<p class="text-[13px] text-text-secondary">
				Where this site gets published once a build succeeds.
			</p>
		</div>
		<Button variant="primary" onclick={openCreate}>
			{#snippet icon()}
				<Plus size={14} strokeWidth={1.75} />
			{/snippet}
			Add target
		</Button>
	</header>

	<div class="mt-8">
		{#if loading}
			<div class="flex items-center justify-center py-12">
				<Spinner />
			</div>
		{:else if loadError}
			<Card padding="md">
				<p class="text-[13px] text-danger">{loadError}</p>
			</Card>
		{:else if targets.length === 0}
			<Card padding="none">
				<EmptyState
					title="No deploy targets"
					description="Add a target to publish your built site to a server, an rsync host, or Dockyard."
				>
					{#snippet action()}
						<Button variant="secondary" onclick={openCreate}>Add deployment target</Button>
					{/snippet}
				</EmptyState>
			</Card>
		{:else}
			<div class="flex flex-col gap-2">
				{#each targets as t (t.id)}
					{@const url = publicURL(t)}
					<Card padding="sm">
						<div class="flex items-center gap-4">
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2">
									<p class="text-[13px] font-medium text-text-primary truncate">{t.name}</p>
									<Badge variant="default">{kindLabel(t.kind)}</Badge>
									{#if t.is_default}
										<Badge variant="accent">Default</Badge>
									{/if}
								</div>
								{#if url}
									<a
										href={url}
										target="_blank"
										rel="noopener noreferrer"
										class="mt-0.5 inline-block truncate font-mono text-[12px] text-text-muted hover:text-text-primary transition-colors"
									>
										{url}
									</a>
								{/if}
							</div>
							{#if !t.is_default}
								<button
									type="button"
									class="inline-flex h-8 items-center gap-1.5 rounded-lg px-2.5 text-[12px] text-text-muted hover:bg-bg-hover hover:text-text-primary transition-colors"
									onclick={() => makeDefault(t)}
									aria-label="Set as default"
								>
									<Star size={13} strokeWidth={1.75} />
									Set default
								</button>
							{/if}
							<button
								type="button"
								class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-text-muted hover:bg-bg-hover hover:text-text-primary transition-colors"
								aria-label="Edit"
								onclick={() => openEdit(t)}
							>
								<Pencil size={14} strokeWidth={1.75} />
							</button>
							<button
								type="button"
								class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-text-muted hover:bg-bg-hover hover:text-danger transition-colors"
								aria-label="Delete"
								onclick={() => remove(t)}
							>
								<Trash2 size={14} strokeWidth={1.75} />
							</button>
						</div>
					</Card>
				{/each}
			</div>
		{/if}
	</div>
</div>

<Dialog
	bind:open={dialogOpen}
	title={dialogMode === 'create' ? 'Add deploy target' : 'Edit deploy target'}
	description="Configure where the built site gets published."
	size="lg"
>
	<TargetForm
		bind:value={formValue}
		mode={dialogMode}
		showDefaultToggle={dialogMode === 'create'
			? !hasDefault
			: !(targets.find((t) => t.id === editingId)?.is_default ?? false)}
	/>
	{#if formError}
		<p class="mt-3 text-[12px] text-danger">{formError}</p>
	{/if}
	{#snippet footer()}
		<Button variant="secondary" onclick={() => (dialogOpen = false)} disabled={submitting}>
			Cancel
		</Button>
		<Button variant="primary" onclick={submit} loading={submitting}>
			{dialogMode === 'create' ? 'Create' : 'Save'}
		</Button>
	{/snippet}
</Dialog>
