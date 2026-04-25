import { test, expect } from '../../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../../fixtures/data';

test.describe('Settings: SEO', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('edit meta title template, save', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/settings/seo`);
		await expect(loggedInPage.getByRole('heading', { name: 'SEO', exact: true })).toBeVisible();

		const tpl = `{page_title} | E2E ${rand()}`;
		await loggedInPage.getByLabel('Meta title template', { exact: true }).fill(tpl);

		const save = loggedInPage.getByRole('button', { name: 'Save', exact: true });
		await expect(save).toBeEnabled({ timeout: 5_000 });
		await save.click();
		await expect(loggedInPage.getByText(/up to date/i)).toBeVisible({ timeout: 8_000 });
	});
});
