import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('Site overview', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('renders 6 stat cards and New page navigates to pages', async ({
		loggedInPage,
		adminApi
	}) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}`);

		// Six stat card labels - scope to the stat cards section so we don't
		// collide with the SectionNav links (which share "Pages" / "Media").
		const statSection = loggedInPage.locator('section').first();
		await expect(statSection.getByText('Pages', { exact: true })).toBeVisible();
		await expect(statSection.getByText('Components', { exact: true })).toBeVisible();
		await expect(statSection.getByText('Media', { exact: true })).toBeVisible();
		await expect(statSection.getByText('Agent keys', { exact: true })).toBeVisible();
		await expect(statSection.getByText('Last build', { exact: true })).toBeVisible();
		await expect(statSection.getByText('Latest eval', { exact: true })).toBeVisible();

		await loggedInPage.getByRole('button', { name: 'New page', exact: true }).click();
		await expect(loggedInPage).toHaveURL(new RegExp(`/sites/${site.id}/pages`));
	});
});
