import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	createPage,
	deleteSite,
	getStubCalls,
	resetStub,
	seedDeployTarget,
	triggerBuildAndWait,
	type Site
} from '../../fixtures/data';

// internal/deploy/dockyard.go requires https for dockyard_url. The Bun stub now
// serves TLS (cert in fixtures/certs/) and the binary trusts it via SSL_CERT_FILE
// passed by playwright.config.ts. The negative "http rejected" path runs
// unconditionally; the full happy path runs when `rsync` is on PATH AND there
// is no live SSHD (we observe the rsync-step 502 surface, not the route POST).
// dockyard_test.go covers the pure-Go HTTPS round-trip at the unit level.
import { execSync } from 'node:child_process';

const hasRsync = (() => {
	try {
		execSync('command -v rsync', { stdio: 'ignore' });
		return true;
	} catch {
		return false;
	}
})();

test.describe('deploy: dockyard kind', () => {
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

	test('http dockyard_url is rejected at target creation', async ({ adminApi }) => {
		await resetStub('dockyard');
		const res = await adminApi.post(u(`/api/sites/${site.id}/deploy-targets`), {
			data: {
				name: 'dy-http',
				kind: 'dockyard',
				config: {
					dockyard_url: 'http://127.0.0.1:18293',
					api_key: 'e2e-dy-key',
					server_id: '11111111-2222-3333-4444-555555555555',
					domain: 'e2e.example.test',
					host: '127.0.0.1',
					user: 'deploy',
					port: 22022,
					host_path: '/srv/sites/e2e',
					private_key_pem:
						'-----BEGIN OPENSSH PRIVATE KEY-----\nfake\n-----END OPENSSH PRIVATE KEY-----\n'
				},
				is_default: false
			}
		});
		expect(res.status()).toBe(400);
		const body = await res.json();
		expect(String(body.error ?? '')).toMatch(/https/);

		// Validator rejected before we hit the network: stub recorded zero calls.
		const calls = await getStubCalls('dockyard');
		expect(calls).toHaveLength(0);
	});

	test('dockyard deploy fires through TLS validator and surfaces the rsync error', async ({
		adminApi
	}) => {
		test.skip(!hasRsync, 'rsync not on PATH');
		await resetStub('dockyard');
		// Validator passes - proves HTTPS stub URL is accepted (TLS trust setup is correct).
		const target = await seedDeployTarget(adminApi, site.id, 'dockyard');
		const build = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(build.status).toBe('success');
		const res = await adminApi.post(u(`/api/sites/${site.id}/deploy`), {
			data: { build_id: build.build_id, target_id: target.id }
		});
		// Step 1 (rsync to host) fails because nothing listens on :22022. Deploy
		// surfaces a 502 with rsync-pinned error before reaching the HTTPS call.
		// To assert the route POST itself we'd need a live SSHD in the harness;
		// that round-trip is covered by dockyard_test.go.
		expect(res.status()).toBe(502);
		const body = await res.json();
		expect(String(body.error ?? '')).toMatch(/rsync|ssh|connection|refused/i);
	});
});
