import { test, expect, u } from '../../fixtures/auth';
import { createSite, createPage, deleteSite, pollBuild, triggerBuild } from '../../fixtures/data';

test.describe('zzz-last: concurrent builds', () => {
	let siteID: string;

	test.beforeAll(async ({ adminApi }) => {
		const site = await createSite(adminApi, { siteType: 'b2b', structureType: 'one-pager' });
		siteID = site.id;
		await createPage(adminApi, siteID);
	});

	test.afterAll(async ({ adminApi }) => {
		if (siteID) await deleteSite(adminApi, siteID);
	});

	test('two parallel builds on the same site both reach a terminal state', async ({ adminApi }) => {
		// Kick off two builds back-to-back. Whether they queue, serialize, or
		// run in parallel is implementation-defined; what matters is that both
		// finish in success or failed (no orphans, no hangs).
		const [a, b] = await Promise.all([triggerBuild(adminApi, siteID), triggerBuild(adminApi, siteID)]);
		expect(a.build_id).toBeTruthy();
		expect(b.build_id).toBeTruthy();

		const [finalA, finalB] = await Promise.all([
			pollBuild(adminApi, siteID, a.build_id, 180_000),
			pollBuild(adminApi, siteID, b.build_id, 180_000)
		]);

		for (const final of [finalA, finalB]) {
			expect(['success', 'failed', 'timeout']).toContain(final.status);
		}
	});
});
