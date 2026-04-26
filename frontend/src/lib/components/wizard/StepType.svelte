<script lang="ts">
	import { goto } from '$app/navigation';
	import { setType, wizard } from '$lib/stores/wizard.svelte';
	import type { SiteType } from '$lib/api/sites';

	interface TypeOption {
		value: SiteType | 'ecommerce';
		title: string;
		description: string;
		icon: string;
		disabled?: boolean;
		comingSoon?: boolean;
	}

	const options: TypeOption[] = [
		{
			value: 'b2b',
			title: 'B2B',
			description: 'Business, law firm, agency',
			icon: 'M3.75 21h16.5M4.5 3h15M5.25 3v18M18.75 3v18M9 6.75h1.5M9 12h1.5M9 17.25h1.5M13.5 6.75H15M13.5 12H15M13.5 17.25H15'
		},
		{
			value: 'b2c',
			title: 'B2C',
			description: 'Local business or consumer brand',
			icon: 'M2.25 21h19.5M2.25 9h19.5m-16.5 0v12m13.5-12v12M9 21V9m6 12V9'
		},
		{
			value: 'personal',
			title: 'Personal',
			description: 'Portfolio or personal site',
			icon: 'M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z'
		},
		{
			value: 'blog',
			title: 'Blog',
			description: 'Content-led site or magazine',
			icon: 'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25M9 16.5v.75m3-3v3M15 12v5.25m-4.5-15H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z'
		},
		{
			value: 'agency',
			title: 'Agency',
			description: 'Multi-client agency site',
			icon: 'M2.25 21h19.5m-18-18v18m10.5-18v18m6-13.5V21M6.75 6.75h.75m-.75 3h.75m-.75 3h.75m3-6h.75m-.75 3h.75m-.75 3h.75M6.75 21v-3.375c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21'
		},
		{
			value: 'one-pager',
			title: 'One-pager',
			description: 'Single-page launch site',
			icon: 'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z'
		},
		{
			value: 'ecommerce',
			title: 'E-commerce',
			description: 'Storefront with cart and Stripe Checkout',
			icon: 'M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 00-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 00-16.536-1.84M7.5 14.25L5.106 5.272M6 20.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zm12.75 0a.75.75 0 11-1.5 0 .75.75 0 011.5 0z'
		}
	];

	function selectType(opt: TypeOption): void {
		if (opt.disabled) return;
		setType(opt.value as SiteType);
		void goto('/sites/new/wizard/kit');
	}

	function isSelected(opt: TypeOption): boolean {
		return wizard.value.type === opt.value;
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-1">
		<h2 class="font-display text-2xl font-extralight tracking-tight text-text-primary">
			What kind of site are you building?
		</h2>
		<p class="text-[13px] text-text-secondary">
			Pick the closest fit. We tune the wizard, defaults, and starter content based on this.
		</p>
	</div>

	<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
		{#each options as opt (opt.value)}
			<button
				type="button"
				class="group relative flex flex-col gap-3 rounded-xl border bg-bg-surface p-5 text-left transition-all duration-200 active:scale-[0.99] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg {opt.disabled
					? 'cursor-not-allowed border-border-light opacity-50'
					: isSelected(opt)
						? 'border-accent bg-bg-elevated ring-1 ring-accent'
						: 'border-border-light hover:bg-bg-hover'}"
				disabled={opt.disabled}
				onclick={() => selectType(opt)}
				aria-pressed={isSelected(opt)}
			>
				<div class="flex items-start justify-between gap-3">
					<svg
						class="h-5 w-5 text-text-secondary"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="1.5"
						stroke="currentColor"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d={opt.icon} />
					</svg>
					{#if opt.comingSoon}
						<span
							class="rounded-full border border-border bg-bg-elevated px-2 py-0.5 text-[10px] font-medium text-text-muted"
						>
							Coming soon
						</span>
					{/if}
				</div>
				<div class="flex flex-col gap-1">
					<span class="text-[14px] font-medium text-text-primary">{opt.title}</span>
					<span class="text-[12px] text-text-secondary">{opt.description}</span>
				</div>
			</button>
		{/each}
	</div>
</div>
