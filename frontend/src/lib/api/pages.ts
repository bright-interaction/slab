import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { Page } from './types';

export interface PagesListResponse {
	pages: Page[];
}

export interface CreatePageInput {
	title: string;
	slug?: string;
	layout?: string;
	show_in_nav?: boolean;
}

export interface UpdatePagePatch {
	title?: string;
	slug?: string;
	status?: string;
	meta_title?: string;
	meta_description?: string;
	og_image_id?: string;
	layout?: string;
	sort_order?: number;
	show_in_nav?: boolean;
	nav_label?: string;
	no_index?: boolean;
	canonical_url?: string;
}

export function list(siteID: string): Promise<PagesListResponse> {
	return apiGet<PagesListResponse>(`/sites/${siteID}/pages`);
}

export function get(siteID: string, pageID: string): Promise<Page> {
	return apiGet<Page>(`/sites/${siteID}/pages/${pageID}`);
}

export function create(siteID: string, input: CreatePageInput): Promise<Page> {
	return apiPost<Page>(`/sites/${siteID}/pages`, input);
}

export function update(siteID: string, pageID: string, patch: UpdatePagePatch): Promise<Page> {
	return apiPatch<Page>(`/sites/${siteID}/pages/${pageID}`, patch);
}

export function remove(siteID: string, pageID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/pages/${pageID}`);
}

export function reorder(siteID: string, sortedPageIDs: string[]): Promise<{ status: string }> {
	const order = sortedPageIDs.map((id, idx) => ({ id, sort_order: idx }));
	return apiPost<{ status: string }>(`/sites/${siteID}/pages/reorder`, { order });
}

export interface PagePreview {
	page_id: string;
	slug: string;
	title: string;
	astro: string;
}

export function previewSource(siteID: string, pageID: string): Promise<PagePreview> {
	return apiGet<PagePreview>(`/sites/${siteID}/pages/${pageID}/preview`);
}
