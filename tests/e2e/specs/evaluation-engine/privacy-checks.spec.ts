import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import {
	createPublishedPageWithBlock,
	upsertSetting
} from '../build-pipeline/_helpers';
import { checkNames, findCategory, getEvaluations, parseChecks, VALID_GRADES } from './_helpers';

test.describe('evaluation-engine: privacy category', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('privacy row has consent + tracking + AI-bot checks', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);
		await upsertSetting(adminApi, site.id, 'analytics', 'cookieproof_enabled', 'true');
		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'Privacy Home' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		const rows = await getEvaluations(adminApi, site.id, result.build_id);
		const privacy = findCategory(rows, 'privacy');
		expect(typeof privacy.score).toBe('number');
		expect(VALID_GRADES.has(privacy.grade), `grade=${privacy.grade}`).toBe(true);

		const names = checkNames(privacy);
		// Names from internal/eval/privacy.go.
		for (const expected of [
			'Consent Banner',
			'Tracker Count',
			'Pre-Consent Tracking',
			'AI Training Bots Blocked'
		]) {
			expect(names, `missing ${expected} in ${names.join(',')}`).toContain(expected);
		}

		// Consent Banner check exists with a real boolean verdict (the eval's
		// substring matcher only recognises specific platform names, so the
		// pass/fail value is intentionally not asserted here).
		const consent = parseChecks(privacy).find((c) => c.name === 'Consent Banner');
		expect(consent, 'Consent Banner check should be present').toBeTruthy();
		expect(typeof consent?.passed).toBe('boolean');
	});
});
