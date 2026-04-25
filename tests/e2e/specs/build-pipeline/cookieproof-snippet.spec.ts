import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock, readDist, upsertSetting } from './_helpers';

test.describe('build-pipeline: cookieproof snippet injection', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('snippet is present when analytics.cookieproof_enabled=true and absent otherwise', async ({
		adminApi
	}) => {
		test.setTimeout(360_000);

		// Site A: cookieproof enabled
		const siteOn = await createSite(adminApi);
		cleanup.push(siteOn.id);
		await upsertSetting(adminApi, siteOn.id, 'analytics', 'cookieproof_enabled', 'true');
		await createPublishedPageWithBlock(adminApi, siteOn.id, {
			slug: '/',
			title: 'Consent On'
		});
		const onResult = await triggerBuildAndWait(adminApi, siteOn.id, 180_000);
		expect(onResult.status, `build_log:\n${onResult.build_log}`).toBe('success');
		const htmlOn = readDist(siteOn.id, 'index.html');
		expect(htmlOn).toContain('consent.example.com/loader.js');

		// Site B: cookieproof disabled (default)
		const siteOff = await createSite(adminApi);
		cleanup.push(siteOff.id);
		await createPublishedPageWithBlock(adminApi, siteOff.id, {
			slug: '/',
			title: 'Consent Off'
		});
		const offResult = await triggerBuildAndWait(adminApi, siteOff.id, 180_000);
		expect(offResult.status, `build_log:\n${offResult.build_log}`).toBe('success');
		const htmlOff = readDist(siteOff.id, 'index.html');
		expect(htmlOff).not.toContain('consent.example.com/loader.js');
	});
});
