import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock, readWorkspaceFile, upsertSetting } from './_helpers';

test.describe('build-pipeline: nginx.conf', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('emits log_format JSON directive when analytics enabled and at least one location block', async ({
		adminApi
	}) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		// Force the JSON log_format branch
		await upsertSetting(
			adminApi,
			site.id,
			'analytics',
			'atomicsite_tracking_enabled',
			'true'
		);

		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'Nginx Home' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		// nginx.conf is at workspace root, not under dist/.
		const conf = readWorkspaceFile(site.id, 'nginx.conf');
		expect(conf).toContain('log_format atomicsite_json');
		// At least one location block
		expect(conf).toMatch(/location\s+[^{]*\{/);
		// Security headers piped through nginx as well
		expect(conf).toContain('add_header Content-Security-Policy');
	});
});
