import { test, expect } from '../../../fixtures/auth';
import { createSite, deleteSite, rand, type Site, u } from '../../../fixtures/data';

test.describe('Settings: General', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('edit site name, save, verify via API', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/settings/general`);
		await expect(loggedInPage.getByRole('heading', { name: 'General', exact: true })).toBeVisible();

		const newName = `Renamed ${rand()}`;
		const nameInput = loggedInPage.getByLabel('Site name', { exact: true });
		await nameInput.fill(newName);

		await loggedInPage.getByRole('button', { name: 'Save', exact: true }).click();
		await expect(loggedInPage.getByText(/up to date/i)).toBeVisible({ timeout: 8_000 });

		const res = await adminApi.get(u(`/api/sites/${site.id}`));
		const body = await res.json();
		const fetched = body.site ?? body;
		expect(fetched.name).toBe(newName);
	});
});
