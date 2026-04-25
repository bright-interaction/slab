<script lang="ts">
	import { page as pageState } from '$app/state';
	import { Copy, KeyRound, ShieldAlert, Trash2 } from 'lucide-svelte';
	import * as agentKeysApi from '$lib/api/agentKeys';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Checkbox from '$lib/components/ui/Checkbox.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { AgentKey } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);

	type Capability = 'read' | 'write' | 'build' | 'deploy' | 'media';

	const ALL_CAPS: Array<{ value: Capability; label: string; help: string }> = [
		{ value: 'read', label: 'read', help: 'Fetch site context, pages, blocks, components.' },
		{ value: 'write', label: 'write', help: 'Create and edit pages, blocks, components, classes.' },
		{ value: 'build', label: 'build', help: 'Trigger production builds.' },
		{ value: 'deploy', label: 'deploy', help: 'Push builds to the deploy target.' },
		{ value: 'media', label: 'media', help: 'Upload and manage media assets.' }
	];

	let keys = $state<AgentKey[]>([]);
	let loading = $state<boolean>(true);
	let loadError = $state<string | null>(null);

	let createOpen = $state<boolean>(false);
	let creating = $state<boolean>(false);
	let newName = $state<string>('');
	let newCaps = $state<Record<Capability, boolean>>({
		read: true,
		write: true,
		build: false,
		deploy: false,
		media: false
	});

	let revealOpen = $state<boolean>(false);
	let revealedKey = $state<string>('');
	let revealedName = $state<string>('');
	let saved = $state<boolean>(false);
	let copied = $state<boolean>(false);

	async function loadKeys(id: string): Promise<void> {
		loading = true;
		loadError = null;
		try {
			const list = await agentKeysApi.list(id);
			keys = list ?? [];
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load agent keys';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void loadKeys(siteID);
	});

	function openCreateDialog(): void {
		newName = '';
		newCaps = { read: true, write: true, build: false, deploy: false, media: false };
		createOpen = true;
	}

	function selectedCaps(): Capability[] {
		return (Object.keys(newCaps) as Capability[]).filter((k) => newCaps[k]);
	}

	async function generate(): Promise<void> {
		const name = newName.trim();
		if (name.length < 2) {
			toast.error('Name must be at least 2 characters.');
			return;
		}
		const caps = selectedCaps();
		if (caps.length === 0) {
			toast.error('Pick at least one capability.');
			return;
		}
		creating = true;
		try {
			const res = await agentKeysApi.create(siteID, { name, capabilities: caps });
			createOpen = false;
			revealedKey = res.key;
			revealedName = res.name;
			saved = false;
			copied = false;
			revealOpen = true;
			toast.success('Agent key generated.');
			await loadKeys(siteID);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to generate key');
		} finally {
			creating = false;
		}
	}

	async function copyKey(): Promise<void> {
		if (!revealedKey) return;
		try {
			await navigator.clipboard.writeText(revealedKey);
			copied = true;
			toast.success('Key copied to clipboard.');
		} catch {
			toast.error('Copy failed. Select the text and copy manually.');
		}
	}

	function closeReveal(): void {
		revealOpen = false;
		revealedKey = '';
		revealedName = '';
		saved = false;
		copied = false;
	}

	async function revoke(k: AgentKey): Promise<void> {
		const ok = await confirm({
			title: `Revoke "${k.name}"?`,
			message:
				'Any agent using this key will immediately lose access. This cannot be undone.',
			confirmText: 'Revoke',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		const previous = keys;
		keys = keys.filter((x) => x.id !== k.id);
		try {
			await agentKeysApi.revoke(siteID, k.id);
			toast.success('Key revoked.');
		} catch (err) {
			keys = previous;
			toast.error(err instanceof Error ? err.message : 'Failed to revoke key');
		}
	}

	function parseCaps(json: string): string[] {
		if (!json) return [];
		try {
			const parsed = JSON.parse(json);
			return Array.isArray(parsed) ? (parsed as string[]) : [];
		} catch {
			return [];
		}
	}

	function relativeTime(ts: string): string {
		if (!ts) return 'never';
		const d = new Date(ts);
		if (Number.isNaN(d.getTime())) return 'never';
		// Treat zero/empty timestamps as never used
		if (d.getTime() <= 0) return 'never';
		const diff = Date.now() - d.getTime();
		const sec = Math.round(diff / 1000);
		if (sec < 60) return 'just now';
		const min = Math.round(sec / 60);
		if (min < 60) return `${min}m ago`;
		const hr = Math.round(min / 60);
		if (hr < 24) return `${hr}h ago`;
		const day = Math.round(hr / 24);
		if (day < 30) return `${day}d ago`;
		return d.toLocaleDateString();
	}

	function toggleCap(cap: Capability): void {
		newCaps = { ...newCaps, [cap]: !newCaps[cap] };
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex items-start justify-between gap-4">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Agent keys
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				API keys for AI agents that read and write to this site.
			</p>
		</div>
		<Button variant="primary" onclick={openCreateDialog}>Generate new key</Button>
	</header>

	<Card padding="md" class="mt-6">
		<div class="flex items-start gap-3">
			<span
				class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning"
			>
				<ShieldAlert class="h-4 w-4" />
			</span>
			<div class="flex-1 text-[13px] text-text-secondary">
				<p>
					Agent keys let any AI agent (Claude CLI, Cursor, etc.) read and write to this
					site via the Agent API. Treat them like passwords.
				</p>
				<p class="mt-1">
					Keys are shown <span class="font-medium text-text-primary">once</span> at creation.
					Save the value somewhere safe right after you generate it. Revoke any key you suspect
					has leaked.
				</p>
			</div>
		</div>
	</Card>

	<section class="mt-6">
		{#if loading}
			<div class="flex flex-col gap-2">
				{#each Array(3) as _, i (i)}
					<Card padding="sm">
						<div class="flex items-center gap-3">
							<Skeleton width="30%" height="0.95rem" />
							<Skeleton width="20%" height="0.7rem" />
							<Skeleton width="15%" height="0.7rem" />
						</div>
					</Card>
				{/each}
			</div>
		{:else if loadError}
			<Card padding="md">
				<p class="text-[13px] text-danger">{loadError}</p>
			</Card>
		{:else if keys.length === 0}
			<Card padding="none">
				<EmptyState
					title="No keys yet"
					description="Generate your first agent key to let an AI agent edit this site."
				>
					{#snippet icon()}
						<KeyRound class="h-6 w-6" />
					{/snippet}
					{#snippet action()}
						<Button variant="primary" onclick={openCreateDialog}>Generate first key</Button>
					{/snippet}
				</EmptyState>
			</Card>
		{:else}
			<Card padding="none">
				<div role="table" class="flex flex-col">
					<div
						role="row"
						class="grid grid-cols-[1.4fr_1.6fr_1fr_0.7fr_auto] items-center gap-4 border-b border-border-light px-4 py-2 text-[10px] font-medium uppercase tracking-wider text-text-muted"
					>
						<span role="columnheader">Name</span>
						<span role="columnheader">Capabilities</span>
						<span role="columnheader">Last used</span>
						<span role="columnheader">Status</span>
						<span role="columnheader" class="sr-only">Actions</span>
					</div>
					{#each keys as k (k.id)}
						{@const caps = parseCaps(k.capabilities)}
						<div
							role="row"
							class="group grid grid-cols-[1.4fr_1.6fr_1fr_0.7fr_auto] items-center gap-4 border-b border-border-light px-4 py-3 last:border-b-0 hover:bg-bg-hover/40"
						>
							<div role="cell" class="flex items-center gap-2 min-w-0">
								<KeyRound class="h-3.5 w-3.5 shrink-0 text-text-muted" />
								<span class="truncate text-[13px] font-medium text-text-primary">
									{k.name}
								</span>
							</div>
							<div role="cell" class="flex flex-wrap items-center gap-1.5">
								{#each caps as cap (cap)}
									<Badge variant="default">{cap}</Badge>
								{/each}
								{#if caps.length === 0}
									<span class="text-[12px] text-text-muted">none</span>
								{/if}
							</div>
							<div role="cell" class="text-[12px] text-text-muted">
								{relativeTime(k.last_used_at)}
							</div>
							<div role="cell">
								{#if k.is_active === 1}
									<Badge variant="success" dot>active</Badge>
								{:else}
									<Badge variant="danger" dot>revoked</Badge>
								{/if}
							</div>
							<div role="cell" class="flex justify-end">
								<button
									type="button"
									class="inline-flex h-8 w-8 items-center justify-center rounded-md text-text-muted opacity-0 transition-all group-hover:opacity-100 hover:bg-bg-hover hover:text-danger focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
									aria-label="Revoke key"
									onclick={() => void revoke(k)}
									disabled={k.is_active !== 1}
								>
									<Trash2 class="h-3.5 w-3.5" />
								</button>
							</div>
						</div>
					{/each}
				</div>
			</Card>
		{/if}
	</section>
</div>

<Dialog bind:open={createOpen} title="Generate agent key" size="md">
	<div class="flex flex-col gap-4">
		<Input
			label="Name"
			value={newName}
			oninput={(e) => {
				newName = (e.currentTarget as HTMLInputElement).value;
			}}
			placeholder="claude-cli on my laptop"
			hint="A label that helps you remember where this key is used."
		/>
		<div class="flex flex-col gap-2">
			<span class="text-[12px] font-medium text-text-secondary">Capabilities</span>
			<div class="flex flex-col gap-2 rounded-lg border border-border-light bg-bg-elevated/40 p-3">
				{#each ALL_CAPS as cap (cap.value)}
					<button
						type="button"
						class="flex w-full items-start gap-3 rounded-md p-1 text-left transition-colors hover:bg-bg-hover/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
						onclick={() => toggleCap(cap.value)}
					>
						<span class="mt-0.5 pointer-events-none">
							<Checkbox checked={newCaps[cap.value]} id={`cap-${cap.value}`} />
						</span>
						<span class="flex flex-col gap-0.5">
							<span class="text-[13px] font-medium text-text-primary">{cap.label}</span>
							<span class="text-[12px] text-text-muted">{cap.help}</span>
						</span>
					</button>
				{/each}
			</div>
		</div>
	</div>
	{#snippet footer()}
		<Button variant="secondary" onclick={() => (createOpen = false)}>Cancel</Button>
		<Button variant="primary" loading={creating} onclick={generate}>Generate</Button>
	{/snippet}
</Dialog>

<Dialog
	bind:open={revealOpen}
	title="Save this key now"
	description="This is the only time you will see this key. If you close this dialog without copying it, you will need to generate a new one."
	size="md"
	onOpenChange={(v) => {
		if (!v && !saved) {
			// Block closing until acknowledged
			revealOpen = true;
		} else if (!v) {
			closeReveal();
		}
	}}
>
	<div class="flex flex-col gap-4">
		<div class="flex flex-col gap-1">
			<span class="text-[12px] font-medium text-text-secondary">Key name</span>
			<span class="text-[13px] text-text-primary">{revealedName}</span>
		</div>
		<div class="flex flex-col gap-1">
			<span class="text-[12px] font-medium text-text-secondary">Key value</span>
			<div class="flex items-center gap-2">
				<input
					readonly
					value={revealedKey}
					class="h-9 w-full rounded-lg border border-border bg-bg-elevated px-3 font-mono text-[12px] text-text-primary focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent"
					onclick={(e) => (e.currentTarget as HTMLInputElement).select()}
					aria-label="Agent key value"
				/>
				<Button variant="secondary" onclick={() => void copyKey()}>
					{#snippet icon()}
						<Copy class="h-3.5 w-3.5" />
					{/snippet}
					{copied ? 'Copied' : 'Copy'}
				</Button>
			</div>
		</div>
		<div
			class="flex items-start gap-2 rounded-lg border border-warning/30 bg-warning/5 px-3 py-2.5"
		>
			<ShieldAlert class="mt-0.5 h-4 w-4 shrink-0 text-warning" />
			<p class="text-[12px] text-text-secondary">
				Store this key in a password manager or your shell's secret store. We hash it on the
				server and cannot recover the original value.
			</p>
		</div>
		<div class="flex items-center justify-between gap-3 border-t border-border-light pt-3">
			<Switch bind:checked={saved} id="reveal-saved" label="I have saved this key" />
		</div>
	</div>
	{#snippet footer()}
		<Button variant="primary" disabled={!saved} onclick={closeReveal}>Close</Button>
	{/snippet}
</Dialog>
