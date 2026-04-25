import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('admin-api: profile', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi, { withSeed: false });
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('GET returns empty profile shape when none exists', async ({ adminApi }) => {
		// blank-created site has no profile yet
		const res = await adminApi.get(u(`/api/sites/${site.id}/profile`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.site_id).toBe(site.id);
	});

	test('PUT creates a profile when absent', async ({ adminApi }) => {
		const businessName = `Acme ${rand()}`;
		const res = await adminApi.put(u(`/api/sites/${site.id}/profile`), {
			data: {
				business_name: businessName,
				country: 'SE',
				contact_email: 'hej@example.test',
				privacy_email: 'privacy@example.test',
				security_email: 'security@example.test',
				city: 'Stockholm'
			}
		});
		expect(res.ok()).toBe(true);
		const profile = await res.json();
		expect(profile.business_name).toBe(businessName);
		expect(profile.contact_email).toBe('hej@example.test');
	});

	test('PUT updates an existing profile', async ({ adminApi }) => {
		const res = await adminApi.put(u(`/api/sites/${site.id}/profile`), {
			data: {
				business_name: 'Renamed AB',
				country: 'NO',
				contact_email: 'updated@example.test'
			}
		});
		expect(res.ok()).toBe(true);
		const profile = await res.json();
		expect(profile.business_name).toBe('Renamed AB');
		expect(profile.country).toBe('NO');
	});

	test('PUT rejects invalid email', async ({ adminApi }) => {
		const res = await adminApi.put(u(`/api/sites/${site.id}/profile`), {
			data: { contact_email: 'not-an-email' }
		});
		expect(res.status()).toBe(400);
	});
});
