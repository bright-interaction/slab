import { test, expect } from '../../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../../fixtures/data';

test.describe('Settings: Security', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('toggle HSTS, save', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/settings/security`);
		await expect(loggedInPage.getByRole('heading', { name: 'Security', exact: true })).toBeVisible();

		await loggedInPage.getByRole('switch', { name: 'Enable HSTS', exact: true }).click();

		const save = loggedInPage.getByRole('button', { name: 'Save', exact: true });
		await expect(save).toBeEnabled({ timeout: 5_000 });
		await save.click();
		await expect(loggedInPage.getByText(/up to date/i)).toBeVisible({ timeout: 8_000 });
	});
});
