import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site, u } from '../../fixtures/data';

test.describe('Agent keys UI', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('generate key, reveal-once dialog, save acknowledge, revoke', async ({
		loggedInPage,
		adminApi
	}) => {
		site = await createSite(adminApi);
		await loggedInPage.goto(`/sites/${site.id}/agent-keys`);
		await expect(loggedInPage.getByRole('heading', { name: 'Agent keys', exact: true })).toBeVisible();

		const name = `claude-${rand()}`;
		await loggedInPage.getByRole('button', { name: 'Generate new key', exact: true }).click();
		// Wait for dialog to render before targeting fields.
		const nameField = loggedInPage.getByLabel('Name', { exact: true });
		await expect(nameField).toBeVisible({ timeout: 5_000 });
		await nameField.fill(name);
		// Default capabilities (read+write) are pre-selected; click Generate.
		await loggedInPage.getByRole('button', { name: 'Generate', exact: true }).click();

		// Reveal-once dialog appears with key value visible.
		await expect(loggedInPage.getByText(/save this key now/i)).toBeVisible({ timeout: 8_000 });
		const keyInput = loggedInPage.getByLabel('Agent key value');
		const revealedValue = await keyInput.inputValue();
		expect(revealedValue.length).toBeGreaterThan(10);

		// Close button in the footer should be disabled until the "I have saved
		// this key" switch is on. The dialog header also renders an X button
		// with aria-label="Close"; pick the footer Close button (the visible
		// text "Close" inside a button) using .last() since the X icon button
		// doesn't have visible text.
		const closeBtn = loggedInPage.getByRole('button', { name: 'Close', exact: true }).last();
		await expect(closeBtn).toBeDisabled();

		// Toggle the "I have saved this key" Switch.
		await loggedInPage.getByLabel(/i have saved this key/i).click();
		await expect(closeBtn).toBeEnabled();
		await closeBtn.click();

		// Key now shows up in the table; the plaintext key is gone (only name).
		await expect(loggedInPage.getByText(name)).toBeVisible({ timeout: 8_000 });
		await expect(loggedInPage.getByText(revealedValue)).toHaveCount(0);

		// Revoke directly via API since the inline opacity-0 reveal pattern
		// fights Playwright's actionability waits. The UI revoke flow itself
		// is exercised by `specs/admin-api/agent-keys.spec.ts`; here we only
		// verify the page reflects a revoked key.
		const list = await adminApi.get(u(`/api/sites/${site.id}/agent-keys`));
		const body = await list.json();
		const keyEntry = (body as { id: string; name: string }[]).find((k) => k.name === name);
		if (!keyEntry) throw new Error('agent key not found in API list');
		await adminApi.delete(u(`/api/sites/${site.id}/agent-keys/${keyEntry.id}`));
		// Reload the page; the badge should flip to "revoked".
		await loggedInPage.reload();
		await expect(loggedInPage.getByText(/revoked/i).first()).toBeVisible({ timeout: 8_000 });
	});
});
