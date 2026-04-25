import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	rand,
	type Site
} from '../../fixtures/data';
import { E2E_CONSTANTS } from '../../playwright.config';

// 1x1 transparent PNG
const PNG_BASE64 =
	'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=';
const PNG_BYTES = Buffer.from(PNG_BASE64, 'base64');
// 8x8 red JPEG (valid header so imaging.Sniff returns image/jpeg)
const JPEG_BASE64 =
	'/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAAIAAgDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAn/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFAEBAAAAAAAAAAAAAAAAAAAAAP/EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAMAwEAAhEDEQA/AL+AB//Z';
const JPEG_BYTES = Buffer.from(JPEG_BASE64, 'base64');

test.describe('agent-api: media uploads', () => {
	const cleanup: string[] = [];
	let site: Site;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		const key = await createAgentKey(adminApi, site.id, ['read', 'write', 'build']);
		agentKey = key.key;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('POST /api/agent/media uploads multipart file', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/media'), {
				multipart: {
					file: { name: `${rand('img')}.png`, mimeType: 'image/png', buffer: PNG_BYTES }
				},
	
			});
			expect(res.status()).toBe(201);
			const media = await res.json();
			expect(media.id).toBeTruthy();
			expect(media.mime_type).toMatch(/^image\//);
		} finally {
			await ctx.dispose();
		}
	});

	test('POST /api/agent/media/from-base64 accepts data URL', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/media/from-base64'), {
				data: {
					data: `data:image/png;base64,${PNG_BASE64}`,
					mime_type: 'image/png',
					filename: `${rand('img')}.png`,
					alt_text: 'b64 test'
				}
			});
			expect(res.status()).toBe(201);
			const media = await res.json();
			expect(media.id).toBeTruthy();
			expect(media.alt_text).toBe('b64 test');
		} finally {
			await ctx.dispose();
		}
	});

	test('POST /api/agent/media/from-base64 accepts raw base64', async () => {
		// The fake JPEG_BYTES at the top of this file aren't a real JPEG - the
		// imaging pipeline rejects them. Use the valid 1x1 PNG payload instead;
		// the only thing we're verifying here is the raw-base64 (no data: URL
		// prefix) code path of the from-base64 handler.
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/media/from-base64'), {
				data: {
					data: PNG_BASE64,
					mime_type: 'image/png',
					filename: `${rand('img')}.png`
				}
			});
			expect(res.status()).toBe(201);
			const media = await res.json();
			expect(media.id).toBeTruthy();
		} finally {
			await ctx.dispose();
		}
	});

	test('POST /api/agent/media/from-url fetches from stub server', async () => {
		// The stub serves /any-path returning ok json. We need an actual image.
		// Use the cookieproof stub's /snippet.js route and expect 4xx (not an image),
		// OR rely on the stub returning something. Since we don't have an image
		// endpoint on stubs, post a base64 data URL instead via from-base64.
		// Here we test that from-url properly rejects non-200 / non-image responses.
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/media/from-url'), {
				data: {
					url: `${E2E_CONSTANTS.STUB_COOKIEPROOF}/__health`,
					filename: 'health.png'
				}
			});
			// __health returns "ok" (text/plain); imaging.Sniff returns "" so 4xx.
			expect(res.status()).toBeGreaterThanOrEqual(400);
			expect(res.status()).toBeLessThan(500);
		} finally {
			await ctx.dispose();
		}
	});

	test('uploaded media has variants_json populated', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/media/from-base64'), {
				data: {
					data: PNG_BASE64,
					mime_type: 'image/png',
					filename: `${rand('var')}.png`
				}
			});
			expect(res.status()).toBe(201);
			const media = await res.json();
			// variants_json should be a non-empty JSON array
			expect(media.variants_json).toBeTruthy();
			const variants = typeof media.variants_json === 'string'
				? JSON.parse(media.variants_json)
				: media.variants_json;
			expect(Array.isArray(variants)).toBe(true);
			expect(variants.length).toBeGreaterThan(0);
		} finally {
			await ctx.dispose();
		}
	});

	test('GET /api/agent/media lists media for site', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.get(u('/api/agent/media'));
			expect(res.ok()).toBe(true);
			const list = await res.json();
			expect(Array.isArray(list)).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});
});
