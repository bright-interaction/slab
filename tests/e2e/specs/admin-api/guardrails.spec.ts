import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('admin-api: guardrails CRUD', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	async function createRule(adminApi: import('@playwright/test').APIRequestContext, overrides: Record<string, unknown> = {}) {
		const res = await adminApi.post(u(`/api/sites/${site.id}/guardrails`), {
			data: {
				rule_type: 'forbid_pattern',
				target: 'block.data',
				value: rand('pat'),
				severity: 'error',
				...overrides
			}
		});
		expect(res.status()).toBe(201);
		return res.json();
	}

	test('POST creates a guardrail rule', async ({ adminApi }) => {
		const r = await createRule(adminApi);
		expect(r.id).toBeTruthy();
		expect(r.rule_type).toBe('forbid_pattern');
		expect(r.severity).toBe('error');
	});

	test('GET lists rules and returns expected shape', async ({ adminApi }) => {
		const r = await createRule(adminApi);
		const res = await adminApi.get(u(`/api/sites/${site.id}/guardrails`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
		const found = list.find((x: { id: string }) => x.id === r.id);
		expect(found).toBeTruthy();
		expect(found.rule_type).toBeTruthy();
		expect(found.target).toBeTruthy();
		expect(found.value).toBeTruthy();
		expect(found.severity).toBeTruthy();
	});

	test('GET /{id} returns one rule', async ({ adminApi }) => {
		const r = await createRule(adminApi);
		const res = await adminApi.get(u(`/api/sites/${site.id}/guardrails/${r.id}`));
		expect(res.ok()).toBe(true);
		const got = await res.json();
		expect(got.id).toBe(r.id);
	});

	test('PATCH updates a rule', async ({ adminApi }) => {
		const r = await createRule(adminApi);
		const res = await adminApi.patch(u(`/api/sites/${site.id}/guardrails/${r.id}`), {
			data: { severity: 'warning' }
		});
		expect(res.ok()).toBe(true);
		const updated = await res.json();
		expect(updated.severity).toBe('warning');
	});

	test('DELETE removes a rule', async ({ adminApi }) => {
		const r = await createRule(adminApi);
		const res = await adminApi.delete(u(`/api/sites/${site.id}/guardrails/${r.id}`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('deleted');
	});
});
