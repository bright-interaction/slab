import { test, expect } from '../../fixtures/auth';

test.describe('Theme toggle', () => {
	test('clicking toggles data-theme; preference persists across reload', async ({
		loggedInPage
	}) => {
		await loggedInPage.goto('/');
		const html = loggedInPage.locator('html');
		const before = (await html.getAttribute('data-theme')) ?? 'light';

		const toggle = loggedInPage.getByRole('button', { name: /switch to (dark|light) mode/i });
		await toggle.click();

		// Wait for attribute change.
		await expect
			.poll(async () => await html.getAttribute('data-theme'), { timeout: 5_000 })
			.not.toBe(before);
		const after = await html.getAttribute('data-theme');
		expect(after).not.toBe(before);

		// Reload - preference persists.
		await loggedInPage.reload();
		await expect.poll(async () => await html.getAttribute('data-theme')).toBe(after);
	});
});
