import { defineConfig } from '@playwright/test';
import path from 'node:path';

const TEST_PORT = process.env.TEST_PORT ?? '18290';
const BASE_URL = process.env.E2E_BASE_URL ?? `http://127.0.0.1:${TEST_PORT}`;
const DATA_DIR = process.env.E2E_DATA_DIR ?? `/tmp/atomicsite-e2e`;

export default defineConfig({
	testDir: './specs',
	fullyParallel: false,
	workers: 1,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? [['html', { open: 'never' }], ['list']] : 'list',
	timeout: 60_000,
	expect: { timeout: 8_000 },

	use: {
		baseURL: BASE_URL,
		trace: 'on-first-retry',
		video: 'retain-on-failure',
		screenshot: 'only-on-failure',
		actionTimeout: 10_000,
		navigationTimeout: 15_000
	},

	webServer: {
		// Wipe the test DB on every run so the seed step always starts fresh
		// (the slug uniqueness check would 409 on a re-run otherwise).
		command: `bash -lc 'rm -rf "$DATA_DIR" && ./bin/atomicsite'`,
		cwd: path.resolve(__dirname, '../..'),
		url: `${BASE_URL}/api/health`,
		reuseExistingServer: false,
		timeout: 60_000,
		stdout: 'pipe',
		stderr: 'pipe',
		env: {
			DATA_DIR,
			DB_PATH: `${DATA_DIR}/db.sqlite`,
			PORT: TEST_PORT
		}
	}
});
