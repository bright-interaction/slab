import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('admin-api: agent-keys', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('POST generates a key (revealed once)', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/agent-keys`), {
			data: { name: rand('key'), capabilities: ['read', 'write'] }
		});
		expect(res.status()).toBe(201);
		const body = await res.json();
		expect(body.id).toBeTruthy();
		// Phase 23 (2026-04-30) renamed the key prefix from `ask_` to
		// `atomic_` and standardised on 64 hex chars.
		expect(body.key).toMatch(/^atomic_[0-9a-f]{64}$/);
		expect(body.note).toMatch(/cannot be retrieved/i);
	});

	test('GET lists keys (no raw key returned)', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/agent-keys`), {
			data: { name: rand('list'), capabilities: ['read'] }
		});
		const created = await create.json();
		const res = await adminApi.get(u(`/api/sites/${site.id}/agent-keys`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
		const row = list.find((k: { id: string }) => k.id === created.id);
		expect(row).toBeTruthy();
		// list query intentionally excludes key + key_hash
		expect(row.key).toBeUndefined();
		expect(row.key_hash).toBeUndefined();
		expect(row.name).toBeTruthy();
		expect(row.capabilities).toBeTruthy();
	});

	test('DELETE revokes a key; revoked key returns 401 on /api/agent/*', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/agent-keys`), {
			data: { name: rand('revoke'), capabilities: ['read', 'write'] }
		});
		const { id, key } = await create.json();

		// Sanity: key works before revoke
		const ctxOk = await request.newContext({
			extraHTTPHeaders: { 'X-Agent-Key': key }
		});
		try {
			const ok = await ctxOk.get(u('/api/agent/context'));
			expect(ok.status()).toBe(200);
		} finally {
			await ctxOk.dispose();
		}

		// Revoke
		const del = await adminApi.delete(u(`/api/sites/${site.id}/agent-keys/${id}`));
		expect(del.ok()).toBe(true);
		const body = await del.json();
		expect(body.status).toBe('revoked');

		// Now the key should be unauthorized
		const ctxRevoked = await request.newContext({
			extraHTTPHeaders: { 'X-Agent-Key': key }
		});
		try {
			const after = await ctxRevoked.get(u('/api/agent/context'));
			expect(after.status()).toBe(401);
		} finally {
			await ctxRevoked.dispose();
		}
	});
});
