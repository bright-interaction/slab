import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock, readDist } from './_helpers';

test.describe('build-pipeline: robots + sitemap', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('robots.txt has User-agent + sitemap reference; sitemap has urlset/url entries', async ({
		adminApi
	}) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'Index' });
		await createPublishedPageWithBlock(adminApi, site.id, { slug: 'about', title: 'About' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		// Side effect: robots.txt
		const robots = readDist(site.id, 'robots.txt');
		expect(robots.toLowerCase()).toContain('user-agent:');
		expect(robots.toLowerCase()).toContain('sitemap:');

		// Side effect: sitemap-index.xml lists at least one sitemap shard
		const sitemapIndex = readDist(site.id, 'sitemap-index.xml');
		expect(sitemapIndex).toMatch(/<sitemapindex/);
		expect(sitemapIndex).toMatch(/<sitemap>/);

		// Astro emits the actual URL list at sitemap-0.xml under <urlset>.
		const sitemapShard = readDist(site.id, 'sitemap-0.xml');
		expect(sitemapShard).toMatch(/<urlset/);
		// At least one <url> entry per published page
		const urlMatches = sitemapShard.match(/<url>/g) ?? [];
		expect(urlMatches.length).toBeGreaterThanOrEqual(2);
	});
});
