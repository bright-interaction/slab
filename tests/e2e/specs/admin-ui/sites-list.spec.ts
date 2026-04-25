import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('Sites list', () => {
	const created: Site[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const s of created) {
			await deleteSite(adminApi, s.id);
		}
	});

	test('shows both sites and clicking one opens overview', async ({ loggedInPage, adminApi }) => {
		const a = await createSite(adminApi);
		const b = await createSite(adminApi);
		created.push(a, b);

		await loggedInPage.goto('/sites');
		await expect(loggedInPage.getByRole('heading', { name: 'Sites', exact: true })).toBeVisible();

		const cardA = loggedInPage.getByRole('link').filter({ hasText: a.name }).first();
		const cardB = loggedInPage.getByRole('link').filter({ hasText: b.name }).first();
		await expect(cardA).toBeVisible();
		await expect(cardB).toBeVisible();

		await cardA.click();
		await expect(loggedInPage).toHaveURL(new RegExp(`/sites/${a.id}(?!/)`));
	});
});
