import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('Dashboard', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('shows stat cards and clicking site card navigates to site overview', async ({
		loggedInPage,
		adminApi
	}) => {
		site = await createSite(adminApi);

		await loggedInPage.goto('/');
		await loggedInPage.waitForLoadState('networkidle');

		// Stat cards on dashboard root.
		await expect(loggedInPage.getByText(/total sites/i)).toBeVisible();
		await expect(loggedInPage.getByText(/builds, last 7 days/i)).toBeVisible();
		await expect(loggedInPage.getByText(/latest grade/i)).toBeVisible();

		// Sites listing - click into the freshly created site.
		await loggedInPage.goto('/sites');
		await expect(loggedInPage.getByRole('heading', { name: 'Sites', exact: true })).toBeVisible();
		const card = loggedInPage.getByRole('link').filter({ hasText: site.name }).first();
		await card.waitFor({ state: 'visible', timeout: 10_000 });
		await card.click();
		await expect(loggedInPage).toHaveURL(new RegExp(`/sites/${site.id}(?!/)`));
	});
});
