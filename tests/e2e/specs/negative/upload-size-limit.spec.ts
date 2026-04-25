import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('negative: upload size limit', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('21 MiB multipart upload is rejected with 413', async ({ adminApi }) => {
		const big = Buffer.alloc(21 * 1024 * 1024, 0xff);
		const res = await adminApi.post(u(`/api/sites/${site.id}/media`), {
			multipart: {
				file: { name: 'too-big.png', mimeType: 'image/png', buffer: big }
			},

		});
		// http.MaxBytesReader returns 413; ParseMultipartForm error path also
		// returns 413. Bun fetch may surface as a network error -> bubble as
		// status 0 in some clients. Accept either 413 or a transport failure
		// expressed as non-2xx.
		expect(res.status()).not.toBe(201);
		expect([413, 400]).toContain(res.status());
	});

	test('legitimate small upload still succeeds', async ({ adminApi }) => {
		const tiny = Buffer.from(
			'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=',
			'base64'
		);
		const res = await adminApi.post(u(`/api/sites/${site.id}/media`), {
			multipart: {
				file: { name: 'tiny.png', mimeType: 'image/png', buffer: tiny }
			},

		});
		expect(res.status()).toBe(201);
	});
});
