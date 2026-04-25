import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('Components UI', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('create, open detail, delete', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/components`);
		await expect(loggedInPage.getByRole('heading', { name: 'Components', exact: true })).toBeVisible();

		const name = `Hero${rand()}`;
		await loggedInPage.getByRole('button', { name: 'New component', exact: true }).click();
		await loggedInPage.getByLabel('Name', { exact: true }).fill(name);
		// Backend requires both `name` and `template`; the dialog leaves
		// template empty by default. Fill it so the POST succeeds.
		await loggedInPage.getByLabel('Template', { exact: true }).fill('<section>e2e</section>');
		await loggedInPage.getByRole('button', { name: 'Create', exact: true }).click();

		// Navigates to the detail page.
		await expect(loggedInPage).toHaveURL(
			new RegExp(`/sites/${site.id}/components/[a-f0-9]{24}`),
			{ timeout: 10_000 }
		);

		// Delete from detail page. The page has a single "Delete" button which
		// opens an inline ConfirmDialog whose footer also has a "Delete" button
		// - that's the one that fires the actual delete.
		await loggedInPage.getByRole('button', { name: 'Delete', exact: true }).first().click();
		await loggedInPage.getByRole('button', { name: 'Delete', exact: true }).last().click();
		await expect(loggedInPage).toHaveURL(new RegExp(`/sites/${site.id}/components$`), {
			timeout: 10_000
		});
		// Component no longer in list.
		await expect(loggedInPage.getByText(name)).toHaveCount(0);
	});
});
