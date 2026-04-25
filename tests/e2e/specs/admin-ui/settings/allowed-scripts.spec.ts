import { test, expect } from '../../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../../fixtures/data';

test.describe('Settings: Allowed scripts', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('add domain, toggle active, delete', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/settings/allowed-scripts`);
		await expect(loggedInPage.getByRole('heading', { name: 'Allowed Scripts', exact: true })).toBeVisible();

		const domain = `e2e-${rand()}.example.com`;
		await loggedInPage.getByRole('button', { name: /add (first )?domain/i }).first().click();
		await loggedInPage.getByLabel('Domain', { exact: true }).fill(domain);
		await loggedInPage.getByLabel('Purpose', { exact: true }).fill('e2e test');
		await loggedInPage.getByRole('button', { name: 'Add', exact: true }).click();

		await expect(loggedInPage.getByText(domain, { exact: true })).toBeVisible({ timeout: 8_000 });

		// New scripts default to is_active=1 → switch aria-label is "Deactivate".
		// Click flips it to "Activate".
		await loggedInPage.getByRole('switch', { name: 'Deactivate' }).click();
		await expect(loggedInPage.getByRole('switch', { name: 'Activate' })).toBeVisible({
			timeout: 5_000
		});

		// Open the row Remove button (icon-only with aria-label="Remove"); confirm in ConfirmDialog.
		await loggedInPage.getByRole('button', { name: 'Remove', exact: true }).first().click();
		await loggedInPage.getByRole('button', { name: 'Remove', exact: true }).last().click();
		await expect(loggedInPage.getByText(domain, { exact: true })).toHaveCount(0, {
			timeout: 8_000
		});
	});
});
