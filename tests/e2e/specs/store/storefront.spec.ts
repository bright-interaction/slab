import { test, expect } from '../../fixtures/auth';
import { createBlock, createSite, deleteSite, triggerBuildAndWait, u } from '../../fixtures/data';
import { createPublishedPage, distExists, readDist } from '../build-pipeline/_helpers';

// 2026-05-22: Sprint 2 slice C public storefront blocks.
//
// The four blocks (product_grid, product_detail, cart_drawer,
// checkout_form) render at build time via Go. We trigger a real build
// and assert the dist output: cards, Add-to-cart snapshots, cart
// drawer shell, checkout form data-attributes, and the storefront
// island JS asset. Cart click behaviour + checkout submit happen in
// the storefront_island.go unit-locked contract test, not here.

test.describe('Store storefront blocks', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('product_grid + cart_drawer + checkout_form render into the built site, storefront island ships', async ({
		adminApi
	}) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		// Seed catalog: two active products with one variant each.
		const teeProd = await (
			await adminApi.post(u(`/api/sites/${site.id}/products`), {
				data: {
					name: 'Storefront Tee',
					slug: 'storefront-tee',
					status: 'active',
					base_price_cents: 2500,
					currency: 'EUR'
				}
			})
		).json();
		const teeVar = await (
			await adminApi.post(u(`/api/sites/${site.id}/products/${teeProd.id}/variants`), {
				data: { name: 'Default', price_cents: 2500, inventory_count: 20 }
			})
		).json();

		const capProd = await (
			await adminApi.post(u(`/api/sites/${site.id}/products`), {
				data: {
					name: 'Storefront Cap',
					slug: 'storefront-cap',
					status: 'active',
					base_price_cents: 1500,
					currency: 'EUR'
				}
			})
		).json();
		await adminApi.post(u(`/api/sites/${site.id}/products/${capProd.id}/variants`), {
			data: { name: 'Default', price_cents: 1500, inventory_count: 20 }
		});

		// Catalog page: hero + product_grid + cart_drawer.
		const catalog = await createPublishedPage(adminApi, site.id, {
			slug: 'shop',
			title: 'Shop'
		});
		await createBlock(adminApi, site.id, catalog.id, {
			block_type: 'product_grid',
			data: { heading: 'All products', columns: '3', limit: 12, cta_label: 'Add to cart' },
			sort_order: 1
		});
		await createBlock(adminApi, site.id, catalog.id, {
			block_type: 'cart_drawer',
			data: {
				trigger_label: 'Bag',
				empty_message: 'Bag is empty.',
				checkout_url: '/checkout'
			},
			sort_order: 2
		});

		// Product detail page.
		const detail = await createPublishedPage(adminApi, site.id, {
			slug: 'products/storefront-tee',
			title: 'Storefront Tee'
		});
		await createBlock(adminApi, site.id, detail.id, {
			block_type: 'product_detail',
			data: { product_slug: 'storefront-tee', cta_label: 'Add to cart' }
		});

		// Checkout page.
		const checkout = await createPublishedPage(adminApi, site.id, {
			slug: 'checkout',
			title: 'Checkout'
		});
		await createBlock(adminApi, site.id, checkout.id, {
			block_type: 'checkout_form',
			data: {
				heading: 'Pay now',
				return_url: '/thank-you',
				submit_label: 'Place order',
				require_shipping_address: true
			}
		});

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		// Catalog page rendered with both products + cart drawer shell.
		const catalogHTML = readDist(site.id, 'shop/index.html');
		expect(catalogHTML).toContain('Storefront Tee');
		expect(catalogHTML).toContain('Storefront Cap');
		expect(catalogHTML).toContain('data-cart-add');
		expect(catalogHTML).toContain(`&quot;variant_id&quot;:&quot;${teeVar.id}&quot;`);
		expect(catalogHTML).toContain('data-cart-drawer');
		expect(catalogHTML).toContain('data-cart-open');
		expect(catalogHTML).toContain('Bag is empty.');

		// Storefront island is referenced from every page.
		expect(catalogHTML).toContain('/_atomic-storefront.js');

		// The asset file landed in public/ -> dist/.
		expect(distExists(site.id, '_atomic-storefront.js')).toBe(true);
		const islandJS = readDist(site.id, '_atomic-storefront.js');
		expect(islandJS).toContain('data-cart-add');
		expect(islandJS).toContain('data-checkout-form');
		expect(islandJS).toContain(site.id);
		expect(islandJS).toContain(`/api/sites/`);

		// Product detail page rendered with name + Add-to-cart anchor.
		const detailHTML = readDist(site.id, 'products/storefront-tee/index.html');
		expect(detailHTML).toContain('data-product-detail');
		expect(detailHTML).toContain('Storefront Tee');
		expect(detailHTML).toContain('data-cart-add');

		// Checkout page rendered with form + site-id baked in.
		const checkoutHTML = readDist(site.id, 'checkout/index.html');
		expect(checkoutHTML).toContain('data-checkout-form');
		expect(checkoutHTML).toContain(`data-site-id="${site.id}"`);
		expect(checkoutHTML).toContain('data-return-url="/thank-you"');
		expect(checkoutHTML).toContain('name="email"');
		expect(checkoutHTML).toContain('name="ship_street"');
		expect(checkoutHTML).toContain('Place order');
	});

	test('product_detail with missing slug emits the editor-only notice rather than blanking the page', async ({
		adminApi
	}) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		const page = await createPublishedPage(adminApi, site.id, {
			slug: 'broken-product',
			title: 'Broken product'
		});
		await createBlock(adminApi, site.id, page.id, {
			block_type: 'product_detail',
			data: { product_slug: 'does-not-exist' }
		});

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		const html = readDist(site.id, 'broken-product/index.html');
		expect(html).toContain('Product not found');
		expect(html).toContain('does-not-exist');
	});

	test('end-to-end: visitor cart payload reaches /checkout and returns a pending order', async ({
		adminApi,
		request
	}) => {
		test.setTimeout(60_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		const prod = await (
			await adminApi.post(u(`/api/sites/${site.id}/products`), {
				data: {
					name: 'Wire-flow product',
					slug: 'wire-flow',
					status: 'active',
					base_price_cents: 4000,
					currency: 'EUR'
				}
			})
		).json();
		const variant = await (
			await adminApi.post(u(`/api/sites/${site.id}/products/${prod.id}/variants`), {
				data: { name: 'Default', price_cents: 4000, inventory_count: 5 }
			})
		).json();

		// Same JSON shape the storefront island posts.
		const resp = await request.post(u(`/api/sites/${site.id}/checkout`), {
			data: {
				items: [{ variant_id: variant.id, quantity: 2 }],
				customer: { name: 'Storefront E2E', email: 'storefront@example.com' },
				shipping_address: {
					street: 'Main 1',
					city: 'Stockholm',
					postal_code: '11122',
					country: 'SE'
				},
				return_url: '/thank-you',
				locale: 'sv'
			}
		});
		expect(resp.status()).toBe(201);
		const body = await resp.json();
		expect(body.order.status).toBe('pending');
		expect(body.order.total_cents).toBe(8000);
		expect(body.order.currency).toBe('EUR');
	});
});
