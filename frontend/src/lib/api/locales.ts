import { apiGet, apiPost, apiPut, apiDelete } from './client';

// Sprint 3 (multilingual v1, 2026-05-22): API wrappers for the three
// CRUD surfaces the locale tabs strip + Settings -> Locales page
// drive. Mirrors the backend shapes in internal/handlers/locales.go.

export interface SiteLocale {
	locale: string;
	is_default: boolean;
	sort_order: number;
	updated_at: string;
}

export interface PageLocale {
	page_id: string;
	locale: string;
	slug_override: string;
	title: string;
	meta_title: string;
	meta_description: string;
	status: 'draft' | 'published' | 'archived';
	updated_at: string;
}

export interface BlockLocale {
	block_id: string;
	locale: string;
	data_json: string;
	is_visible: boolean;
	updated_at: string;
}

export interface SiteLocaleInput {
	locale: string;
	is_default: boolean;
	sort_order: number;
}

export interface PageLocaleInput {
	slug_override?: string;
	title?: string;
	meta_title?: string;
	meta_description?: string;
	status?: 'draft' | 'published' | 'archived';
}

export interface BlockLocaleInput {
	data_json: string;
	is_visible?: boolean;
}

// --- site_locales ---

export async function listSiteLocales(siteID: string): Promise<SiteLocale[]> {
	const r = await apiGet<{ locales: SiteLocale[]; count: number }>(`/api/sites/${siteID}/locales`);
	return r.locales;
}

export function upsertSiteLocale(siteID: string, input: SiteLocaleInput): Promise<SiteLocale> {
	return apiPost(`/api/sites/${siteID}/locales`, input);
}

export function deleteSiteLocale(siteID: string, locale: string): Promise<void> {
	return apiDelete(`/api/sites/${siteID}/locales/${encodeURIComponent(locale)}`);
}

// --- page_locales ---

export async function listPageLocales(siteID: string, pageID: string): Promise<PageLocale[]> {
	const r = await apiGet<{ page_locales: PageLocale[]; count: number }>(
		`/api/sites/${siteID}/pages/${pageID}/locales`
	);
	return r.page_locales;
}

export function upsertPageLocale(
	siteID: string,
	pageID: string,
	locale: string,
	input: PageLocaleInput
): Promise<PageLocale> {
	return apiPut(
		`/api/sites/${siteID}/pages/${pageID}/locales/${encodeURIComponent(locale)}`,
		input
	);
}

export function deletePageLocale(siteID: string, pageID: string, locale: string): Promise<void> {
	return apiDelete(`/api/sites/${siteID}/pages/${pageID}/locales/${encodeURIComponent(locale)}`);
}

// --- block_locales ---

export async function listBlockLocales(siteID: string, blockID: string): Promise<BlockLocale[]> {
	const r = await apiGet<{ block_locales: BlockLocale[]; count: number }>(
		`/api/sites/${siteID}/blocks/${blockID}/locales`
	);
	return r.block_locales;
}

// listBlockLocalesByPage returns every block_locale overlay for every
// block on the page in one fetch. Used by the per-locale page editor.
export async function listBlockLocalesByPage(
	siteID: string,
	pageID: string
): Promise<BlockLocale[]> {
	const r = await apiGet<{ block_locales: BlockLocale[]; count: number }>(
		`/api/sites/${siteID}/pages/${pageID}/block-locales`
	);
	return r.block_locales;
}

export function upsertBlockLocale(
	siteID: string,
	blockID: string,
	locale: string,
	input: BlockLocaleInput
): Promise<BlockLocale> {
	return apiPut(
		`/api/sites/${siteID}/blocks/${blockID}/locales/${encodeURIComponent(locale)}`,
		input
	);
}

export function deleteBlockLocale(siteID: string, blockID: string, locale: string): Promise<void> {
	return apiDelete(`/api/sites/${siteID}/blocks/${blockID}/locales/${encodeURIComponent(locale)}`);
}
