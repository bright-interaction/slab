import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand } from '../../fixtures/data';

test.describe('negative: slug uniqueness', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('two seed calls with same slug -> second returns 409', async ({ adminApi }) => {
		const slug = rand('uniq');
		const first = await createSite(adminApi, { slug });
		cleanup.push(first.id);

		const res = await adminApi.post(u('/api/sites/seed'), {
			data: {
				type: 'b2b',
				info: {
					name: 'Dup',
					slug,
					domain: `${slug}-2.e2e.test`,
					lang: 'en',
					business_name: 'Dup AB',
					contact_email: 'dup@example.test',
					country: 'SE'
				},
				structure: { structure_type: 'one-pager', max_depth: 3 },
				silos: [],
				branding: {
					primary_color: '#000000',
					secondary_color: '#111111',
					bg_color: '#ffffff',
					text_color: '#000000',
					font_heading: 'Inter',
					font_body: 'Inter'
				}
			}
		});
		expect(res.status()).toBe(409);
		const body = await res.json();
		expect(body.error).toMatch(/slug/i);
	});

	test('two blank /api/sites POSTs with same slug -> second returns 409', async ({ adminApi }) => {
		const slug = rand('blank-uniq');
		const a = await adminApi.post(u('/api/sites'), {
			data: { name: 'A', slug, domain: `${slug}.e2e.test` }
		});
		expect(a.status()).toBe(201);
		const aBody = await a.json();
		cleanup.push(aBody.id);

		const b = await adminApi.post(u('/api/sites'), {
			data: { name: 'B', slug, domain: `${slug}-other.e2e.test` }
		});
		expect(b.status()).toBe(409);
	});
});
