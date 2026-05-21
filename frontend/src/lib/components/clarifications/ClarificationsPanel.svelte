<script lang="ts">
	import { onMount } from 'svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as clarifications from '$lib/api/clarifications';
	import { ApiError } from '$lib/api/client';
	import type { Clarification, ClarificationStatus } from '$lib/api/clarifications';

	let { siteID }: { siteID: string } = $props();

	let rows = $state<Clarification[]>([]);
	let loading = $state(true);
	let filter = $state<'pending' | 'all'>('pending');

	let resolveOpen = $state(false);
	let cancelOpen = $state(false);
	let selected = $state<Clarification | null>(null);
	let pickedIndex = $state('0');
	let freeText = $state('');
	let useFreeText = $state(false);
	let submitting = $state(false);

	const pickedOptions = $derived(
		selected
			? selected.options.map((label, i) => ({ value: String(i), label }))
			: []
	);

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			rows = await clarifications.listClarifications(siteID, { status: filter });
		} catch {
			rows = [];
		} finally {
			loading = false;
		}
	}

	async function setFilter(next: 'pending' | 'all') {
		filter = next;
		await load();
	}

	function openResolve(row: Clarification) {
		selected = row;
		pickedIndex = '0';
		freeText = '';
		useFreeText = row.options.length === 0;
		resolveOpen = true;
	}

	function openCancel(row: Clarification) {
		selected = row;
		cancelOpen = true;
	}

	async function submitResolve() {
		if (!selected) return;
		const text = freeText.trim();
		const option = useFreeText ? -1 : parseInt(pickedIndex, 10);
		if (option === -1 && text === '') {
			toast.error('Write an answer before resolving.');
			return;
		}
		submitting = true;
		try {
			const updated = await clarifications.resolveClarification(siteID, selected.id, {
				option,
				text
			});
			toast.success(`Resolved: ${truncate(selected.question, 60)}`);
			applyUpdate(updated);
			resolveOpen = false;
			selected = null;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Resolve failed');
		} finally {
			submitting = false;
		}
	}

	async function submitCancel() {
		if (!selected) return;
		submitting = true;
		try {
			const updated = await clarifications.cancelClarification(siteID, selected.id);
			toast.success(`Cancelled: ${truncate(selected.question, 60)}`);
			applyUpdate(updated);
			cancelOpen = false;
			selected = null;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Cancel failed');
		} finally {
			submitting = false;
		}
	}

	function applyUpdate(updated: Clarification) {
		if (filter === 'pending') {
			rows = rows.filter((r) => r.id !== updated.id);
		} else {
			rows = rows.map((r) => (r.id === updated.id ? updated : r));
		}
	}

	function truncate(s: string, n: number): string {
		if (s.length <= n) return s;
		return s.slice(0, n) + '…';
	}

	function statusVariant(s: ClarificationStatus): 'default' | 'success' | 'warning' {
		if (s === 'resolved') return 'success';
		if (s === 'cancelled') return 'default';
		return 'warning';
	}

	function resolutionSummary(row: Clarification): string {
		if (row.status !== 'resolved') return '';
		if (row.resolution_option >= 0 && row.resolution_option < row.options.length) {
			const label = row.options[row.resolution_option];
			return row.resolution_text ? `${label}: ${row.resolution_text}` : label;
		}
		return row.resolution_text;
	}
</script>

<Card>
	<div class="mb-4 flex items-baseline justify-between">
		<div>
			<h2 class="text-sm font-medium text-text-primary">Build agent inbox</h2>
			<p class="mt-1 text-xs text-text-muted">
				When the agent hits a decision it cannot guess safely, it opens a clarification here and
				waits for an answer. Resolve to unblock the build.
			</p>
		</div>
		<div class="flex items-center gap-1 text-xs">
			<button
				class="rounded px-2 py-1 text-text-muted hover:text-text-primary {filter === 'pending'
					? 'font-medium text-text-primary'
					: ''}"
				onclick={() => setFilter('pending')}
				type="button">Pending</button
			>
			<button
				class="rounded px-2 py-1 text-text-muted hover:text-text-primary {filter === 'all'
					? 'font-medium text-text-primary'
					: ''}"
				onclick={() => setFilter('all')}
				type="button">All</button
			>
		</div>
	</div>

	{#if loading}
		<div class="space-y-2">
			<Skeleton class="h-16" />
			<Skeleton class="h-16" />
			<Skeleton class="h-16" />
		</div>
	{:else if rows.length === 0}
		<EmptyState
			title={filter === 'pending' ? 'No pending clarifications' : 'No clarifications yet'}
			description={filter === 'pending'
				? 'The agent has nothing blocking it right now. New questions will show up here in real time.'
				: 'When the build agent asks questions, they will appear here along with their resolutions.'}
		/>
	{:else}
		<ul class="divide-y divide-border-light">
			{#each rows as row (row.id)}
				<li class="py-3">
					<div class="flex items-start justify-between gap-4">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2">
								<Badge variant={statusVariant(row.status)}>{row.status}</Badge>
								<span class="truncate text-[11px] font-mono text-text-muted">
									{row.requested_by}
								</span>
								<span class="text-[11px] text-text-muted">{row.created_at}</span>
							</div>
							<div class="mt-1.5 text-sm font-medium text-text-primary">
								{row.question}
							</div>
							{#if row.context}
								<div class="mt-1 line-clamp-3 whitespace-pre-wrap text-xs text-text-muted">
									{row.context}
								</div>
							{/if}
							{#if row.options.length > 0 && row.status === 'pending'}
								<div class="mt-2 flex flex-wrap gap-1.5">
									{#each row.options as opt}
										<span
											class="rounded-md border border-border-light bg-bg-elevated px-2 py-0.5 text-[11px] text-text-secondary"
										>
											{opt}
										</span>
									{/each}
								</div>
							{/if}
							{#if row.status !== 'pending'}
								<div class="mt-2 text-xs text-text-secondary">
									<span class="font-mono text-text-muted">{row.resolved_by}</span>
									<span class="ml-2 text-text-muted">{row.resolved_at}</span>
									{#if row.status === 'resolved'}
										<div class="mt-1 text-text-primary">{resolutionSummary(row)}</div>
									{/if}
								</div>
							{/if}
						</div>
						{#if row.status === 'pending'}
							<div class="flex shrink-0 items-center gap-2">
								<button
									type="button"
									class="text-accent text-xs hover:underline"
									onclick={() => openResolve(row)}>Resolve</button
								>
								<button
									type="button"
									class="text-text-muted hover:text-danger text-xs"
									onclick={() => openCancel(row)}>Cancel</button
								>
							</div>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</Card>

<Dialog
	bind:open={resolveOpen}
	title="Resolve clarification"
	description={selected ? selected.question : ''}
>
	<div class="space-y-3">
		{#if selected && selected.options.length > 0}
			<div>
				<label class="mb-1.5 block text-xs text-text-muted">Pick an option</label>
				<Select bind:value={pickedIndex} options={pickedOptions} disabled={useFreeText} />
			</div>
			<label class="flex items-center gap-2 text-xs text-text-secondary">
				<input
					type="checkbox"
					class="rounded border-border"
					bind:checked={useFreeText}
				/>
				Write a free-text answer instead
			</label>
		{/if}
		<div>
			<Input
				label={selected && selected.options.length > 0 && !useFreeText
					? 'Elaboration (optional)'
					: 'Answer'}
				bind:value={freeText}
				placeholder="Your answer goes back to the agent."
			/>
		</div>
	</div>
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (resolveOpen = false)} disabled={submitting}>
				Close
			</Button>
			<Button
				onclick={submitResolve}
				loading={submitting}
				disabled={useFreeText && freeText.trim() === ''}
			>
				Send to agent
			</Button>
		</div>
	{/snippet}
</Dialog>

<ConfirmDialog
	bind:open={cancelOpen}
	title="Cancel this clarification?"
	message={selected
		? `"${truncate(selected.question, 100)}" will be marked cancelled. The agent's polling loop will see the status flip and move on without an answer.`
		: ''}
	confirmText="Cancel clarification"
	variant="danger"
	onconfirm={submitCancel}
/>
