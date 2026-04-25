/**
 * Shared helpers for build-pipeline + evaluation-engine specs.
 *
 * The builder writes its workspace at:
 *   ${DATA_DIR}/workspaces/<siteID>/
 * with the static output under:
 *   ${DATA_DIR}/workspaces/<siteID>/dist/
 * and nginx.conf at:
 *   ${DATA_DIR}/workspaces/<siteID>/nginx.conf
 *
 * Tests read these directly off disk after pollBuild reports success.
 */
import path from 'node:path';
import { existsSync, readFileSync } from 'node:fs';
import { type APIRequestContext } from '@playwright/test';
import { E2E_CONSTANTS } from '../../playwright.config';
import { createPage, createBlock, type Page_ } from '../../fixtures/data';
import { u } from '../../fixtures/auth';

export const DATA_DIR = E2E_CONSTANTS.DATA_DIR;

export function workspaceDir(siteID: string): string {
	return path.join(DATA_DIR, 'workspaces', siteID);
}

export function distDir(siteID: string): string {
	return path.join(workspaceDir(siteID), 'dist');
}

export function readDist(siteID: string, relPath: string): string {
	return readFileSync(path.join(distDir(siteID), relPath), 'utf8');
}

export function distExists(siteID: string, relPath: string): boolean {
	return existsSync(path.join(distDir(siteID), relPath));
}

export function readWorkspaceFile(siteID: string, relPath: string): string {
	return readFileSync(path.join(workspaceDir(siteID), relPath), 'utf8');
}

/**
 * Create a page and immediately publish it. The Create endpoint always sets
 * status=draft regardless of payload, so the PATCH is required for the page
 * to be picked up by ListPublishedPagesBySite during the build.
 */
export async function createPublishedPage(
	api: APIRequestContext,
	siteID: string,
	overrides: Partial<{ title: string; slug: string }> = {}
): Promise<Page_> {
	const page = await createPage(api, siteID, overrides);
	const res = await api.patch(u(`/api/sites/${siteID}/pages/${page.id}`), {
		data: { status: 'published' }
	});
	if (!res.ok()) {
		throw new Error(`publish page failed: ${res.status()} ${await res.text()}`);
	}
	const body = await res.json();
	return (body.page ?? body) as Page_;
}

/**
 * Create a published page with a known text block. Returns the page + the
 * block's text content so specs can assert it ends up in the rendered HTML.
 */
export async function createPublishedPageWithBlock(
	api: APIRequestContext,
	siteID: string,
	overrides: Partial<{ title: string; slug: string; text: string }> = {}
): Promise<{ page: Page_; text: string }> {
	const page = await createPublishedPage(api, siteID, overrides);
	const text = overrides.text ?? `e2e-marker-${page.slug}`;
	await createBlock(api, siteID, page.id, {
		block_type: 'text',
		data: { heading: 'Marker', text }
	});
	return { page, text };
}

/**
 * Upsert a single (category, key, value) setting.
 */
export async function upsertSetting(
	api: APIRequestContext,
	siteID: string,
	category: string,
	key: string,
	value: string
): Promise<void> {
	const res = await api.put(u(`/api/sites/${siteID}/settings`), {
		data: { category, key, value }
	});
	if (!res.ok()) {
		throw new Error(`upsertSetting failed: ${res.status()} ${await res.text()}`);
	}
}
