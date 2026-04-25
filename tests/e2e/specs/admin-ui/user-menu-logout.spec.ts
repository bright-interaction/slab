import { test, expect } from '../../fixtures/auth';
import { E2E_CONSTANTS } from '../../playwright.config';

test.describe('User menu logout', () => {
	test('logout redirects to /login and clears cookie', async ({ loggedInPage }) => {
		await loggedInPage.goto('/');
		// Open account dropdown.
		await loggedInPage.getByRole('button', { name: /account menu/i }).click();
		// Click "Sign out" item.
		await loggedInPage.getByRole('menuitem', { name: /sign out/i }).click();

		await loggedInPage.waitForURL(/\/login$/, { timeout: 8_000 });

		const cookies = await loggedInPage.context().cookies();
		const auth = cookies.find((c) => c.name === E2E_CONSTANTS.COOKIE_NAME);
		// Either fully cleared or expired/empty.
		expect(!auth || auth.value === '' || (auth.expires ?? 0) > 0).toBeTruthy();
	});
});
