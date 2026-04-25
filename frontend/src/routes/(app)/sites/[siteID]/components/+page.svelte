<script lang="ts">
	import { goto } from '$app/navigation';
	import { page as pageState } from '$app/state';
	import * as componentsApi from '$lib/api/components';
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
	import type { Component } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);

	let components = $state<Component[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let search = $state('');

	let newDialogOpen = $state(false);
	let creating = $state(false);
	let newName = $state('');
	let newCategory = $state('section');
	let newTemplate = $state('');
	let newUsageNote = $state('');

	const categoryOptions = [
		{ value: 'section', label: 'Section' },
		{ value: 'layout', label: 'Layout' },
		{ value: 'ui', label: 'UI' },
		{ value: 'navigation', label: 'Navigation' },
		{ value: 'form', label: 'Form' },
		{ value: 'custom', label: 'Custom' }
	];

	async function loadComponents(id: string) {
		loading = true;
		loadError = null;
		try {
			const list = await componentsApi.list(id);
			components = [...list].sort(
				(a, b) =>
					(a.category || '').localeCompare(b.category || '') || a.name.localeCompare(b.name)
			);
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load components';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void loadComponents(siteID);
	});

	const filtered = $derived.by(() => {
		const q = search.trim().toLowerCase();
		if (!q) return components;
		return components.filter((c) => c.name.toLowerCase().includes(q));
	});

	const grouped = $derived.by(() => {
		const groups = new Map<string, Component[]>();
		for (const c of filtered) {
			const cat = c.category || 'section';
			if (!groups.has(cat)) groups.set(cat, []);
			groups.get(cat)!.push(c);
		}
		return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b));
	});

	function openNewDialog() {
		newName = '';
		newCategory = 'section';
		newTemplate = '';
		newUsageNote = '';
		newDialogOpen = true;
	}

	async function createComponent() {
		const name = newName.trim();
		if (name.length < 2) {
			toast.error('Name must be at least 2 characters.');
			return;
		}
		creating = true;
		try {
			const created = await componentsApi.create(siteID, {
				name,
				category: newCategory,
				template: newTemplate,
				usage_note: newUsageNote
			});
			newDialogOpen = false;
			toast.success('Component created.');
			await goto(`/sites/${siteID}/components/${created.id}`);
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to create component';
			toast.error(msg);
		} finally {
			creating = false;
		}
	}

	function openComponent(c: Component) {
		void goto(`/sites/${siteID}/components/${c.id}`);
	}

	async function deleteComponent(c: Component) {
		const ok = await confirm({
			title: `Delete "${c.name}"?`,
			message: 'This removes the component permanently. This cannot be undone.',
			confirmText: 'Delete',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		const previous = components;
		components = components.filter((x) => x.id !== c.id);
		try {
			await componentsApi.remove(siteID, c.id);
			toast.success('Component deleted.');
		} catch (err) {
			components = previous;
			const msg = err instanceof ApiError ? err.message : 'Failed to delete component';
			toast.error(msg);
		}
	}

	function truncate(s: string, n: number): string {
		if (!s) return '';
		return s.length > n ? `${s.slice(0, n)}...` : s;
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex items-center justify-between">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Components
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Reusable Astro components with typed props. Click a row to open the editor.
			</p>
		</div>
		<Button variant="primary" onclick={openNewDialog}>New component</Button>
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
		{:else if components.length === 0}
			<Card padding="none">
				<EmptyState
					title="No components yet"
					description="Components are reusable Astro snippets. Create your first to start composing pages."
				>
					{#snippet action()}
						<Button variant="primary" onclick={openNewDialog}>Create component</Button>
					{/snippet}
				</EmptyState>
			</Card>
		{:else if filtered.length === 0}
			<Card padding="md">
				<p class="text-[13px] text-text-muted">No components match "{search}".</p>
			</Card>
		{:else}
			<div class="flex flex-col gap-6">
				{#each grouped as [cat, items] (cat)}
					<div class="flex flex-col gap-2">
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							{cat}
						</h2>
						<div class="flex flex-col gap-1.5">
							{#each items as c (c.id)}
								<div
									class="group flex items-center gap-3 rounded-lg border border-border-light bg-bg-surface px-4 py-3 transition-colors hover:bg-bg-elevated/40"
								>
									<button
										type="button"
										class="flex min-w-0 flex-1 flex-col gap-0.5 text-left focus-visible:outline-none"
										onclick={() => openComponent(c)}
									>
										<div class="flex items-center gap-2">
											<span
												class="font-mono text-[13px] text-text-primary group-hover:text-accent"
											>
												{c.name}
											</span>
											<span
												class="rounded-md border border-border-light bg-bg-elevated px-1.5 py-0.5 text-[10px] uppercase tracking-wider text-text-muted"
											>
												{c.category || 'section'}
											</span>
										</div>
										{#if c.usage_note}
											<p class="truncate text-[12px] text-text-muted">
												{truncate(c.usage_note, 140)}
											</p>
										{/if}
									</button>
									<div
										class="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100"
									>
										<button
											type="button"
											class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary"
											aria-label="Edit component"
											onclick={() => openComponent(c)}
										>
											<Pencil class="h-3.5 w-3.5" />
										</button>
										<button
											type="button"
											class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
											aria-label="Delete component"
											onclick={() => deleteComponent(c)}
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

<Dialog bind:open={newDialogOpen} title="New component" size="md">
	<div class="flex flex-col gap-3">
		<Input
			label="Name"
			value={newName}
			oninput={(e) => {
				newName = (e.currentTarget as HTMLInputElement).value;
			}}
			placeholder="Hero, FeatureGrid, PricingTable"
		/>
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Category</span>
			<Select options={categoryOptions} bind:value={newCategory} />
		</div>
		<Textarea
			label="Template"
			class="font-mono text-[13px]"
			rows={6}
			value={newTemplate}
			oninput={(e) => {
				newTemplate = (e.currentTarget as HTMLTextAreaElement).value;
			}}
			placeholder={'---\nconst { headline } = Astro.props;\n---\n<section>...</section>'}
			hint="Astro source. You can edit later."
		/>
		<Textarea
			label="Usage note"
			rows={2}
			value={newUsageNote}
			oninput={(e) => {
				newUsageNote = (e.currentTarget as HTMLTextAreaElement).value;
			}}
			placeholder="When to use this component."
		/>
	</div>
	{#snippet footer()}
		<Button variant="secondary" onclick={() => (newDialogOpen = false)}>Cancel</Button>
		<Button variant="primary" loading={creating} onclick={createComponent}>Create</Button>
	{/snippet}
</Dialog>
