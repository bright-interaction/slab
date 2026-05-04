import { test, expect } from '../../fixtures/auth';
import { createBlock, createPage, createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('Pages editor – live preview modal', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('Preview button opens a modal; closing it stops fetches', async ({ loggedInPage, adminApi }) => {
		site = await createSite(adminApi);
		const page = await createPage(adminApi, site.id);
		await createBlock(adminApi, site.id, page.id, {
			block_type: 'text',
			data: { content: 'Preview-modal smoke' }
		});

		await loggedInPage.goto(`/sites/${site.id}/pages/${page.id}`);
		await expect(loggedInPage.getByText(/1 block\(s\)/i)).toBeVisible({ timeout: 10_000 });

		// Right-column inline preview iframe must NOT be present in the
		// expanded block (Phase 28: it is now behind a modal).
		const blockTypeChip = loggedInPage.getByText(/^TEXT$/i).first();
		await blockTypeChip.click();
		await expect(loggedInPage.locator('iframe[title="Block preview"]')).toHaveCount(0);

		// Click the Eye/Preview button on the block row, the dialog mounts.
		await loggedInPage.getByRole('button', { name: /open live preview/i }).first().click();
		await expect(loggedInPage.getByRole('dialog', { name: /live preview/i })).toBeVisible();
		await expect(loggedInPage.locator('iframe[title="Block preview"]')).toHaveCount(1);

		// Close the dialog, the iframe goes away.
		await loggedInPage.keyboard.press('Escape');
		await expect(loggedInPage.locator('iframe[title="Block preview"]')).toHaveCount(0);
	});

	test('Preview button is disabled for a block with no id (newly added, not saved)', async ({
		loggedInPage,
		adminApi
	}) => {
		const fresh = await createSite(adminApi);
		const freshPage = await createPage(adminApi, fresh.id);
		try {
			await loggedInPage.goto(`/sites/${fresh.id}/pages/${freshPage.id}`);
			await loggedInPage.getByRole('button', { name: 'Add block', exact: true }).first().click();
			await loggedInPage.getByRole('button', { name: 'Text', exact: true }).click();

			// New block in DOM, but block.id is empty until bulk save.
			const previewBtn = loggedInPage.getByRole('button', { name: /save the page once before previewing/i }).first();
			await expect(previewBtn).toBeDisabled();
		} finally {
			await deleteSite(adminApi, fresh.id);
		}
	});
});
