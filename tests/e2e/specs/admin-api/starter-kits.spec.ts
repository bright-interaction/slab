import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';

test.describe('admin-api: starter-kits (public catalog)', () => {
	test('GET /api/starter-kits requires no auth and returns array', async () => {
		const ctx = await request.newContext();
		try {
			const res = await ctx.get(u('/api/starter-kits'));
			expect(res.ok()).toBe(true);
			const list = await res.json();
			expect(Array.isArray(list)).toBe(true);
			// Each entry has the catalog shape
			for (const k of list) {
				expect(typeof k.id).toBe('string');
				expect(typeof k.name).toBe('string');
				expect(typeof k.description).toBe('string');
				expect(Array.isArray(k.target_site_types)).toBe(true);
			}
		} finally {
			await ctx.dispose();
		}
	});

	test('cookie-auth context also returns the catalog', async ({ adminApi }) => {
		const res = await adminApi.get(u('/api/starter-kits'));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
	});
});
