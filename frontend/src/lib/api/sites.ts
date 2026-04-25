import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { Site } from './types';

export interface SitesListResponse {
	sites: Site[];
}

export interface CreateSiteInput {
	name: string;
	slug: string;
	domain?: string;
	primary_color?: string;
	secondary_color?: string;
	bg_color?: string;
	text_color?: string;
	font_heading?: string;
	font_body?: string;
	lang?: string;
}

export interface UpdateSitePatch {
	name?: string;
	slug?: string;
	domain?: string;
	status?: string;
	primary_color?: string;
	secondary_color?: string;
	bg_color?: string;
	text_color?: string;
	font_heading?: string;
	font_body?: string;
	meta_title?: string;
	meta_description?: string;
	og_image_id?: string;
	favicon_id?: string;
	ga4_id?: string;
	umami_id?: string;
	umami_url?: string;
	cookieproof_domain?: string;
	lang?: string;
}

export function list(): Promise<SitesListResponse> {
	return apiGet<SitesListResponse>('/sites');
}

export function get(siteID: string): Promise<Site> {
	return apiGet<Site>(`/sites/${siteID}`);
}

export function create(input: CreateSiteInput): Promise<Site> {
	return apiPost<Site>('/sites', input);
}

export function update(siteID: string, patch: UpdateSitePatch): Promise<Site> {
	return apiPatch<Site>(`/sites/${siteID}`, patch);
}

export function remove(siteID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}`);
}
