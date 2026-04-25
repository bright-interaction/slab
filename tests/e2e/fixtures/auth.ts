import type { Page } from '@playwright/test';

export const ADMIN_EMAIL = 'admin@atomicsite.dev';
export const ADMIN_PASSWORD = 'changeme123';

export async function loginAsAdmin(page: Page): Promise<void> {
	await page.goto('/login');
	await page.getByLabel(/email/i).fill(ADMIN_EMAIL);
	await page.getByLabel(/password/i).fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: /sign in|log in|login/i }).click();
	await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 10_000 });
}
