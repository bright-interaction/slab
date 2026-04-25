import { execSync } from 'node:child_process';
import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	createPage,
	deleteSite,
	seedDeployTarget,
	triggerBuildAndWait,
	type Site
} from '../../fixtures/data';

// Rsync deploy targets a configured SSH host/port. The e2e harness does not
// spin up sshd, so we drive the *failure* path: target validation passes,
// deploy fires, rsync attempts to connect to 127.0.0.1:22022 (where nothing
// listens), and the server surfaces a clean 502 with a descriptive message.
// rsync_test.go covers the happy path at the unit level.
//
// If `rsync` is not on PATH the deploy would error before we observe the
// SSH-unreachable path, so the spec is skipped in that case only.

const hasRsync = (() => {
	try {
		execSync('command -v rsync', { stdio: 'ignore' });
		return true;
	} catch {
		return false;
	}
})();

test.describe('deploy: rsync kind', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi, { siteType: 'b2b', structureType: 'one-pager' });
		cleanup.push(site.id);
		await createPage(adminApi, site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('rsync deploy without SSHD surfaces a clean failure', async ({ adminApi }) => {
		test.skip(!hasRsync, 'rsync not on PATH');
		const target = await seedDeployTarget(adminApi, site.id, 'rsync');
		const build = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(build.status).toBe('success');
		const res = await adminApi.post(u(`/api/sites/${site.id}/deploy`), {
			data: { build_id: build.build_id, target_id: target.id }
		});
		expect(res.status()).toBe(502);
		const body = await res.json();
		expect(String(body.error ?? '')).toMatch(/rsync|ssh|connection|refused/i);
	});
});
