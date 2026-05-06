import { apiGet, apiPost, apiDelete } from './client';
import type { Timestamp } from './types';

export interface MissingURL {
	id: string;
	site_id: string;
	path: string;
	referer: string;
	hit_count: number;
	first_seen: Timestamp;
	last_seen: Timestamp;
}

export type MissingURLRedirectStatus = 301 | 302 | 307 | 308 | 410;

export interface CreateRedirectFromMissingRequest {
	to_path: string;
	status_code?: MissingURLRedirectStatus;
}

// Two outcomes from POST /missing-urls/{id}/redirect:
//  - "created": we wrote a new redirect + cleared the missing-url row.
//  - "already_existed": a redirect was already in place; we cleared the
//    missing-url row and surfaced the existing redirect's destination.
export interface CreateRedirectFromMissingResponse {
	status: 'created' | 'already_existed';
	redirect_id: string;
	from: string;
	to: string;
	status_code?: number;
}

export function listMissingURLs(siteID: string, limit = 50): Promise<MissingURL[]> {
	return apiGet<MissingURL[]>(`/sites/${siteID}/missing-urls?limit=${limit}`);
}

export function createRedirectFromMissing(
	siteID: string,
	missingID: string,
	body: CreateRedirectFromMissingRequest
): Promise<CreateRedirectFromMissingResponse> {
	return apiPost<CreateRedirectFromMissingResponse>(
		`/sites/${siteID}/missing-urls/${missingID}/redirect`,
		body
	);
}

export function dismissMissingURL(
	siteID: string,
	missingID: string
): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/missing-urls/${missingID}`);
}
