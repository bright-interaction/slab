import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('Knowledgebase + Guardrails UI', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('knowledgebase: create entry', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/knowledgebase`);
		await expect(loggedInPage.getByRole('heading', { name: 'Knowledgebase', exact: true })).toBeVisible();

		const title = `Brand voice ${rand()}`;
		await loggedInPage.getByRole('button', { name: 'Add entry', exact: true }).first().click();
		await loggedInPage.getByLabel('Title', { exact: true }).fill(title);
		// Backend requires non-empty content.
		await loggedInPage.getByLabel('Content', { exact: true }).fill('Use a calm, declarative tone.');
		await loggedInPage.getByRole('button', { name: 'Create', exact: true }).click();

		// Entry should now show up under the active (default: brand) category.
		await expect(loggedInPage.getByText(title)).toBeVisible({ timeout: 8_000 });
	});

	test('guardrails: quick-add an Allow block types rule', async ({ loggedInPage, adminApi }) => {
		// Reuse the site if it exists, otherwise create a new one.
		if (!site) site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/guardrails`);
		await expect(loggedInPage.getByRole('heading', { name: 'Guardrails', exact: true })).toBeVisible();

		await loggedInPage.getByRole('button', { name: /allow block types/i }).click();
		// The new rule appears under the "Allow block type" group.
		await expect(loggedInPage.getByText(/allow block type/i).first()).toBeVisible({
			timeout: 8_000
		});
	});
});
