#!/usr/bin/env bun
/**
 * Patch playwright-core's fetch.js to handle Bun's path-only response.url.
 *
 * Background. Playwright's APIRequestContext parses Set-Cookie headers via
 * `_parseSetCookieHeader(responseUrl, ...)`, which calls `new URL(responseUrl)`.
 * Under Bun the response.url returned by node:http is the path-only string
 * (e.g. "/api/auth/login"), which throws ERR_INVALID_URL. We fall back to the
 * known-absolute request URL when response.url is not absolute.
 *
 * Idempotent. Re-runs safely.
 */
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const target = path.resolve(__dirname, '..', 'node_modules/playwright-core/lib/server/fetch.js');

if (!existsSync(target)) {
	console.log(`[patch-playwright] target not found, skipping: ${target}`);
	process.exit(0);
}

const before = `const cookies = this._parseSetCookieHeader(response.url || url.toString(), response.headers["set-cookie"]);`;
const after = `const cookies = this._parseSetCookieHeader((response.url && /^https?:\\/\\//.test(response.url)) ? response.url : url.toString(), response.headers["set-cookie"]);`;

const src = readFileSync(target, 'utf8');
if (src.includes(after)) {
	console.log('[patch-playwright] already patched');
	process.exit(0);
}
if (!src.includes(before)) {
	console.error('[patch-playwright] expected line not found in fetch.js - playwright-core version drifted');
	process.exit(1);
}
writeFileSync(target, src.replace(before, after), 'utf8');
console.log('[patch-playwright] patched fetch.js for Bun compat');
