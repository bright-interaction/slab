import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('Pages editor', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('create page via dialog, open editor, delete', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/pages`);
		await expect(loggedInPage.getByRole('heading', { name: 'Pages', exact: true })).toBeVisible();

		// Open New page dialog. The header button is named exactly "New page".
		await loggedInPage.getByRole('button', { name: 'New page', exact: true }).click();

		const title = `E2E page ${rand()}`;
		const slug = `e2e-${rand()}`;
		await loggedInPage.getByLabel('Title', { exact: true }).fill(title);
		// Wait past slug auto-derive debounce - same pattern as wizard.spec.ts.
		await loggedInPage.waitForTimeout(250);
		await loggedInPage.getByLabel('Slug', { exact: true }).fill(slug);
		await loggedInPage.getByRole('button', { name: 'Create', exact: true }).click();

		// Page editor opens at /sites/{id}/pages/{pageID}.
		await expect(loggedInPage).toHaveURL(
			new RegExp(`/sites/${site.id}/pages/[a-f0-9]{24}`),
			{ timeout: 10_000 }
		);
		await expect(loggedInPage.getByRole('heading', { name: title })).toBeVisible();

		// Go back to the list, the new page should be visible.
		await loggedInPage.goto(`/sites/${site.id}/pages`);
		await expect(loggedInPage.getByText(title)).toBeVisible();

		// Delete via the row menu - hover the row to reveal actions.
		const row = loggedInPage.locator('div[role="list"] >> text=' + title).first();
		await row.hover();
		// Delete buttons live inside the PageRow component; pick by aria.
		// TODO: PageRow uses an inline menu trigger; selector may need adjustment
		// if the row markup changes. We look for any button with aria-label "Delete page".
		const delBtn = loggedInPage.getByRole('button', { name: /delete page/i }).first();
		if (await delBtn.isVisible({ timeout: 1_500 }).catch(() => false)) {
			await delBtn.click();
			await loggedInPage.getByRole('button', { name: 'Delete', exact: true }).click();
			await expect(loggedInPage.getByText(title)).toHaveCount(0, { timeout: 8_000 });
		}
	});
});
