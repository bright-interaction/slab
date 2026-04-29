import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('admin-api: trusted-domain kind routes into the right CSP directive', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	async function add(adminApi: typeof site extends never ? never : import('@playwright/test').APIRequestContext, kind: string, domain: string) {
		const res = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain, purpose: `${kind} test`, kind }
		});
		expect(res.status()).toBe(201);
		return res.json();
	}

	test('POST without kind defaults to script', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain: 'js.stripe.com', purpose: 'payments' }
		});
		expect(res.status()).toBe(201);
		const body = await res.json();
		expect(body.kind).toBe('script');
	});

	test('POST kind=frame round-trips on the row', async ({ adminApi }) => {
		const created = await add(adminApi as never, 'frame', 'cal.com');
		expect(created.kind).toBe('frame');
		const list = await adminApi.get(u(`/api/sites/${site.id}/allowed-scripts`));
		const rows = await list.json();
		const row = rows.find((r: { id: string }) => r.id === created.id);
		expect(row.kind).toBe('frame');
	});

	test('POST rejects an invalid kind', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain: 'bad.example.com', kind: 'nonsense' }
		});
		expect(res.status()).toBe(400);
	});

	test('PATCH can move a row to a different kind', async ({ adminApi }) => {
		const created = await add(adminApi as never, 'image', 'res.cloudinary.com');
		const patch = await adminApi.patch(
			u(`/api/sites/${site.id}/allowed-scripts/${created.id}`),
			{ data: { kind: 'media' } }
		);
		expect(patch.ok()).toBe(true);
		const list = await adminApi.get(u(`/api/sites/${site.id}/allowed-scripts`));
		const rows = await list.json();
		const row = rows.find((r: { id: string }) => r.id === created.id);
		expect(row.kind).toBe('media');
	});

	test('PATCH rejects an invalid kind', async ({ adminApi }) => {
		const created = await add(adminApi as never, 'connect', 'api.example.com');
		const res = await adminApi.patch(
			u(`/api/sites/${site.id}/allowed-scripts/${created.id}`),
			{ data: { kind: 'nope' } }
		);
		expect(res.status()).toBe(400);
	});

	test('Frame-kind domain lands in frame-src and not script-src in the resolved CSP', async ({
		adminApi
	}) => {
		// Ensure CSP is enabled (default true), then add a frame-only domain.
		await add(adminApi as never, 'frame', 'embed.example.com');

		const res = await adminApi.get(
			u(`/api/sites/${site.id}/settings/security/preview`)
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		const csp: string = body.csp;
		expect(csp).toContain('frame-src');
		expect(csp).toContain('embed.example.com');
		// frame-only domain must NOT appear in script-src
		const scriptSrcMatch = /script-src ([^;]+)(?:;|$)/.exec(csp);
		expect(scriptSrcMatch).not.toBeNull();
		if (scriptSrcMatch) {
			expect(scriptSrcMatch[1]).not.toContain('embed.example.com');
		}
	});

	test('Resolved CSP includes upgrade-insecure-requests + frame-ancestors none by default', async ({
		adminApi
	}) => {
		const res = await adminApi.get(
			u(`/api/sites/${site.id}/settings/security/preview`)
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.csp).toContain('upgrade-insecure-requests');
		expect(body.csp).toContain("frame-ancestors 'none'");
		expect(body.hsts).toContain('max-age=');
		expect(body.cross_origin_opener_policy).toBe('same-origin');
	});
});
