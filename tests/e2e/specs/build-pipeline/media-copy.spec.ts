import { deflateSync } from 'node:zlib';
import { test, expect } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	triggerBuildAndWait,
	uploadMedia
} from '../../fixtures/data';
import { createPublishedPageWithBlock, distDir, distExists } from './_helpers';
import { readdirSync } from 'node:fs';
import path from 'node:path';

/**
 * Generate a minimal valid PNG of the given size and solid RGB color.
 * Used to feed the imaging pipeline an image larger than every variant width
 * (320/640/1280/1920) so we can assert variant files actually emit.
 */
function makePng(width: number, height: number, rgb: [number, number, number]): Buffer {
	const sig = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
	const ihdr = Buffer.alloc(13);
	ihdr.writeUInt32BE(width, 0);
	ihdr.writeUInt32BE(height, 4);
	ihdr[8] = 8; // bit depth
	ihdr[9] = 2; // color type RGB
	ihdr[10] = 0; // compression
	ihdr[11] = 0; // filter
	ihdr[12] = 0; // interlace

	// Raw image data: each row = 1 filter byte (0) + width*3 bytes
	const rowLen = 1 + width * 3;
	const raw = Buffer.alloc(rowLen * height);
	for (let y = 0; y < height; y++) {
		raw[y * rowLen] = 0; // no filter
		for (let x = 0; x < width; x++) {
			const off = y * rowLen + 1 + x * 3;
			raw[off] = rgb[0];
			raw[off + 1] = rgb[1];
			raw[off + 2] = rgb[2];
		}
	}
	const compressed = deflateSync(raw);

	function chunk(type: string, data: Buffer): Buffer {
		const len = Buffer.alloc(4);
		len.writeUInt32BE(data.length, 0);
		const typeBuf = Buffer.from(type, 'ascii');
		const crcInput = Buffer.concat([typeBuf, data]);
		const crc = crc32(crcInput);
		const crcBuf = Buffer.alloc(4);
		crcBuf.writeUInt32BE(crc >>> 0, 0);
		return Buffer.concat([len, typeBuf, data, crcBuf]);
	}

	return Buffer.concat([
		sig,
		chunk('IHDR', ihdr),
		chunk('IDAT', compressed),
		chunk('IEND', Buffer.alloc(0))
	]);
}

const CRC_TABLE: number[] = (() => {
	const t = new Array<number>(256);
	for (let n = 0; n < 256; n++) {
		let c = n;
		for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
		t[n] = c >>> 0;
	}
	return t;
})();
function crc32(buf: Buffer): number {
	let c = 0xffffffff;
	for (let i = 0; i < buf.length; i++) c = CRC_TABLE[(c ^ buf[i]) & 0xff] ^ (c >>> 8);
	return (c ^ 0xffffffff) >>> 0;
}

test.describe('build-pipeline: media copy into dist/media/', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('uploaded media variants are copied into dist/media/<id>/', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		// 2000x2000 ensures every configured variant width (320/640/1280/1920)
		// is below origW and therefore actually emitted.
		const png = makePng(2000, 2000, [200, 100, 50]);
		const media = await uploadMedia(adminApi, site.id, {
			filename: 'banner.png',
			contentType: 'image/png',
			bytes: png
		});
		expect(media.id).toBeTruthy();

		await createPublishedPageWithBlock(adminApi, site.id, {
			slug: '/',
			title: 'Media Home'
		});

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		// Side effect: the per-media directory is hardlinked into dist/media/<mediaID>/
		const mediaDistDir = path.join(distDir(site.id), 'media', media.id);
		expect(distExists(site.id, `media/${media.id}`)).toBe(true);

		const files = readdirSync(mediaDistDir);
		// Original + at least one resized variant (.webp or source format) per width.
		const widths = [320, 640, 1280, 1920];
		for (const w of widths) {
			const hasWebp = files.some((f) => f === `${w}.webp`);
			const hasOrig = files.some((f) => f === `${w}.png` || f === `${w}.jpg` || f === `${w}.jpeg`);
			expect(hasWebp || hasOrig, `expected a ${w}px variant in ${files.join(',')}`).toBe(true);
		}
		// Original is also copied
		expect(files.some((f) => f.startsWith('original.'))).toBe(true);
	});
});
