<script lang="ts">
	import { page as pageState } from '$app/state';
	import * as guardrailsApi from '$lib/api/guardrails';
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
	import type { BoolInt, GuardrailRule } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);

	const RULE_TYPES = [
		{ value: 'allow_block_type', label: 'Allow block type' },
		{ value: 'forbid_pattern', label: 'Forbid pattern' },
		{ value: 'require_block', label: 'Require block' },
		{ value: 'max_blocks', label: 'Max blocks' },
		{ value: 'allowed_host', label: 'Allowed host' },
		{ value: 'css_convention', label: 'CSS convention' }
	];

	const SEVERITY_OPTIONS = [
		{ value: 'error', label: 'Error (blocks write)' },
		{ value: 'warning', label: 'Warning (allow with toast)' }
	];

	let rules = $state<GuardrailRule[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let dialogOpen = $state(false);
	let saving = $state(false);
	let editingId = $state<string | null>(null);
	let formRuleType = $state<string>('allow_block_type');
	let formTarget = $state('');
	let formValue = $state('');
	let formSeverity = $state<string>('error');
	let formIsActive = $state(true);

	async function loadRules(id: string) {
		loading = true;
		loadError = null;
		try {
			const list = await guardrailsApi.list(id);
			rules = [...list].sort(
				(a, b) =>
					(a.rule_type || '').localeCompare(b.rule_type || '') ||
					a.target.localeCompare(b.target)
			);
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load guardrails';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void loadRules(siteID);
	});

	const grouped = $derived.by(() => {
		const groups = new Map<string, GuardrailRule[]>();
		for (const r of rules) {
			const t = r.rule_type || 'custom';
			if (!groups.has(t)) groups.set(t, []);
			groups.get(t)!.push(r);
		}
		return [...groups.entries()].sort(([a], [b]) => a.localeCompare(b));
	});

	function ruleTypeLabel(value: string): string {
		return RULE_TYPES.find((t) => t.value === value)?.label ?? value;
	}

	function openAdd() {
		editingId = null;
		formRuleType = 'allow_block_type';
		formTarget = '';
		formValue = '';
		formSeverity = 'error';
		formIsActive = true;
		dialogOpen = true;
	}

	function openEdit(r: GuardrailRule) {
		editingId = r.id;
		formRuleType = r.rule_type;
		formTarget = r.target || '';
		formValue = r.value || '';
		formSeverity = r.severity || 'error';
		formIsActive = r.is_active === 1;
		dialogOpen = true;
	}

	async function save() {
		if (!formTarget.trim() && formRuleType !== 'forbid_pattern' && formRuleType !== 'max_blocks') {
			// target is optional for some types; warn but allow empty for forbid_pattern.
		}
		saving = true;
		try {
			if (editingId) {
				const updated = await guardrailsApi.update(siteID, editingId, {
					rule_type: formRuleType,
					target: formTarget,
					value: formValue,
					severity: formSeverity,
					is_active: formIsActive ? 1 : 0
				});
				rules = rules.map((r) => (r.id === editingId ? updated : r));
				toast.success('Rule updated.');
			} else {
				const created = await guardrailsApi.create(siteID, {
					rule_type: formRuleType,
					target: formTarget,
					value: formValue,
					severity: formSeverity
				});
				let next = created;
				if (!formIsActive) {
					try {
						next = await guardrailsApi.update(siteID, created.id, { is_active: 0 });
					} catch {
						/* keep created */
					}
				}
				rules = [...rules, next].sort(
					(a, b) =>
						(a.rule_type || '').localeCompare(b.rule_type || '') ||
						a.target.localeCompare(b.target)
				);
				toast.success('Rule created.');
			}
			dialogOpen = false;
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save rule';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	async function quickAdd(input: {
		rule_type: string;
		target: string;
		value: string;
		severity: string;
	}) {
		try {
			const created = await guardrailsApi.create(siteID, input);
			rules = [...rules, created].sort(
				(a, b) =>
					(a.rule_type || '').localeCompare(b.rule_type || '') ||
					a.target.localeCompare(b.target)
			);
			toast.success('Rule added.');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to add rule';
			toast.error(msg);
		}
	}

	async function toggleActive(r: GuardrailRule, next: boolean) {
		const previous = rules;
		const newVal: BoolInt = next ? 1 : 0;
		rules = rules.map((x) => (x.id === r.id ? { ...x, is_active: newVal } : x));
		try {
			await guardrailsApi.update(siteID, r.id, { is_active: newVal });
		} catch (err) {
			rules = previous;
			const msg = err instanceof ApiError ? err.message : 'Failed to update rule';
			toast.error(msg);
		}
	}

	async function deleteRule(r: GuardrailRule) {
		const ok = await confirm({
			title: 'Delete guardrail rule?',
			message: `${ruleTypeLabel(r.rule_type)}: ${r.target || '(no target)'}. This cannot be undone.`,
			confirmText: 'Delete',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		const previous = rules;
		rules = rules.filter((x) => x.id !== r.id);
		try {
			await guardrailsApi.remove(siteID, r.id);
			toast.success('Rule deleted.');
		} catch (err) {
			rules = previous;
			const msg = err instanceof ApiError ? err.message : 'Failed to delete rule';
			toast.error(msg);
		}
	}

	function severityClasses(severity: string): string {
		if (severity === 'error') {
			return 'bg-danger/10 text-danger border border-danger/40';
		}
		return 'bg-warning/10 text-warning border border-warning/40';
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
				Guardrails
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Constraints validated on every agent write to this site.
			</p>
		</div>
		<Button variant="primary" onclick={openAdd}>Add rule</Button>
	</header>

	<Card class="mt-6" padding="sm">
		<p class="text-[12px] text-text-secondary">
			Rules are validated on every agent write. <span class="font-medium">Errors</span> block,
			<span class="font-medium">warnings</span> allow with a toast.
		</p>
	</Card>

	<section class="mt-6">
		<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Quick add</h2>
		<div class="mt-2 flex flex-wrap gap-2">
			<Button
				variant="secondary"
				size="sm"
				onclick={() =>
					quickAdd({
						rule_type: 'allow_block_type',
						target: 'block_types',
						value: 'hero,text,image,cta,feature_grid',
						severity: 'error'
					})}
			>
				Allow block types
			</Button>
			<Button
				variant="secondary"
				size="sm"
				onclick={() =>
					quickAdd({
						rule_type: 'forbid_pattern',
						target: 'inline_styles',
						value: 'style="',
						severity: 'error'
					})}
			>
				Forbid inline styles
			</Button>
			<Button
				variant="secondary"
				size="sm"
				onclick={() =>
					quickAdd({
						rule_type: 'max_blocks',
						target: 'page',
						value: '30',
						severity: 'warning'
					})}
			>
				Max 30 blocks per page
			</Button>
		</div>
	</section>

	<section class="mt-8">
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
		{:else if rules.length === 0}
			<Card padding="none">
				<EmptyState
					title="No guardrails yet"
					description="Add rules so AI agents do not break the site structure."
				>
					{#snippet action()}
						<Button variant="primary" onclick={openAdd}>Add rule</Button>
					{/snippet}
				</EmptyState>
			</Card>
		{:else}
			<div class="flex flex-col gap-6">
				{#each grouped as [type, items] (type)}
					<div class="flex flex-col gap-2">
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							{ruleTypeLabel(type)}
						</h2>
						<div class="overflow-hidden rounded-lg border border-border-light">
							{#each items as r, idx (r.id)}
								<div
									class="group flex items-center gap-3 bg-bg-surface px-4 py-3 transition-colors hover:bg-bg-elevated/40 {idx >
									0
										? 'border-t border-border-light'
										: ''}"
								>
									<span
										class="rounded-md px-1.5 py-0.5 text-[10px] uppercase tracking-wider {severityClasses(
											r.severity
										)}"
									>
										{r.severity}
									</span>
									<div class="min-w-0 flex-1">
										<div class="flex items-center gap-2">
											<span class="font-mono text-[13px] text-text-primary">
												{r.target || '(no target)'}
											</span>
										</div>
										{#if r.value}
											<p class="mt-0.5 truncate font-mono text-[12px] text-text-muted">
												{truncate(r.value, 140)}
											</p>
										{/if}
									</div>
									<BitsSwitch.Root
										checked={r.is_active === 1}
										onCheckedChange={(v: boolean) => toggleActive(r, v)}
										class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border border-transparent transition-colors data-[state=checked]:bg-accent data-[state=unchecked]:bg-bg-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
										aria-label="Toggle rule"
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
											aria-label="Edit rule"
											onclick={() => openEdit(r)}
										>
											<Pencil class="h-3.5 w-3.5" />
										</button>
										<button
											type="button"
											class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
											aria-label="Delete rule"
											onclick={() => deleteRule(r)}
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

<Dialog bind:open={dialogOpen} title={editingId ? 'Edit rule' : 'New rule'} size="lg">
	<div class="flex flex-col gap-3">
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Rule type</span>
			<Select options={RULE_TYPES} bind:value={formRuleType} />
		</div>
		<Input
			label="Target"
			value={formTarget}
			oninput={(e) => {
				formTarget = (e.currentTarget as HTMLInputElement).value;
			}}
			placeholder="block_types, inline_styles, page"
			hint="What this rule applies to."
		/>
		<Textarea
			label="Value"
			class="font-mono text-[13px]"
			rows={5}
			value={formValue}
			oninput={(e) => {
				formValue = (e.currentTarget as HTMLTextAreaElement).value;
			}}
			placeholder={'hero,text,image,cta,feature_grid'}
		/>
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Severity</span>
			<Select options={SEVERITY_OPTIONS} bind:value={formSeverity} />
		</div>
		<div class="flex items-center justify-between">
			<div>
				<p class="text-[13px] font-medium text-text-primary">Active</p>
				<p class="text-[11px] text-text-muted">Inactive rules are skipped during validation.</p>
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
