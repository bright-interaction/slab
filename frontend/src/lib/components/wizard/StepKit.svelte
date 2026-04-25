<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import * as starterKitsApi from '$lib/api/starterKits';
	import type { StarterKit } from '$lib/api/starterKits';
	import { setStarterKit, wizard } from '$lib/stores/wizard.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	let kits = $state<StarterKit[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	const currentType = $derived(wizard.value.type);
	const selectedKit = $derived(wizard.value.starterKit);

	const filteredKits = $derived.by<StarterKit[]>(() => {
		if (!currentType) return kits;
		return kits.filter((k) =>
			Array.isArray(k.target_site_types) && k.target_site_types.includes(currentType)
		);
	});

	onMount(async () => {
		try {
			const list = await starterKitsApi.list();
			kits = Array.isArray(list) ? list : [];
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load starter kits.';
			kits = [];
		} finally {
			loading = false;
		}
	});

	function pickKit(kit: StarterKit): void {
		setStarterKit(kit.id);
		void goto('/sites/new/wizard/info');
	}

	function skipKit(): void {
		setStarterKit(null);
		void goto('/sites/new/wizard/info');
	}

	function isSelected(kit: StarterKit): boolean {
		return selectedKit === kit.id;
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-1">
		<h2 class="font-display text-2xl font-extralight tracking-tight text-text-primary">
			Pick a starter kit
		</h2>
		<p class="text-[13px] text-text-secondary">
			We'll preload your site with components, CSS, and content. You can always change everything later.
		</p>
	</div>

	{#if loading}
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each Array.from({ length: 3 }) as _, i (i)}
				<div class="flex flex-col gap-3 rounded-xl border border-border-light bg-bg-surface p-5">
					<Skeleton width="60%" height="1.5rem" />
					<Skeleton width="100%" height="0.75rem" />
					<Skeleton width="80%" height="0.75rem" />
					<Skeleton width="40%" height="1rem" />
				</div>
			{/each}
		</div>
	{:else if loadError}
		<EmptyState
			title="Couldn't load starter kits"
			description={loadError}
		>
			{#snippet action()}
				<Button variant="primary" onclick={skipKit}>Skip kit, start blank</Button>
			{/snippet}
		</EmptyState>
	{:else if filteredKits.length === 0}
		<EmptyState
			title="No kits for this site type yet"
			description="We don't have a starter kit that matches your site type. You can start blank and add components later."
		>
			{#snippet action()}
				<Button variant="primary" onclick={skipKit}>Skip kit, start blank</Button>
			{/snippet}
		</EmptyState>
	{:else}
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each filteredKits as kit (kit.id)}
				<button
					type="button"
					class="group relative flex flex-col gap-3 rounded-xl border bg-bg-surface p-5 text-left transition-all duration-200 hover:-translate-y-0.5 hover:shadow-sm active:scale-[0.99] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg {isSelected(
						kit
					)
						? 'border-accent bg-bg-elevated ring-1 ring-accent'
						: 'border-border-light hover:bg-bg-hover'}"
					onclick={() => pickKit(kit)}
					aria-pressed={isSelected(kit)}
				>
					<div class="flex flex-col gap-1.5">
						<span class="font-display text-2xl font-extralight tracking-tight text-text-primary">
							{kit.name}
						</span>
						<span class="text-[13px] text-text-secondary">{kit.description}</span>
					</div>
					{#if kit.target_site_types && kit.target_site_types.length > 0}
						<div class="mt-auto flex flex-wrap gap-1.5 pt-2">
							{#each kit.target_site_types as t (t)}
								<Badge variant="default">{t}</Badge>
							{/each}
						</div>
					{/if}
				</button>
			{/each}

			<button
				type="button"
				class="group relative flex flex-col items-start justify-between gap-3 rounded-xl border border-dashed bg-transparent p-5 text-left transition-all duration-200 hover:-translate-y-0.5 active:scale-[0.99] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg {selectedKit ===
				null
					? 'border-accent bg-bg-elevated ring-1 ring-accent'
					: 'border-border hover:bg-bg-hover'}"
				onclick={skipKit}
				aria-pressed={selectedKit === null}
			>
				<div class="flex flex-col gap-1.5">
					<span class="font-display text-2xl font-extralight tracking-tight text-text-primary">
						Skip kit, start blank
					</span>
					<span class="text-[13px] text-text-secondary">
						Begin with an empty site. Add components, content, and styling yourself.
					</span>
				</div>
			</button>
		</div>
	{/if}
</div>
