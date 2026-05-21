import { apiGet, apiPost, apiDelete } from './client';
import type { Timestamp } from './types';

export type ClarificationStatus = 'pending' | 'resolved' | 'cancelled';

export interface Clarification {
	id: string;
	site_id: string;
	requested_by: string;
	question: string;
	context: string;
	options: string[];
	status: ClarificationStatus;
	resolution_option: number;
	resolution_text: string;
	resolved_by: string;
	created_at: Timestamp;
	resolved_at: Timestamp;
}

interface ListResponse {
	clarifications: Clarification[];
	count: number;
}

export async function listClarifications(
	siteID: string,
	opts: { status?: 'pending' | 'all'; limit?: number } = {}
): Promise<Clarification[]> {
	const params = new URLSearchParams();
	if (opts.status) params.set('status', opts.status);
	if (opts.limit) params.set('limit', String(opts.limit));
	const qs = params.toString() ? `?${params.toString()}` : '';
	const resp = await apiGet<ListResponse>(`/sites/${siteID}/clarifications${qs}`);
	return resp.clarifications;
}

export function getClarification(siteID: string, id: string): Promise<Clarification> {
	return apiGet<Clarification>(`/sites/${siteID}/clarifications/${id}`);
}

export function resolveClarification(
	siteID: string,
	id: string,
	body: { option: number; text: string }
): Promise<Clarification> {
	return apiPost<Clarification>(`/sites/${siteID}/clarifications/${id}/resolve`, body);
}

export function cancelClarification(siteID: string, id: string): Promise<Clarification> {
	return apiDelete<Clarification>(`/sites/${siteID}/clarifications/${id}`);
}
