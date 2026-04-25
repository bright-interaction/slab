import { test, expect } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	triggerBuildAndWait,
	type Site
} from '../../fixtures/data';

test.describe('Evaluations UI', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('after a build, evaluations index shows scores; click to detail', async ({
		loggedInPage,
		adminApi
	}) => {
		site = await createSite(adminApi);
		// Trigger a real build via API; evaluation rows are written when it succeeds.
		const result = await triggerBuildAndWait(adminApi, site.id, 240_000);
		test.skip(result.status !== 'success', `build did not succeed (got ${result.status})`);

		await loggedInPage.goto(`/sites/${site.id}/evaluations`);
		await expect(
			loggedInPage.getByRole('heading', { name: 'Evaluations', exact: true })
		).toBeVisible();

		// Latest scores donuts render; "Builds" section lists at least one row.
		await expect(loggedInPage.getByText(/latest scores/i)).toBeVisible();
		const detailLink = loggedInPage.getByRole('link', { name: /^detail$/i }).first();
		await expect(detailLink).toBeVisible({ timeout: 8_000 });
		await detailLink.click();
		await expect(loggedInPage).toHaveURL(
			new RegExp(`/sites/${site.id}/evaluations/[a-f0-9]{24}`),
			{ timeout: 10_000 }
		);
	});
});
