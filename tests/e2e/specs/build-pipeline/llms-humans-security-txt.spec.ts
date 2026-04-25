import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock, distExists, readDist } from './_helpers';

test.describe('build-pipeline: llms.txt + humans.txt + security.txt', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('all three RFC/well-known files emit with non-empty content', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		// security.txt is gated on profile.security_email being set.
		const securityEmail = 'security@example.test';
		const profileRes = await adminApi.put(u(`/api/sites/${site.id}/profile`), {
			data: {
				business_name: 'E2E Co',
				contact_email: 'hello@example.test',
				security_email: securityEmail,
				country: 'SE'
			}
		});
		expect(profileRes.ok(), await profileRes.text()).toBe(true);

		await createPublishedPageWithBlock(adminApi, site.id, {
			slug: '/',
			title: 'Well-Known Home'
		});

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		// llms.txt
		expect(distExists(site.id, 'llms.txt')).toBe(true);
		const llms = readDist(site.id, 'llms.txt');
		expect(llms.length).toBeGreaterThan(0);

		// humans.txt
		expect(distExists(site.id, 'humans.txt')).toBe(true);
		const humans = readDist(site.id, 'humans.txt');
		expect(humans.length).toBeGreaterThan(0);

		// .well-known/security.txt with Contact: from profile
		expect(distExists(site.id, '.well-known/security.txt')).toBe(true);
		const sec = readDist(site.id, '.well-known/security.txt');
		expect(sec).toMatch(/Contact:\s+mailto:/);
		expect(sec).toContain(securityEmail);
	});
});
