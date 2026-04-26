import { test, expect } from '../../fixtures/auth';

test.describe('Account settings page', () => {
	test('shows back-to-dashboard link, profile metadata, sessions and preferences cards', async ({
		loggedInPage
	}) => {
		await loggedInPage.goto('/account');
		await loggedInPage.waitForLoadState('networkidle');

		await expect(
			loggedInPage.getByRole('heading', { name: 'Account settings', exact: true })
		).toBeVisible();

		const backLink = loggedInPage.getByRole('link', { name: /dashboard/i }).first();
		await expect(backLink).toBeVisible();
		await expect(backLink).toHaveAttribute('href', '/');

		await expect(loggedInPage.getByText(/member since/i)).toBeVisible();
		await expect(loggedInPage.getByText(/last updated/i)).toBeVisible();
		await expect(loggedInPage.getByText(/account id/i)).toBeVisible();

		await expect(loggedInPage.getByRole('heading', { name: /^sessions$/i })).toBeVisible();
		await expect(
			loggedInPage.getByRole('button', { name: /sign out everywhere else/i })
		).toBeVisible();

		await expect(loggedInPage.getByRole('heading', { name: /^preferences$/i })).toBeVisible();
		await expect(loggedInPage.getByText(/^theme$/i)).toBeVisible();
	});

	test('back link from /members points to /account', async ({ loggedInPage }) => {
		await loggedInPage.goto('/members');
		await loggedInPage.waitForLoadState('networkidle');
		const backLink = loggedInPage.getByRole('link', { name: /account settings/i }).first();
		await expect(backLink).toBeVisible();
		await expect(backLink).toHaveAttribute('href', '/account');
	});
});
