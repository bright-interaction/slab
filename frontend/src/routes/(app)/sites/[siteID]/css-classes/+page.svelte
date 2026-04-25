<script lang="ts">
	import { page as pageState } from '$app/state';
	import * as cssClassesApi from '$lib/api/cssClasses';
	import { ApiError } from '$lib/api/client';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Pencil, Trash2 } from 'lucide-svelte';
	import type { CssClass } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);

	let classes = $state<CssClass[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let search = $state('');

	let dialogOpen = $state(false);
	let saving = $state(false);
	let editingId = $state<string | null>(null);
	let formName = $state('');
	let formCategory = $state('utility');
	let formCss = $state('');
	let formUsageNote = $state('');

	const categoryOptions = [
		{ value: 'utility', label: 'Utility' },
		{ value: 'component', label: 'Component' },
		{ value: 'layout', label: 'Layout' },
		{ value: 'typography', label: 'Typography' },
		{ value: 'animation', label: 'Animation' }
	];

	async function loadClasses(id: string) {
		loading = true;
		loadError = null;
		try {
			const list = await cssClassesApi.list(id);
			classes = [...list].sort(
				(a, b) =>
					(a.category || '').localeCompare(b.category || '') ||
					a.sort_order - b.sort_order ||
					a.name.localeCompare(b.name)
			);
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load CSS classes';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void loadClasses(siteID);
	});

	const filtered = $derived.by(() => {
		const q = search.trim().toLowerCase();
		if (!q) return classes;
		return classes.filter((c) => c.name.toLowerCase().includes(q));
	});

	const grouped = $derived.by(() => {
		const groups = new Map<string, CssClass[]>();
		for (const c of filtered) {
			const cat = c.category || 'utility';
			if (!groups.has(cat)) groups.set(cat, []);
			groups.get(cat)!.push(c);
		}
		return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b));
	});

	function openAdd() {
		editingId = null;
		formName = '';
		formCategory = 'utility';
		formCss = '';
		formUsageNote = '';
		dialogOpen = true;
	}

	function openEdit(c: CssClass) {
		editingId = c.id;
		formName = c.name;
		formCategory = c.category || 'utility';
		formCss = c.css || '';
		formUsageNote = c.usage_note || '';
		dialogOpen = true;
	}

	async function save() {
		const name = formName.trim();
		if (name.length < 1) {
			toast.error('Name is required.');
			return;
		}
		saving = true;
		try {
			if (editingId) {
				const updated = await cssClassesApi.update(siteID, editingId, {
					name,
					category: formCategory,
					css: formCss,
					usage_note: formUsageNote
				});
				classes = classes.map((c) => (c.id === editingId ? updated : c));
				toast.success('Class updated.');
			} else {
				const created = await cssClassesApi.create(siteID, {
					name,
					category: formCategory,
					css: formCss,
					usage_note: formUsageNote
				});
				classes = [...classes, created].sort(
					(a, b) =>
						(a.category || '').localeCompare(b.category || '') ||
						a.sort_order - b.sort_order ||
						a.name.localeCompare(b.name)
				);
				toast.success('Class created.');
			}
			dialogOpen = false;
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save class';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	async function deleteClass(c: CssClass) {
		const ok = await confirm({
			title: `Delete "${c.name}"?`,
			message: 'This removes the CSS class permanently. This cannot be undone.',
			confirmText: 'Delete',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		const previous = classes;
		classes = classes.filter((x) => x.id !== c.id);
		try {
			await cssClassesApi.remove(siteID, c.id);
			toast.success('Class deleted.');
		} catch (err) {
			classes = previous;
			const msg = err instanceof ApiError ? err.message : 'Failed to delete class';
			toast.error(msg);
		}
	}

	function truncate(s: string, n: number): string {
		if (!s) return '';
		const flat = s.replace(/\s+/g, ' ');
		return flat.length > n ? `${flat.slice(0, n)}...` : flat;
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex items-center justify-between">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				CSS classes
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Global CSS classes available across components and blocks.
			</p>
		</div>
		<Button variant="primary" onclick={openAdd}>Add class</Button>
	</header>

	<div class="mt-6 max-w-md">
		<Input
			placeholder="Search by name."
			value={search}
			oninput={(e) => {
				search = (e.currentTarget as HTMLInputElement).value;
			}}
		/>
	</div>

	<section class="mt-6">
		{#if loading}
			<div class="flex flex-col gap-2">
				{#each Array(4) as _, i (i)}
					<Card padding="sm">
						<Skeleton width="40%" height="0.95rem" />
					</Card>
				{/each}
			</div>
		{:else if loadError}
			<Card padding="md">
				<p class="text-[13px] text-danger">{loadError}</p>
			</Card>
		{:else if classes.length === 0}
			<Card padding="none">
				<EmptyState
					title="No CSS classes yet"
					description="Define utilities, layouts, typography, or component classes for this site."
				>
					{#snippet action()}
						<Button variant="primary" onclick={openAdd}>Add class</Button>
					{/snippet}
				</EmptyState>
			</Card>
		{:else if filtered.length === 0}
			<Card padding="md">
				<p class="text-[13px] text-text-muted">No classes match "{search}".</p>
			</Card>
		{:else}
			<div class="flex flex-col gap-6">
				{#each grouped as [cat, items] (cat)}
					<div class="flex flex-col gap-2">
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							{cat}
						</h2>
						<div class="overflow-hidden rounded-lg border border-border-light">
							{#each items as c, idx (c.id)}
								<div
									class="group flex items-center gap-3 bg-bg-surface px-4 py-3 transition-colors hover:bg-bg-elevated/40 {idx >
									0
										? 'border-t border-border-light'
										: ''}"
								>
									<div class="min-w-0 flex-1">
										<div class="flex items-center gap-2">
											<span class="font-mono text-[13px] text-text-primary">.{c.name}</span>
											<span
												class="rounded-md border border-border-light bg-bg-elevated px-1.5 py-0.5 text-[10px] uppercase tracking-wider text-text-muted"
											>
												{c.category || 'utility'}
											</span>
										</div>
										{#if c.css}
											<p class="mt-0.5 truncate font-mono text-[12px] text-text-muted">
												{truncate(c.css, 100)}
											</p>
										{/if}
										{#if c.usage_note}
											<p class="mt-0.5 truncate text-[12px] text-text-muted">
												{truncate(c.usage_note, 120)}
											</p>
										{/if}
									</div>
									<div
										class="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100"
									>
										<button
											type="button"
											class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary"
											aria-label="Edit class"
											onclick={() => openEdit(c)}
										>
											<Pencil class="h-3.5 w-3.5" />
										</button>
										<button
											type="button"
											class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
											aria-label="Delete class"
											onclick={() => deleteClass(c)}
										>
											<Trash2 class="h-3.5 w-3.5" />
										</button>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</section>
</div>

<Dialog bind:open={dialogOpen} title={editingId ? 'Edit class' : 'Add class'} size="md">
	<div class="flex flex-col gap-3">
		<Input
			label="Name"
			value={formName}
			oninput={(e) => {
				formName = (e.currentTarget as HTMLInputElement).value;
			}}
			placeholder="btn-primary"
			hint="No leading dot."
		/>
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Category</span>
			<Select options={categoryOptions} bind:value={formCategory} />
		</div>
		<Textarea
			label="CSS"
			class="font-mono text-[13px]"
			rows={8}
			value={formCss}
			oninput={(e) => {
				formCss = (e.currentTarget as HTMLTextAreaElement).value;
			}}
			placeholder={'background: var(--accent);\ncolor: white;\npadding: 0.5rem 1rem;'}
		/>
		<Textarea
			label="Usage note"
			rows={2}
			value={formUsageNote}
			oninput={(e) => {
				formUsageNote = (e.currentTarget as HTMLTextAreaElement).value;
			}}
			placeholder="When to apply this class."
		/>
	</div>
	{#snippet footer()}
		<Button variant="secondary" onclick={() => (dialogOpen = false)}>Cancel</Button>
		<Button variant="primary" loading={saving} onclick={save}>
			{editingId ? 'Save' : 'Create'}
		</Button>
	{/snippet}
</Dialog>
