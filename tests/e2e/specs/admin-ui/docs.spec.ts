import { test, expect } from '../../fixtures/auth';

const ROUTES: Array<{ path: string; heading: RegExp }> = [
	{ path: '/docs', heading: /^Documentation$/ },
	{ path: '/docs/architecture', heading: /^Architecture$/ },
	{ path: '/docs/features', heading: /^Features$/ },
	{ path: '/docs/agent-api', heading: /^Agent API$/ },
	{ path: '/docs/connect-your-agent', heading: /^Connect your agent$/ },
	{ path: '/docs/integrations', heading: /^Integrations$/ }
];

test.describe('Documentation pages', () => {
	for (const { path, heading } of ROUTES) {
		test(`renders ${path} without console errors`, async ({ loggedInPage }) => {
			const consoleErrors: string[] = [];
			loggedInPage.on('console', (msg) => {
				if (msg.type() === 'error') consoleErrors.push(msg.text());
			});
			loggedInPage.on('pageerror', (err) => consoleErrors.push(err.message));

			await loggedInPage.goto(path);
			await loggedInPage.waitForLoadState('networkidle');

			await expect(loggedInPage.getByRole('heading', { name: heading, level: 1 })).toBeVisible();

			expect(consoleErrors, `console errors on ${path}: ${consoleErrors.join('\n')}`).toHaveLength(
				0
			);
		});
	}

	test('hub links resolve to all five sub-pages', async ({ loggedInPage }) => {
		await loggedInPage.goto('/docs');
		await loggedInPage.waitForLoadState('networkidle');

		for (const sub of ['architecture', 'features', 'agent-api', 'connect-your-agent', 'integrations']) {
			const link = loggedInPage.locator(`a[href="/docs/${sub}"]`).first();
			await expect(link).toBeVisible();
		}
	});

	test('sub-pages have a back link to /docs', async ({ loggedInPage }) => {
		for (const sub of ['architecture', 'features', 'agent-api', 'connect-your-agent', 'integrations']) {
			await loggedInPage.goto(`/docs/${sub}`);
			await loggedInPage.waitForLoadState('networkidle');
			const back = loggedInPage.getByRole('link', { name: /documentation/i }).first();
			await expect(back).toBeVisible();
			await expect(back).toHaveAttribute('href', '/docs');
		}
	});

	test('Topbar Documentation button routes to /docs', async ({ loggedInPage }) => {
		await loggedInPage.goto('/');
		await loggedInPage.waitForLoadState('networkidle');
		await loggedInPage.getByRole('button', { name: /^Documentation$/ }).click();
		await loggedInPage.waitForURL('**/docs');
		await expect(
			loggedInPage.getByRole('heading', { name: 'Documentation', level: 1 })
		).toBeVisible();
	});
});
