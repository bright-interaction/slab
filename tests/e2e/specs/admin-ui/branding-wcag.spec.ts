import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, type Site, u } from '../../fixtures/data';

test.describe('Branding & WCAG matrix', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('change primary color, see WCAG matrix, save persists via API', async ({
		loggedInPage,
		adminApi
	}) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/branding`);
		await expect(loggedInPage.getByRole('heading', { name: 'Branding', exact: true })).toBeVisible();

		// Contrast matrix heading is always rendered.
		await expect(loggedInPage.getByText(/contrast matrix/i)).toBeVisible();

		// ColorSlot exposes each hex as an input with aria-label "<Label> hex value".
		await loggedInPage.getByLabel('Primary hex value').fill('#1d4ed8');

		// Save.
		const saveBtn = loggedInPage.getByRole('button', { name: 'Save', exact: true });
		await expect(saveBtn).toBeEnabled({ timeout: 5_000 });
		await saveBtn.click();
		await expect(loggedInPage.getByText(/up to date/i)).toBeVisible({ timeout: 8_000 });

		// Verify via API.
		const res = await adminApi.get(u(`/api/sites/${site.id}`));
		expect(res.ok()).toBeTruthy();
		const body = await res.json();
		const fetched = body.site ?? body;
		expect(fetched.primary_color?.toLowerCase()).toBe('#1d4ed8');
	});
});
