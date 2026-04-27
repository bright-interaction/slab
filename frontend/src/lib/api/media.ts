import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { ListResponse, MediaFolderListResponse, MediaVariant, Medium } from './types';

export interface MediaListOptions {
	limit?: number;
	offset?: number;
	q?: string;
	folder?: string;
}

export type MediaListResponse = ListResponse<Medium> & { folder?: string };

export function list(siteID: string, opts: MediaListOptions = {}): Promise<MediaListResponse> {
	const params = new URLSearchParams();
	if (opts.limit !== undefined) params.set('limit', String(opts.limit));
	if (opts.offset !== undefined) params.set('offset', String(opts.offset));
	if (opts.q) params.set('q', opts.q);
	if (opts.folder !== undefined) params.set('folder', opts.folder);
	const qs = params.toString();
	return apiGet<MediaListResponse>(`/sites/${siteID}/media${qs ? `?${qs}` : ''}`);
}

export function get(siteID: string, mediaID: string): Promise<Medium> {
	return apiGet<Medium>(`/sites/${siteID}/media/${mediaID}`);
}

export function upload(
	siteID: string,
	file: File,
	altText?: string,
	folder?: string
): Promise<Medium> {
	const fd = new FormData();
	fd.append('file', file);
	if (altText) fd.append('alt_text', altText);
	if (folder) fd.append('folder', folder);
	return apiPost<Medium>(`/sites/${siteID}/media`, fd);
}

export interface MediaPatch {
	alt_text?: string;
	folder?: string;
}

export function update(siteID: string, mediaID: string, patch: MediaPatch): Promise<Medium> {
	return apiPatch<Medium>(`/sites/${siteID}/media/${mediaID}`, patch);
}

export function remove(siteID: string, mediaID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/media/${mediaID}`);
}

export function listFolders(siteID: string): Promise<MediaFolderListResponse> {
	return apiGet<MediaFolderListResponse>(`/sites/${siteID}/media/folders`);
}

export function createFolder(
	siteID: string,
	name: string
): Promise<{ name: string; is_system: boolean }> {
	return apiPost<{ name: string; is_system: boolean }>(`/sites/${siteID}/media/folders`, { name });
}

export function deleteFolder(siteID: string, name: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(
		`/sites/${siteID}/media/folders/${encodeURIComponent(name)}`
	);
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
