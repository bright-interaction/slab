import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

// FingerprintMiddleware (mounted on the /t/* group) sets a 16-hex
// slab_fp HttpOnly cookie when the request arrives without one. A
// follow-up request that echoes the cookie keeps the same value.

function readSetCookie(headers: Array<{ name: string; value: string }>, name: string): string | null {
	const sc = headers.filter((h) => h.name.toLowerCase() === 'set-cookie');
	for (const h of sc) {
		const m = h.value.match(new RegExp(`${name}=([^;]+)`));
		if (m) return m[1];
	}
	return null;
}

test.describe('analytics: fingerprint cookie lifecycle', () => {
	let site: Site;
	const cleanup: string[] = [];

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('first /t/pageview without cookie sets slab_fp; second request echoes the same value', async () => {
		// Use a fresh request context so we control which cookies go out.
		const ctx = await request.newContext({
			extraHTTPHeaders: {
				'Content-Type': 'application/json',
				'User-Agent': 'e2e-fp-test/1.0',
				'Accept-Language': 'en-US'
			}
		});
		try {
			const first = await ctx.post(u('/t/pageview'), {
				data: { siteId: site.id, path: '/', referrer: '' }
			});
			expect([204, 400, 500]).toContain(first.status());
			expect(first.status()).toBe(204);

			const fp1 = readSetCookie(first.headersArray(), 'slab_fp');
			expect(fp1, 'expected slab_fp cookie on first pageview').toBeTruthy();
			expect(fp1!).toMatch(/^[0-9a-f]{16}$/);

			// Second request, sending the cookie back -- server should NOT roll a
			// fresh cookie, and the visitor identity stays the same.
			const second = await ctx.post(u('/t/pageview'), {
				headers: { Cookie: `slab_fp=${fp1}` },
				data: { siteId: site.id, path: '/about', referrer: '/' }
			});
			expect(second.status()).toBe(204);
			const fp2 = readSetCookie(second.headersArray(), 'slab_fp');
			// Either nothing was set (server trusted the inbound cookie) OR if the
			// server did echo, it must echo the SAME value. Both are acceptable.
			if (fp2 !== null) expect(fp2).toBe(fp1);
		} finally {
			await ctx.dispose();
		}
	});

	test('malformed slab_fp cookie is replaced with a fresh value', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const res = await ctx.post(u('/t/pageview'), {
				headers: { Cookie: `slab_fp=NOT-HEX-VALUE` },
				data: { siteId: site.id, path: '/' }
			});
			expect(res.status()).toBe(204);
			const fp = readSetCookie(res.headersArray(), 'slab_fp');
			expect(fp).toBeTruthy();
			expect(fp!).toMatch(/^[0-9a-f]{16}$/);
		} finally {
			await ctx.dispose();
		}
	});
});
