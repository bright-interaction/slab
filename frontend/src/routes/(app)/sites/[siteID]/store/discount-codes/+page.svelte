<script lang="ts">
	import { onMount } from 'svelte';
	import { page as pageState } from '$app/state';
	import * as discountApi from '$lib/api/discountCodes';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { ApiError } from '$lib/api/client';
	import type { DiscountCode, DiscountKind } from '$lib/api/discountCodes';

	const siteID = $derived(pageState.params.siteID as string);

	let codes = $state<DiscountCode[]>([]);
	let loading = $state(true);
	let error = $state('');

	let createOpen = $state(false);
	let creating = $state(false);
	let newCode = $state('');
	let newKind = $state<DiscountKind>('percent');
	let newPercent = $state('15');
	let newFixedEur = $state('10.00');
	let newMinSubtotalEur = $state('');
	let newMaxUses = $state('');
	let newActive = $state(true);

	const kindOptions = [
		{ value: 'percent', label: 'Percent off' },
		{ value: 'fixed', label: 'Fixed amount off' }
	];

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			codes = await discountApi.listDiscountCodes(siteID);
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Failed to load discount codes';
			codes = [];
		} finally {
			loading = false;
		}
	}

	function formatValue(c: DiscountCode): string {
		if (c.kind === 'percent') {
			return `${(c.value / 100).toFixed(c.value % 100 === 0 ? 0 : 2)}% off`;
		}
		return `€${(c.value / 100).toFixed(2)} off`;
	}

	async function createCode() {
		const code = newCode.trim().toUpperCase();
		if (!/^[A-Z0-9][A-Z0-9_-]{1,49}$/.test(code)) {
			toast.error('Code must be 2-50 uppercase / digits / dash / underscore.');
			return;
		}
		const value =
			newKind === 'percent'
				? Math.round(Number(newPercent) * 100)
				: Math.round(Number(newFixedEur) * 100);
		if (newKind === 'percent' && (value < 1 || value > 10000)) {
			toast.error('Percent must be between 0.01 and 100.');
			return;
		}
		if (newKind === 'fixed' && value < 1) {
			toast.error('Fixed amount must be > 0.');
			return;
		}
		creating = true;
		try {
			const body: discountApi.DiscountCodeInput = {
				code,
				kind: newKind,
				value,
				is_active: newActive
			};
			if (newMinSubtotalEur.trim()) {
				body.min_subtotal_cents = Math.round(Number(newMinSubtotalEur) * 100);
			}
			if (newMaxUses.trim()) {
				body.max_uses = Math.round(Number(newMaxUses));
			}
			const created = await discountApi.createDiscountCode(siteID, body);
			codes = [created, ...codes];
			toast.success(`Created ${created.code}`);
			createOpen = false;
			newCode = '';
			newPercent = '15';
			newFixedEur = '10.00';
			newMinSubtotalEur = '';
			newMaxUses = '';
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Create failed');
		} finally {
			creating = false;
		}
	}

	async function toggleActive(c: DiscountCode) {
		try {
			const updated = await discountApi.updateDiscountCode(siteID, c.id, {
				is_active: c.is_active !== 1
			});
			codes = codes.map((x) => (x.id === updated.id ? updated : x));
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Toggle failed');
		}
	}

	async function deleteCode(c: DiscountCode) {
		const ok = await confirm({
			title: `Delete code ${c.code}?`,
			message: 'This removes the code permanently. Use deactivate (toggle) if you might re-enable later.',
			confirmText: 'Delete',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await discountApi.deleteDiscountCode(siteID, c.id);
			codes = codes.filter((x) => x.id !== c.id);
			toast.success('Code deleted');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed');
		}
	}
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-sm font-medium text-text-primary">Discount codes</h2>
			<p class="text-xs text-text-muted mt-1">
				{codes.length} code{codes.length === 1 ? '' : 's'}. Codes are redeemed at checkout in Slice B.
			</p>
		</div>
		<Button onclick={() => (createOpen = true)}>+ New code</Button>
	</div>

	<Card>
		{#if loading}
			<div class="space-y-2"><Skeleton class="h-10" /><Skeleton class="h-10" /></div>
		{:else if error}
			<div class="rounded-lg border border-danger/30 bg-danger/5 p-3 text-xs text-danger">{error}</div>
		{:else if codes.length === 0}
			<EmptyState
				title="No discount codes"
				description="Create a code with a percent (e.g. 15% off) or fixed amount (e.g. €10 off). Apply to the whole cart or a subset of products / categories."
			>
				{#snippet action()}
					<Button onclick={() => (createOpen = true)}>+ New code</Button>
				{/snippet}
			</EmptyState>
		{:else}
			<div class="overflow-x-auto rounded-lg border border-border-light">
				<table class="w-full text-xs">
					<thead class="bg-bg-elevated text-text-muted">
						<tr>
							<th class="text-left font-normal px-3 py-2">Code</th>
							<th class="text-left font-normal px-3 py-2">Value</th>
							<th class="text-left font-normal px-3 py-2">Applies to</th>
							<th class="text-right font-normal px-3 py-2 tabular-nums">Used</th>
							<th class="text-left font-normal px-3 py-2">Active</th>
							<th class="text-right font-normal px-3 py-2"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border-light">
						{#each codes as c (c.id)}
							<tr>
								<td class="px-3 py-1.5 font-mono">{c.code}</td>
								<td class="px-3 py-1.5">{formatValue(c)}</td>
								<td class="px-3 py-1.5 text-text-muted">{c.applies_to}</td>
								<td class="px-3 py-1.5 text-right tabular-nums">
									{c.used_count}{c.max_uses > 0 ? ` / ${c.max_uses}` : ''}
								</td>
								<td class="px-3 py-1.5">
									<button
										type="button"
										class="cursor-pointer"
										onclick={() => toggleActive(c)}
										aria-label="Toggle active"
									>
										<Badge variant={c.is_active === 1 ? 'success' : 'default'}>
											{c.is_active === 1 ? 'Active' : 'Inactive'}
										</Badge>
									</button>
								</td>
								<td class="px-3 py-1.5 text-right">
									<button
										type="button"
										class="text-text-muted text-xs hover:text-danger"
										onclick={() => deleteCode(c)}
									>Delete</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</Card>
</div>

<Dialog
	bind:open={createOpen}
	title="New discount code"
	description="Code is unique per site. Percent stored as basis points (1500 = 15%), fixed stored as cents."
>
	<div class="space-y-3">
		<Input label="Code" bind:value={newCode} placeholder="SUMMER15" hint="Uppercase A-Z, digits, dash, underscore" />
		<div class="grid grid-cols-2 gap-3">
			<div>
				<label class="mb-1.5 block text-xs text-text-muted">Kind</label>
				<Select bind:value={newKind} options={kindOptions} />
			</div>
			{#if newKind === 'percent'}
				<Input label="Percent off" bind:value={newPercent} placeholder="15" />
			{:else}
				<Input label="Fixed off" bind:value={newFixedEur} placeholder="10.00" />
			{/if}
		</div>
		<div class="grid grid-cols-2 gap-3">
			<Input label="Min subtotal (optional)" bind:value={newMinSubtotalEur} placeholder="50.00" />
			<Input label="Max uses (optional)" bind:value={newMaxUses} placeholder="100" />
		</div>
		<label class="flex items-center gap-2 text-xs text-text-secondary">
			<input type="checkbox" class="rounded border-border" bind:checked={newActive} />
			Active immediately
		</label>
	</div>
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (createOpen = false)} disabled={creating}>Cancel</Button>
			<Button onclick={createCode} loading={creating} disabled={!newCode.trim()}>Create</Button>
		</div>
	{/snippet}
</Dialog>
