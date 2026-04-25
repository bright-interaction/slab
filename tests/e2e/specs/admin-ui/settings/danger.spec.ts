import { test, expect } from '../../../fixtures/auth';
import { createSite, type Site, u } from '../../../fixtures/data';

test.describe('Settings: Danger zone', () => {
	test('delete site requires slug confirmation; site is gone after', async ({
		loggedInPage,
		adminApi
	}) => {
		const site: Site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/settings/danger`);
		await expect(loggedInPage.getByRole('heading', { name: 'Danger zone', exact: true })).toBeVisible();

		await loggedInPage.getByRole('button', { name: 'Delete site', exact: true }).click();

		// Confirm button is disabled until the slug is typed exactly.
		const confirmBtn = loggedInPage.getByRole('button', { name: 'Delete forever', exact: true });
		await expect(confirmBtn).toBeDisabled();

		const slugInput = loggedInPage.getByLabel(`Type "${site.slug}" to confirm`);
		await slugInput.fill(site.slug);
		await expect(confirmBtn).toBeEnabled();
		await confirmBtn.click();

		// Redirects to /sites.
		await loggedInPage.waitForURL(/\/sites$/, { timeout: 10_000 });

		// Site GET returns 404.
		const res = await adminApi.get(u(`/api/sites/${site.id}`));
		expect(res.status()).toBe(404);
	});
});
