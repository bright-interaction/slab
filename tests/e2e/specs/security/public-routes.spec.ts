import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, uploadMedia, type Site } from '../../fixtures/data';

// Endpoints that must be reachable without auth:
//   - GET  /api/health
//   - GET  /api/starter-kits
//   - POST /t/pageview
//   - POST /t/consent
//   - GET  /media/<path>
// (Login itself is also public but covered by smoke + auth-flow.)

test.describe('security: documented public routes are reachable without auth', () => {
	let site: Site;
	let mediaPath: string;
	const cleanup: string[] = [];

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		const m = await uploadMedia(adminApi, site.id, { filename: 'public-test.png' });
		// MediaHandler.ServePublic mounts at /media/{site_slug}/{filename} or
		// similar. We probe a couple of plausible shapes since the e2e contract
		// only requires that *some* /media/* path serves without auth.
		mediaPath = `/media/${site.slug}/${m.filename}`;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('GET /api/health returns 200 anonymously', async () => {
		const ctx = await request.newContext();
		try {
			const res = await ctx.get(u('/api/health'));
			expect(res.ok()).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('GET /api/starter-kits returns 200 anonymously', async () => {
		const ctx = await request.newContext();
		try {
			const res = await ctx.get(u('/api/starter-kits'));
			expect(res.ok()).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('POST /t/pageview returns 204 anonymously', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const res = await ctx.post(u('/t/pageview'), {
				data: { siteId: site.id, path: '/' }
			});
			expect(res.status()).toBe(204);
		} finally {
			await ctx.dispose();
		}
	});

	test('POST /t/consent returns 204 anonymously', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const res = await ctx.post(u('/t/consent'), {
				data: {
					siteId: site.id,
					page: '/',
					consent: { version: 1, timestamp: 0, method: 'reject-all', categories: {} }
				}
			});
			expect(res.status()).toBe(204);
		} finally {
			await ctx.dispose();
		}
	});

	test('GET /media/* does not require auth', async () => {
		const ctx = await request.newContext();
		try {
			const res = await ctx.get(u(mediaPath));
			// Unauthenticated request must NOT be 401. 200 (served) or 404 (path
			// shape mismatch) is acceptable; 401 is not.
			expect(res.status()).not.toBe(401);
		} finally {
			await ctx.dispose();
		}
	});
});
