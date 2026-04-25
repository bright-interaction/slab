import { test, expect } from '../../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../../fixtures/data';

test.describe('Settings: Nginx preview', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	// The /settings/nginx route is read-only; it renders a preview from current
	// settings. There is no editable snippet on the page itself. We assert it
	// renders a non-empty preview and that the Refresh button works without
	// crashing - a reasonable smoke test for this route.
	test('preview renders, refresh button works', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/settings/nginx`);
		await expect(loggedInPage.getByRole('heading', { name: 'Nginx preview', exact: true })).toBeVisible();

		// Wait for preview to load.
		await loggedInPage.waitForLoadState('networkidle');
		const pre = loggedInPage.locator('pre').first();
		await expect(pre).toBeVisible({ timeout: 8_000 });
		const txt = (await pre.textContent()) ?? '';
		expect(txt.length).toBeGreaterThan(10);

		await loggedInPage.getByRole('button', { name: 'Refresh', exact: true }).click();
		await expect(pre).toBeVisible();
	});
});
