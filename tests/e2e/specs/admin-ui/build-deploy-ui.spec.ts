import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('Build + deploy UI', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('trigger build via UI fires the build API and renders progress', async ({
		loggedInPage,
		adminApi
	}) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/build`);
		await expect(loggedInPage.getByRole('heading', { name: 'Build', exact: true })).toBeVisible();

		// Click the trigger button and capture the build_id from the POST response.
		const triggerResp = loggedInPage.waitForResponse(
			(r) => r.url().includes(`/api/sites/${site.id}/build`) && r.request().method() === 'POST'
		);
		await loggedInPage.getByRole('button', { name: 'Trigger build', exact: true }).click();
		const resp = await triggerResp;
		expect(resp.ok()).toBeTruthy();
		const body = (await resp.json()) as { build_id: string };
		expect(body.build_id).toBeTruthy();

		// UI shows the live polling state: elapsed counter and either the building badge
		// or (if the build finished quickly) the success/failed/timeout badge.
		await expect(loggedInPage.getByText(/elapsed/i)).toBeVisible({ timeout: 15_000 });

		// Build completion itself (and the UI showing the final badge) is exercised by
		// specs/build-pipeline/full-build.spec.ts. This UI spec only owns the trigger wiring.
	});
});
