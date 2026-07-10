import { test, expect } from '@playwright/test';
import { ADMIN_EMAIL, ADMIN_PASSWORD } from '../../fixtures/auth';

test.describe('Login flow', () => {
	test('shows error on bad creds', async ({ page }) => {
		await page.goto('/login');
		await page.getByLabel(/email/i).fill('admin@slab.dev');
		await page.getByLabel(/password/i).fill('not-the-real-password');
		await page.getByRole('button', { name: /sign in/i }).click();
		const alert = page.getByRole('alert');
		await expect(alert).toBeVisible({ timeout: 8_000 });
		await expect(page).toHaveURL(/\/login/);
	});

	test('valid creds redirect to dashboard', async ({ page }) => {
		await page.goto('/login');
		await page.getByLabel(/email/i).fill(ADMIN_EMAIL);
		await page.getByLabel(/password/i).fill(ADMIN_PASSWORD);
		await page.getByRole('button', { name: /sign in/i }).click();
		await page.waitForURL((url) => !url.pathname.startsWith('/login'), {
			timeout: 10_000
		});
		// Topbar always visible after login.
		await expect(page.getByRole('button', { name: /open command palette/i })).toBeVisible();
	});

	test('empty fields blocked by browser-required validation', async ({ page }) => {
		await page.goto('/login');
		const submit = page.getByRole('button', { name: /sign in/i });
		await submit.click();
		// Inputs are required; the form does not submit, URL stays /login.
		await expect(page).toHaveURL(/\/login/);
		const email = page.getByLabel(/email/i);
		const valid = await email.evaluate((el: HTMLInputElement) => el.validity.valid);
		expect(valid).toBe(false);
	});
});
