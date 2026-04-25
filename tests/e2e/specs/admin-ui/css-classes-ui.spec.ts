import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('CSS classes UI', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('create class, see in list, delete', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/css-classes`);
		await expect(loggedInPage.getByRole('heading', { name: 'CSS classes', exact: true })).toBeVisible();

		const name = `e2e-${rand()}`;
		await loggedInPage.getByRole('button', { name: 'Add class', exact: true }).first().click();
		await loggedInPage.getByLabel('Name', { exact: true }).fill(name);
		// Backend requires non-empty CSS body.
		await loggedInPage.getByLabel('CSS', { exact: true }).fill('color: red;');
		await loggedInPage.getByRole('button', { name: 'Create', exact: true }).click();

		// New class appears in the list (rendered with leading dot).
		await expect(loggedInPage.getByText(`.${name}`, { exact: true })).toBeVisible({
			timeout: 8_000
		});

		// Delete via aria-label "Delete class".
		await loggedInPage
			.getByText(`.${name}`, { exact: true })
			.locator('xpath=ancestor::div[contains(@class,"group")]')
			.first()
			.hover();
		await loggedInPage.getByRole('button', { name: 'Delete class' }).first().click();
		await loggedInPage.getByRole('button', { name: 'Delete', exact: true }).click();
		await expect(loggedInPage.getByText(`.${name}`, { exact: true })).toHaveCount(0, {
			timeout: 8_000
		});
	});
});
