import { test, expect } from '@playwright/test';
import { loginAsAdmin } from '../../fixtures/auth';

test.describe('Onboarding wizard', () => {
	test('walk all 6 steps and create a site', async ({ page }) => {
		await loginAsAdmin(page);

		// Step 1: Type
		await page.goto('/sites/new/wizard/type');
		await expect(page.getByRole('heading', { name: /what kind of site/i })).toBeVisible();
		await page.getByRole('button', { name: /^B2B/ }).click();
		await expect(page).toHaveURL(/\/sites\/new\/wizard\/kit/);

		// Step 2: Kit (skip via the card-style button with this text)
		await expect(page.getByRole('heading', { name: /pick a starter kit/i })).toBeVisible();
		// Wait for the async kits fetch to land + card render
		await page.waitForLoadState('networkidle');
		const skipCard = page.locator('button').filter({ hasText: 'Skip kit, start blank' }).first();
		await skipCard.waitFor({ state: 'visible', timeout: 10_000 });
		await skipCard.click();
		await expect(page).toHaveURL(/\/sites\/new\/wizard\/info/);

		// Step 3: Info. Each oninput commits to the wizard store.
		await page.getByLabel('Site name', { exact: true }).fill('Lab Test Site');
		// Wait past the 200ms slug auto-derive debounce, then mark slug as
		// user-touched by typing the desired slug.
		await page.waitForTimeout(250);
		await page.getByLabel('Slug', { exact: true }).fill('lab-e2e');
		await page.getByLabel('Business name', { exact: true }).fill('Lab Test AB');
		await page.getByLabel('Contact email', { exact: true }).fill('hej@labtest.example');
		// Wait for validation to settle (validation is reactive on store changes).
		await expect(page.getByRole('button', { name: 'Next', exact: true })).toBeEnabled();
		await page.getByRole('button', { name: 'Next', exact: true }).click();
		await expect(page).toHaveURL(/\/sites\/new\/wizard\/structure/);

		// Step 4: Structure. Buttons with aria-pressed (not native radios).
		await page.getByRole('button', { name: /one-pager/i }).click();
		await expect(page.getByRole('button', { name: 'Next', exact: true })).toBeEnabled();
		await page.getByRole('button', { name: 'Next', exact: true }).click();
		await expect(page).toHaveURL(/\/sites\/new\/wizard\/branding/);

		// Step 5: Branding (defaults are valid)
		await page.getByRole('button', { name: 'Next', exact: true }).click();
		await expect(page).toHaveURL(/\/sites\/new\/wizard\/confirm/);

		// Step 6: Confirm
		await page.getByRole('button', { name: /create site/i }).click();
		await expect(page).toHaveURL(/\/sites\/[a-f0-9]{24}/, { timeout: 15_000 });
	});

	test('Step 1 click selects type and auto-advances to kit', async ({ page }) => {
		await loginAsAdmin(page);
		await page.goto('/sites/new/wizard/type');
		await page.getByRole('button', { name: /^B2B/ }).click();
		await expect(page).toHaveURL(/\/sites\/new\/wizard\/kit/, { timeout: 5_000 });
	});

	test('E-commerce card advances to kit step', async ({ page }) => {
		await loginAsAdmin(page);
		await page.goto('/sites/new/wizard/type');
		const ecom = page.getByRole('button', { name: /e-commerce/i });
		await expect(ecom).toBeEnabled();
		await ecom.click();
		await expect(page).toHaveURL(/\/sites\/new\/wizard\/kit/, { timeout: 5_000 });
	});

	test('Nordic and accented characters transliterate to ASCII in slug + silo prefix', async ({ page }) => {
		await loginAsAdmin(page);
		await page.goto('/sites/new/wizard/info');

		// Title with ÅÄÖ ÜßÉ should auto-derive a clean ASCII slug.
		await page.getByLabel('Site name', { exact: true }).fill('Tjänster Åre & Östergötland Café');
		// Wait past the slug auto-derive debounce.
		await page.waitForTimeout(300);
		const slugInput = page.getByLabel('Slug', { exact: true });
		const slug = await slugInput.inputValue();
		expect(slug).toBe('tjanster-are-ostergotland-cafe');

		// Typing accented chars directly into slug also normalizes.
		await slugInput.fill('tjänster');
		expect(await slugInput.inputValue()).toBe('tjanster');

		// And the silo slug_prefix in the structure step normalizes too.
		await page.getByLabel('Business name', { exact: true }).fill('Acme AB');
		await page.getByLabel('Contact email', { exact: true }).fill('a@b.example');
		await page.getByRole('button', { name: 'Next', exact: true }).click();
		await expect(page).toHaveURL(/\/sites\/new\/wizard\/structure/);
		await page.getByRole('button', { name: /soft silo/i }).click();
		// Type accented characters into the first silo's Name; slug_prefix
		// auto-derives without stripping ÅÄÖ.
		await page.getByLabel('Name', { exact: true }).first().fill('Tjänster');
		const slugPrefixInputs = page.getByLabel('Slug prefix', { exact: true });
		await expect(slugPrefixInputs.first()).toHaveValue('tjanster');
	});

	test('no Svelte effect_update_depth_exceeded errors during the wizard flow', async ({ page }) => {
		const errors: string[] = [];
		page.on('pageerror', (e) => errors.push(e.message));
		await loginAsAdmin(page);
		await page.goto('/sites/new/wizard/type');
		await page.getByRole('button', { name: /^B2B/ }).click();
		await page.waitForURL(/\/sites\/new\/wizard\/kit/);
		const skipCard = page.locator('button').filter({ hasText: 'Skip kit, start blank' }).first();
		await skipCard.waitFor({ state: 'visible', timeout: 10_000 });
		await skipCard.click();
		await page.waitForURL(/\/sites\/new\/wizard\/info/);
		expect(errors.filter((e) => e.includes('effect_update_depth'))).toEqual([]);
	});
});
