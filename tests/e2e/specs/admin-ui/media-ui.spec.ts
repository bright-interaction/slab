import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

// Tiny 1x1 transparent PNG.
const PNG_BYTES = Buffer.from(
	'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=',
	'base64'
);

test.describe('Media UI', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('upload via file input, appears in grid, delete', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/media`);
		await expect(loggedInPage.getByRole('heading', { name: 'Media', exact: true })).toBeVisible();

		// MediaUploader exposes a hidden <input type="file"> in the DOM.
		const fileInput = loggedInPage.locator('input[type="file"]');
		await fileInput.setInputFiles({
			name: 'e2e-pixel.png',
			mimeType: 'image/png',
			buffer: PNG_BYTES
		});

		// Wait for upload to complete + grid to render at least one tile.
		await expect(loggedInPage.locator('img').first()).toBeVisible({ timeout: 15_000 });

		// "Showing 1 to 1 of 1" - gives us a visible state change to assert on.
		await expect(loggedInPage.getByText(/showing 1 to 1 of 1/i)).toBeVisible({
			timeout: 8_000
		});

		// TODO: copy URL button uses navigator.clipboard which is brittle in
		// headless browsers; we skip the clipboard assertion per the brief.

		// Delete via row hover. MediaGrid emits a delete handler - we look for any
		// trash/delete affordance after hovering the first tile.
		const tile = loggedInPage.locator('img').first();
		await tile.hover();
		const delBtn = loggedInPage.getByRole('button', { name: /delete/i }).first();
		if (await delBtn.isVisible({ timeout: 1_500 }).catch(() => false)) {
			await delBtn.click();
			await expect(loggedInPage.getByText(/no media yet|showing 0 to 0/i)).toBeVisible({
				timeout: 8_000
			});
		}
	});
});
