import { test, expect } from '../../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../../fixtures/data';

test.describe('Settings: Analytics', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('enable GA4 + fill ID, save', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/settings/analytics`);
		await expect(loggedInPage.getByRole('heading', { name: 'Analytics', exact: true })).toBeVisible();

		await loggedInPage.getByRole('switch', { name: 'Enable GA4' }).click();

		// Once enabled, the Measurement ID input becomes visible.
		await loggedInPage.getByLabel('Measurement ID', { exact: true }).fill('G-E2E12345');

		const save = loggedInPage.getByRole('button', { name: 'Save', exact: true });
		await expect(save).toBeEnabled({ timeout: 5_000 });
		await save.click();
		await expect(loggedInPage.getByText(/up to date/i)).toBeVisible({ timeout: 8_000 });
	});
});
