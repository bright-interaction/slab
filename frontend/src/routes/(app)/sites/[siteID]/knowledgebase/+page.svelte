<script lang="ts">
	import { page as pageState } from '$app/state';
	import * as kbApi from '$lib/api/knowledgebase';
	import { ApiError } from '$lib/api/client';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import { Switch as BitsSwitch } from 'bits-ui';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Pencil, Trash2 } from 'lucide-svelte';
	import type { BoolInt, KnowledgebaseEntry } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);

	const CATEGORIES = [
		{ value: 'brand', label: 'Brand' },
		{ value: 'voice', label: 'Voice' },
		{ value: 'technical', label: 'Technical' },
		{ value: 'seo', label: 'SEO' },
		{ value: 'accessibility', label: 'Accessibility' },
		{ value: 'custom', label: 'Custom' }
	] as const;

	let entries = $state<KnowledgebaseEntry[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let search = $state('');
	let activeCategory = $state<string>('brand');

	let dialogOpen = $state(false);
	let saving = $state(false);
	let editingId = $state<string | null>(null);
	let formTitle = $state('');
	let formCategory = $state<string>('brand');
	let formContent = $state('');
	let formIsActive = $state(true);

	async function loadEntries(id: string) {
		loading = true;
		loadError = null;
		try {
			const list = await kbApi.list(id);
			entries = [...list].sort(
				(a, b) =>
					(a.category || '').localeCompare(b.category || '') ||
					a.sort_order - b.sort_order ||
					a.title.localeCompare(b.title)
			);
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load knowledgebase';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void loadEntries(siteID);
	});

	const filtered = $derived.by(() => {
		const q = search.trim().toLowerCase();
		const inCategory = entries.filter((e) => (e.category || 'custom') === activeCategory);
		if (!q) return inCategory;
		return inCategory.filter((e) => e.title.toLowerCase().includes(q));
	});

	const categoryCounts = $derived.by(() => {
		const counts = new Map<string, number>();
		for (const e of entries) {
			const c = e.category || 'custom';
			counts.set(c, (counts.get(c) ?? 0) + 1);
		}
		return counts;
	});

	function openAdd() {
		editingId = null;
		formTitle = '';
		formCategory = activeCategory;
		formContent = '';
		formIsActive = true;
		dialogOpen = true;
	}

	function openEdit(e: KnowledgebaseEntry) {
		editingId = e.id;
		formTitle = e.title;
		formCategory = e.category || 'custom';
		formContent = e.content || '';
		formIsActive = e.is_active === 1;
		dialogOpen = true;
	}

	async function save() {
		const title = formTitle.trim();
		if (title.length < 1) {
			toast.error('Title is required.');
			return;
		}
		saving = true;
		try {
			if (editingId) {
				const updated = await kbApi.update(siteID, editingId, {
					title,
					category: formCategory,
					content: formContent,
					is_active: formIsActive ? 1 : 0
				});
				entries = entries.map((e) => (e.id === editingId ? updated : e));
				toast.success('Entry updated.');
			} else {
				const created = await kbApi.create(siteID, {
					title,
					category: formCategory,
					content: formContent
				});
				let next = created;
				if (!formIsActive) {
					try {
						next = await kbApi.update(siteID, created.id, { is_active: 0 });
					} catch {
						/* keep created as-is */
					}
				}
				entries = [...entries, next].sort(
					(a, b) =>
						(a.category || '').localeCompare(b.category || '') ||
						a.sort_order - b.sort_order ||
						a.title.localeCompare(b.title)
				);
				toast.success('Entry created.');
			}
			dialogOpen = false;
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save entry';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	async function toggleActive(e: KnowledgebaseEntry, next: boolean) {
		const previous = entries;
		const newVal: BoolInt = next ? 1 : 0;
		entries = entries.map((x) => (x.id === e.id ? { ...x, is_active: newVal } : x));
		try {
			await kbApi.update(siteID, e.id, { is_active: newVal });
		} catch (err) {
			entries = previous;
			const msg = err instanceof ApiError ? err.message : 'Failed to update entry';
			toast.error(msg);
		}
	}

	async function deleteEntry(e: KnowledgebaseEntry) {
		const ok = await confirm({
			title: `Delete "${e.title}"?`,
			message: 'This removes the knowledgebase entry permanently. This cannot be undone.',
			confirmText: 'Delete',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		const previous = entries;
		entries = entries.filter((x) => x.id !== e.id);
		try {
			await kbApi.remove(siteID, e.id);
			toast.success('Entry deleted.');
		} catch (err) {
			entries = previous;
			const msg = err instanceof ApiError ? err.message : 'Failed to delete entry';
			toast.error(msg);
		}
	}

	function formatDate(ts: string): string {
		if (!ts) return '';
		const d = new Date(ts);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex items-center justify-between">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Knowledgebase
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Brand voice, technical guidelines, and constraints for AI agents.
			</p>
		</div>
		<Button variant="primary" onclick={openAdd}>Add entry</Button>
	</header>

	<Card class="mt-6" padding="sm">
		<p class="text-[12px] text-text-secondary">
			AI agents see these entries when they call <span class="font-mono">/api/agent/context</span>.
			Toggle <span class="font-medium">active</span> to include or exclude an entry from the agent context.
		</p>
	</Card>

	<div class="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-[14rem_1fr]">
		<aside class="flex flex-col gap-1">
			{#each CATEGORIES as cat (cat.value)}
				{@const count = categoryCounts.get(cat.value) ?? 0}
				{@const active = activeCategory === cat.value}
				<button
					type="button"
					class="flex items-center justify-between rounded-md px-3 py-2 text-left text-[13px] transition-colors {active
						? 'bg-bg-elevated text-text-primary font-medium'
						: 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'}"
					onclick={() => (activeCategory = cat.value)}
				>
					<span>{cat.label}</span>
					<span class="text-[11px] text-text-muted">{count}</span>
				</button>
			{/each}
		</aside>

		<section>
			<div class="mb-4 max-w-md">
				<Input
					placeholder="Search by title."
					value={search}
					oninput={(e) => {
						search = (e.currentTarget as HTMLInputElement).value;
					}}
				/>
			</div>

			{#if loading}
				<div class="flex flex-col gap-2">
					{#each Array(3) as _, i (i)}
						<Card padding="sm">
							<Skeleton width="40%" height="0.95rem" />
						</Card>
					{/each}
				</div>
			{:else if loadError}
				<Card padding="md">
					<p class="text-[13px] text-danger">{loadError}</p>
				</Card>
			{:else if filtered.length === 0}
				<Card padding="none">
					<EmptyState
						title={`No ${activeCategory} entries`}
						description="Add an entry to give agents this kind of context for the site."
					>
						{#snippet action()}
							<Button variant="primary" onclick={openAdd}>Add entry</Button>
						{/snippet}
					</EmptyState>
				</Card>
			{:else}
				<div class="overflow-hidden rounded-lg border border-border-light">
					{#each filtered as e, idx (e.id)}
						<div
							class="group flex items-center gap-3 bg-bg-surface px-4 py-3 transition-colors hover:bg-bg-elevated/40 {idx >
							0
								? 'border-t border-border-light'
								: ''}"
						>
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<span class="truncate text-[13px] font-medium text-text-primary">
										{e.title}
									</span>
									{#if e.is_active === 0}
										<span
											class="rounded-md border border-border-light bg-bg-elevated px-1.5 py-0.5 text-[10px] uppercase tracking-wider text-text-muted"
										>
											inactive
										</span>
									{/if}
								</div>
								<p class="mt-0.5 text-[11px] text-text-muted">
									updated {formatDate(e.updated_at)}
								</p>
							</div>
							<BitsSwitch.Root
								checked={e.is_active === 1}
								onCheckedChange={(v: boolean) => toggleActive(e, v)}
								class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border border-transparent transition-colors data-[state=checked]:bg-accent data-[state=unchecked]:bg-bg-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
								aria-label={`Toggle ${e.title}`}
							>
								<BitsSwitch.Thumb
									class="pointer-events-none block h-3.5 w-3.5 rounded-full bg-bg-surface shadow-sm transition-transform data-[state=checked]:translate-x-[18px] data-[state=unchecked]:translate-x-[3px]"
								/>
							</BitsSwitch.Root>
							<div
								class="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100"
							>
								<button
									type="button"
									class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary"
									aria-label="Edit entry"
									onclick={() => openEdit(e)}
								>
									<Pencil class="h-3.5 w-3.5" />
								</button>
								<button
									type="button"
									class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
									aria-label="Delete entry"
									onclick={() => deleteEntry(e)}
								>
									<Trash2 class="h-3.5 w-3.5" />
								</button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</section>
	</div>
</div>

<Dialog bind:open={dialogOpen} title={editingId ? 'Edit entry' : 'New entry'} size="lg">
	<div class="flex flex-col gap-3">
		<Input
			label="Title"
			value={formTitle}
			oninput={(e) => {
				formTitle = (e.currentTarget as HTMLInputElement).value;
			}}
			placeholder="Brand voice principles"
		/>
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Category</span>
			<Select options={CATEGORIES.map((c) => ({ value: c.value, label: c.label }))} bind:value={formCategory} />
		</div>
		<Textarea
			label="Content"
			class="font-mono text-[13px]"
			rows={12}
			value={formContent}
			oninput={(e) => {
				formContent = (e.currentTarget as HTMLTextAreaElement).value;
			}}
			placeholder="Markdown content. Use plain text or markdown."
			hint="Markdown. Agents read this verbatim."
		/>
		<div class="flex items-center justify-between">
			<div>
				<p class="text-[13px] font-medium text-text-primary">Active</p>
				<p class="text-[11px] text-text-muted">Inactive entries are hidden from agent context.</p>
			</div>
			<Switch bind:checked={formIsActive} />
		</div>
	</div>
	{#snippet footer()}
		<Button variant="secondary" onclick={() => (dialogOpen = false)}>Cancel</Button>
		<Button variant="primary" loading={saving} onclick={save}>
			{editingId ? 'Save' : 'Create'}
		</Button>
	{/snippet}
</Dialog>
