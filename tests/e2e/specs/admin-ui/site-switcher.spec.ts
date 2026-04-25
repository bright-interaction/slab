import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('Site switcher', () => {
	const created: Site[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const s of created) await deleteSite(adminApi, s.id);
	});

	test('switch from site A to site B via topbar dropdown', async ({
		loggedInPage,
		adminApi
	}) => {
		const a = await createSite(adminApi);
		const b = await createSite(adminApi);
		created.push(a, b);

		await loggedInPage.goto(`/sites/${a.id}`);
		await expect(loggedInPage).toHaveURL(new RegExp(`/sites/${a.id}(?!/)`));

		// Open the SiteSwitcher popover (trigger labelled by the active site name).
		await loggedInPage.getByRole('button', { name: a.name }).first().click();
		// The dropdown lists site B; click it.
		await loggedInPage.getByRole('button').filter({ hasText: b.name }).first().click();

		await expect(loggedInPage).toHaveURL(new RegExp(`/sites/${b.id}(?!/)`), {
			timeout: 8_000
		});
	});
});
