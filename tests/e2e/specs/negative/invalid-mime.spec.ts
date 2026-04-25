import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('negative: invalid MIME / non-image upload', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('text file upload is rejected (sniffed, not Content-Type)', async ({ adminApi }) => {
		const text = Buffer.from('this is plain text, not an image\n');
		const res = await adminApi.post(u(`/api/sites/${site.id}/media`), {
			multipart: {
				file: { name: 'note.txt', mimeType: 'image/png', buffer: text }
			},

		});
		// imaging.Sniff returns "" -> 415 from the handler.
		expect([400, 415]).toContain(res.status());
	});

	test('HTML payload is rejected even with image extension', async ({ adminApi }) => {
		const html = Buffer.from('<html><body>not an image</body></html>');
		const res = await adminApi.post(u(`/api/sites/${site.id}/media`), {
			multipart: {
				file: { name: 'evil.png', mimeType: 'image/png', buffer: html }
			},

		});
		expect([400, 415]).toContain(res.status());
	});

	test('SVG file is rejected (XSS risk)', async ({ adminApi }) => {
		const svg = Buffer.from(
			'<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>'
		);
		const res = await adminApi.post(u(`/api/sites/${site.id}/media`), {
			multipart: {
				file: { name: 'evil.svg', mimeType: 'image/svg+xml', buffer: svg }
			},

		});
		expect([400, 415]).toContain(res.status());
	});
});
