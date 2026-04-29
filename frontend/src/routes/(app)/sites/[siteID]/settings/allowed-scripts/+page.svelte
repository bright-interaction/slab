<script lang="ts">
	import * as allowedScriptsApi from '$lib/api/allowedScripts';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Site, AllowedScript } from '$lib/api/types';
	import { Plus, Trash2, Pencil } from 'lucide-svelte';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let loading = $state(true);
	let scripts = $state<AllowedScript[]>([]);
	let loadError = $state<string | null>(null);

	let dialogOpen = $state(false);
	let editingId = $state<string | null>(null);
	let formDomain = $state('');
	let formPurpose = $state('');
	let formError = $state<string | null>(null);
	let submitting = $state(false);

	const HOSTNAME_RE = /^(?!:\/\/)([a-zA-Z0-9-_]+\.)*[a-zA-Z0-9][a-zA-Z0-9-_]*\.[a-zA-Z]{2,}$/;

	async function load() {
		loading = true;
		loadError = null;
		try {
			scripts = await allowedScriptsApi.list(siteID);
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load allowed scripts.';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	function openAdd() {
		editingId = null;
		formDomain = '';
		formPurpose = '';
		formError = null;
		dialogOpen = true;
	}

	function openEdit(s: AllowedScript) {
		editingId = s.id;
		formDomain = s.domain;
		formPurpose = s.purpose;
		formError = null;
		dialogOpen = true;
	}

	function validateDomain(d: string): string | null {
		const trimmed = d.trim();
		if (!trimmed) return 'Domain is required.';
		if (trimmed.includes('://')) return 'Hostname only. Drop the protocol.';
		if (trimmed.includes('/')) return 'Hostname only. No paths.';
		if (!HOSTNAME_RE.test(trimmed)) return 'Looks invalid. Use example.com format.';
		return null;
	}

	async function submit() {
		const err = validateDomain(formDomain);
		if (err) {
			formError = err;
			return;
		}
		formError = null;
		submitting = true;
		try {
			if (editingId) {
				await allowedScriptsApi.update(siteID, editingId, {
					domain: formDomain.trim(),
					purpose: formPurpose.trim()
				});
				toast.success('Allowed script updated.');
			} else {
				await allowedScriptsApi.create(siteID, {
					domain: formDomain.trim(),
					purpose: formPurpose.trim()
				});
				toast.success('Allowed script added.');
			}
			dialogOpen = false;
			await load();
		} catch (e) {
			const msg = e instanceof ApiError ? e.message : 'Failed to save.';
			formError = msg;
		} finally {
			submitting = false;
		}
	}

	async function toggleActive(s: AllowedScript) {
		const next = s.is_active === 1 ? 0 : 1;
		try {
			await allowedScriptsApi.update(siteID, s.id, { is_active: next });
			scripts = scripts.map((it) => (it.id === s.id ? { ...it, is_active: next } : it));
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to toggle.');
		}
	}

	async function remove(s: AllowedScript) {
		const ok = await confirm({
			title: 'Remove allowed script',
			message: `Remove ${s.domain} from the CSP whitelist? Pages using it will be blocked on next build.`,
			confirmText: 'Remove',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await allowedScriptsApi.remove(siteID, s.id);
			scripts = scripts.filter((it) => it.id !== s.id);
			toast.success('Removed.');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to remove.');
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex items-start justify-between gap-4">
		<div class="flex flex-col gap-1.5">
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Allowed Scripts
			</h1>
			<p class="text-[13px] text-text-secondary">
				Third-party domains permitted by Content-Security-Policy.
			</p>
		</div>
		<Button variant="primary" onclick={openAdd}>
			{#snippet icon()}
				<Plus size={14} strokeWidth={1.75} />
			{/snippet}
			Add domain
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
		{:else if scripts.length === 0}
			<Card padding="none">
				<EmptyState
					title="No allowed scripts"
					description="Add a domain to whitelist it in this site's Content-Security-Policy."
				>
					{#snippet action()}
						<Button variant="secondary" onclick={openAdd}>Add first domain</Button>
					{/snippet}
				</EmptyState>
			</Card>
		{:else}
			<div class="flex flex-col gap-2">
				{#each scripts as s (s.id)}
					<Card padding="sm">
						<div class="flex items-center gap-4">
							<div class="flex-1 min-w-0">
								<p class="font-mono text-[13px] text-text-primary truncate">{s.domain}</p>
								{#if s.purpose}
									<p class="text-[12px] text-text-muted truncate">{s.purpose}</p>
								{:else}
									<p class="text-[12px] text-text-muted italic">No purpose noted.</p>
								{/if}
							</div>
							<button
								type="button"
								role="switch"
								aria-checked={s.is_active === 1}
								aria-label={s.is_active === 1 ? 'Deactivate' : 'Activate'}
								onclick={() => toggleActive(s)}
								class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg {s.is_active ===
								1
									? 'bg-accent'
									: 'bg-bg-hover'}"
							>
								<span
									class="block h-3.5 w-3.5 rounded-full bg-bg-surface shadow-sm transition-transform {s.is_active ===
									1
										? 'translate-x-[18px]'
										: 'translate-x-[3px]'}"
								></span>
							</button>
							<button
								type="button"
								class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-text-muted hover:bg-bg-hover hover:text-text-primary transition-colors"
								aria-label="Edit"
								onclick={() => openEdit(s)}
							>
								<Pencil size={14} strokeWidth={1.75} />
							</button>
							<button
								type="button"
								class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-text-muted hover:bg-bg-hover hover:text-danger transition-colors"
								aria-label="Remove"
								onclick={() => remove(s)}
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
	title={editingId ? 'Edit allowed script' : 'Add allowed script'}
	description="Hostname only. No protocol, no path."
	size="md"
>
	<div class="flex flex-col gap-4">
		<Input
			label="Domain"
			placeholder="cdn.example.com"
			bind:value={formDomain}
			error={formError ?? undefined}
		/>
		<Input
			label="Purpose"
			placeholder="Analytics, payment, fonts..."
			hint="Why this domain is needed. Audit-friendly note."
			bind:value={formPurpose}
		/>
	</div>
	{#snippet footer()}
		<Button variant="secondary" onclick={() => (dialogOpen = false)} disabled={submitting}>
			Cancel
		</Button>
		<Button variant="primary" onclick={submit} loading={submitting}>
			{editingId ? 'Save' : 'Add'}
		</Button>
	{/snippet}
</Dialog>
