import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { ListResponse, MediaVariant, Medium } from './types';

export interface MediaListOptions {
	limit?: number;
	offset?: number;
	q?: string;
}

export type MediaListResponse = ListResponse<Medium>;

export function list(siteID: string, opts: MediaListOptions = {}): Promise<MediaListResponse> {
	const params = new URLSearchParams();
	if (opts.limit !== undefined) params.set('limit', String(opts.limit));
	if (opts.offset !== undefined) params.set('offset', String(opts.offset));
	if (opts.q) params.set('q', opts.q);
	const qs = params.toString();
	return apiGet<MediaListResponse>(`/sites/${siteID}/media${qs ? `?${qs}` : ''}`);
}

export function get(siteID: string, mediaID: string): Promise<Medium> {
	return apiGet<Medium>(`/sites/${siteID}/media/${mediaID}`);
}

export function upload(siteID: string, file: File, altText?: string): Promise<Medium> {
	const fd = new FormData();
	fd.append('file', file);
	if (altText) fd.append('alt_text', altText);
	return apiPost<Medium>(`/sites/${siteID}/media`, fd);
}

export function update(
	siteID: string,
	mediaID: string,
	patch: { alt_text: string }
): Promise<Medium> {
	return apiPatch<Medium>(`/sites/${siteID}/media/${mediaID}`, patch);
}

export function remove(siteID: string, mediaID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/media/${mediaID}`);
}

export function mediaUrl(siteID: string, mediaID: string, variant: string): string {
	return `/media/${siteID}/${mediaID}/${variant}`;
}

export function parseMediaVariants(variants_json: string): MediaVariant[] {
	if (!variants_json) return [];
	try {
		const parsed = JSON.parse(variants_json);
		return Array.isArray(parsed) ? (parsed as MediaVariant[]) : [];
	} catch {
		return [];
	}
}
