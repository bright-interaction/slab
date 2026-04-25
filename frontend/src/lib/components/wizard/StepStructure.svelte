<script lang="ts">
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import {
		updateSilos,
		updateStructure,
		wizard,
		deriveSlug
	} from '$lib/stores/wizard.svelte';
	import type { SeedSiteSilo, StructureType } from '$lib/api/sites';

	interface StructureOption {
		value: StructureType;
		title: string;
		description: string;
		preview: string[];
	}

	const options: StructureOption[] = [
		{
			value: 'one-pager',
			title: 'One-pager',
			description: 'A single page with anchor sections.',
			preview: ['/']
		},
		{
			value: 'soft-silo',
			title: 'Soft silo',
			description: 'Topical clusters with cross-links between silos.',
			preview: ['/services/seo', '/services/web', '/about', '/contact']
		},
		{
			value: 'hard-silo',
			title: 'Hard silo',
			description: 'Strict topical clusters, no cross-silo links.',
			preview: ['/services/seo/audits', '/services/seo/local', '/about']
		}
	];

	const initialSilos: SeedSiteSilo[] =
		wizard.value.silos.length > 0
			? [...wizard.value.silos]
			: [
					{ name: 'Services', slug_prefix: 'services' },
					{ name: 'About', slug_prefix: 'about' }
				];

	let silos = $state<SeedSiteSilo[]>(initialSilos);

	$effect(() => {
		updateSilos(silos);
	});

	const structureType = $derived(wizard.value.structure.structure_type);

	function selectStructure(value: StructureType): void {
		updateStructure({
			structure_type: value,
			max_depth: value === 'one-pager' ? 1 : 3
		});
	}

	function updateSilo(idx: number, field: 'name' | 'slug_prefix', value: string): void {
		const next = [...silos];
		const existing = next[idx];
		if (!existing) return;
		const updated: SeedSiteSilo = { ...existing, [field]: value };
		if (field === 'name' && existing.slug_prefix === deriveSlug(existing.name)) {
			updated.slug_prefix = deriveSlug(value);
		}
		next[idx] = updated;
		silos = next;
	}

	function addSilo(): void {
		if (silos.length >= 5) return;
		silos = [...silos, { name: '', slug_prefix: '' }];
	}

	function removeSilo(idx: number): void {
		silos = silos.filter((_, i) => i !== idx);
	}

	const showSilos = $derived(structureType !== 'one-pager');

	const previewUrls = $derived.by(() => {
		if (structureType === 'one-pager') return ['/'];
		const items = silos
			.filter((s) => s.slug_prefix.trim().length > 0)
			.slice(0, 4)
			.map((s) => `/${s.slug_prefix.replace(/^\/+|\/+$/g, '')}`);
		if (items.length === 0) {
			return options.find((o) => o.value === structureType)?.preview ?? [];
		}
		return items;
	});
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-1">
		<h2 class="font-display text-2xl font-extralight tracking-tight text-text-primary">
			Site structure
		</h2>
		<p class="text-[13px] text-text-secondary">
			Pick how your URLs are organized. You can add more pages later.
		</p>
	</div>

	<div class="grid grid-cols-1 gap-3 md:grid-cols-3">
		{#each options as opt (opt.value)}
			{@const isActive = structureType === opt.value}
			<button
				type="button"
				onclick={() => selectStructure(opt.value)}
				aria-pressed={isActive}
				class="flex flex-col gap-3 rounded-xl border bg-bg-surface p-5 text-left transition-all duration-200 active:scale-[0.99] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg {isActive
					? 'border-accent bg-bg-elevated ring-1 ring-accent'
					: 'border-border-light hover:bg-bg-hover'}"
			>
				<div class="flex flex-col gap-1">
					<span class="text-[14px] font-medium text-text-primary">{opt.title}</span>
					<span class="text-[12px] text-text-secondary">{opt.description}</span>
				</div>
				<ul
					class="flex flex-col gap-1 rounded-lg border border-border-light bg-bg-elevated p-3 font-mono text-[11px] text-text-muted"
				>
					{#each opt.preview as url (url)}
						<li>{url}</li>
					{/each}
				</ul>
			</button>
		{/each}
	</div>

	{#if showSilos}
		<div class="flex flex-col gap-3">
			<div class="flex items-center justify-between">
				<div class="flex flex-col">
					<h3 class="text-[14px] font-medium text-text-primary">Silos</h3>
					<p class="text-[12px] text-text-secondary">
						Up to 5 top-level URL prefixes. Examples: services, blog, cases.
					</p>
				</div>
				<Button
					variant="secondary"
					size="sm"
					disabled={silos.length >= 5}
					onclick={addSilo}
				>
					Add silo
				</Button>
			</div>
			<div class="flex flex-col gap-3">
				{#each silos as silo, idx (idx)}
					<div
						class="grid grid-cols-1 gap-3 rounded-xl border border-border-light bg-bg-surface p-4 md:grid-cols-[1fr_1fr_auto]"
					>
						<Input
							label="Name"
							placeholder="Services"
							value={silo.name}
							oninput={(e) =>
								updateSilo(idx, 'name', (e.currentTarget as HTMLInputElement).value)}
						/>
						<Input
							label="Slug prefix"
							placeholder="services"
							value={silo.slug_prefix}
							oninput={(e) =>
								updateSilo(
									idx,
									'slug_prefix',
									(e.currentTarget as HTMLInputElement).value
										.toLowerCase()
										.replace(/[^a-z0-9-]/g, '')
								)}
						/>
						<div class="flex items-end">
							<Button variant="ghost" size="sm" onclick={() => removeSilo(idx)}>
								Remove
							</Button>
						</div>
					</div>
				{/each}
				{#if silos.length === 0}
					<p class="text-[12px] text-text-muted">
						No silos yet. Add one to organize your URLs.
					</p>
				{/if}
			</div>
		</div>
	{/if}

	<div class="flex flex-col gap-2 rounded-xl border border-border-light bg-bg-elevated p-4">
		<span class="text-[12px] font-medium text-text-secondary">URL preview</span>
		<ul class="flex flex-col gap-1 font-mono text-[12px] text-text-primary">
			{#each previewUrls as url (url)}
				<li>{url}</li>
			{/each}
		</ul>
	</div>
</div>
