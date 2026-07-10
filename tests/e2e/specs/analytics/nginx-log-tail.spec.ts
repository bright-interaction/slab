import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';
import { E2E_CONSTANTS } from '../../playwright.config';

// internal/analytics/manager.go spawns a per-site Parser goroutine that tails a
// JSON-formatted nginx access log and writes visit_events rows. The harness
// overrides the log dir via SLAB_LOG_DIR (set by playwright.config.ts) so
// the parser tails a path the test can append to. We then assert via the
// admin GET /api/sites/{id}/visit-events that the row landed.

test.describe('analytics: nginx log tail', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('appended JSON log line is ingested into visit_events', async ({ adminApi }) => {
		site = await createSite(adminApi, { siteType: 'b2b', structureType: 'one-pager' });

		// Enable slab tracking - the manager's settings hook spawns a parser
		// for this site as soon as the bulk-upsert returns.
		const enableRes = await adminApi.put(u(`/api/sites/${site.id}/settings/bulk`), {
			data: [
				{ category: 'analytics', key: 'slab_tracking_enabled', value: 'true' }
			]
		});
		expect(enableRes.ok()).toBeTruthy();

		// Make sure the dir + file exist before we append. Parser tolerates ENOENT
		// but seeks to end-of-file on first open, so any line written before it
		// could open would be missed.
		// File is keyed by slug, not site_id, because the production host nginx
		// derives slug from $host and can't see internal site_ids. The parser
		// keys visit_events rows by site_id at write time.
		const logPath = path.join(E2E_CONSTANTS.NGINX_LOG_DIR, `${site.slug}.json.log`);
		await fs.mkdir(path.dirname(logPath), { recursive: true });
		await fs.writeFile(logPath, '', { flag: 'a' });

		// Give the parser a moment to start tailing the file. pollInterval is
		// 250ms; 1s is comfortably above that.
		await new Promise((r) => setTimeout(r, 1_000));

		const marker = `/e2e-marker-${Date.now()}`;
		const line =
			JSON.stringify({
				ts: new Date().toISOString(),
				site_id: site.id,
				ip: '127.0.0.1',
				ua: 'e2e/1.0',
				al: 'en',
				path: marker,
				status: 200,
				ms: 0.005,
				ref: ''
			}) + '\n';
		await fs.appendFile(logPath, line);

		// Poll the admin endpoint for up to 10s. Parser pollInterval is 250ms,
		// so in practice the row lands within ~1s.
		await expect
			.poll(
				async () => {
					const r = await adminApi.get(u(`/api/sites/${site.id}/visit-events?limit=20`));
					if (!r.ok()) return null;
					const rows = (await r.json()) as Array<{ path: string }>;
					return rows.find((row) => row.path === marker) ?? null;
				},
				{ timeout: 10_000, intervals: [200, 400, 800] }
			)
			.not.toBeNull();
	});
});
