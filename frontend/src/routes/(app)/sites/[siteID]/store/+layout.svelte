<script lang="ts">
	import { page } from '$app/state';
	let { children } = $props();
	const siteID = $derived(page.params.siteID as string);

	const subTabs = $derived([
		{ id: 'products', label: 'Products', href: `/sites/${siteID}/store/products` },
		{ id: 'orders', label: 'Orders', href: `/sites/${siteID}/store/orders` },
		{ id: 'discount-codes', label: 'Discount codes', href: `/sites/${siteID}/store/discount-codes` }
	]);

	function isActive(href: string): boolean {
		const path = page.url.pathname;
		return path === href || path.startsWith(`${href}/`);
	}
</script>

<div class="mx-auto max-w-7xl px-6 pt-6">
	<div class="mb-4">
		<h1 class="text-2xl font-light tracking-tight text-text-primary">Store</h1>
		<p class="mt-1 text-sm text-text-muted">
			Catalog, variants, inventory, and promo codes. Orders and checkout are in
			Slice B of the roadmap.
		</p>
	</div>
	<div class="border-b border-border-light">
		<div class="flex items-center gap-1">
			{#each subTabs as t (t.id)}
				{@const active = isActive(t.href)}
				<a
					href={t.href}
					class="relative -mb-px inline-flex items-center px-3 py-2 text-[13px] transition-colors {active
						? 'text-text-primary border-b-2 border-accent font-medium'
						: 'text-text-muted hover:text-text-primary border-b-2 border-transparent'}"
					aria-current={active ? 'page' : undefined}
				>
					{t.label}
				</a>
			{/each}
		</div>
	</div>
</div>

<div class="mx-auto max-w-7xl px-6 py-6">
	{@render children()}
</div>
