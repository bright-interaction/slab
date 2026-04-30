import { apiGet } from './client';

export interface ConsentRecord {
	id: string;
	created_at: string;
	created_at_ms: number;
	domain: string;
	method: 'accept-all' | 'reject-all' | 'custom' | 'gpc' | 'dns' | 'do-not-sell';
	version: number;
	categories: Record<string, boolean | undefined>;
	page: string;
	referrer: string;
	user_agent: string;
	ip_hash_trunc: string;
	gpc_active: boolean;
	session_id?: string;
}

export interface ConsentListResponse {
	records: ConsentRecord[];
	total: number;
	limit: number;
	offset: number;
	from: number;
	to: number;
}

export interface ConsentDailyRow {
	day: string;
	by_method: Record<string, number>;
	total: number;
}

export interface ConsentStats {
	total: number;
	accepts: number;
	rejects: number;
	customs: number;
	gpcs: number;
	dns: number;
	do_not_sell: number;
	daily: ConsentDailyRow[];
	from: number;
	to: number;
}

export interface ListParams {
	from?: number;
	to?: number;
	method?: string;
	limit?: number;
	offset?: number;
}

function buildQuery(params: { [key: string]: unknown }): string {
	const usp = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v === undefined || v === null || v === '') continue;
		usp.set(k, String(v));
	}
	const s = usp.toString();
	return s ? `?${s}` : '';
}

export function list(siteID: string, params: ListParams = {}): Promise<ConsentListResponse> {
	return apiGet<ConsentListResponse>(`/sites/${siteID}/consent/records${buildQuery({ ...params })}`);
}

export function get(siteID: string, recordID: string): Promise<ConsentRecord> {
	return apiGet<ConsentRecord>(`/sites/${siteID}/consent/records/${recordID}`);
}

export function stats(siteID: string, params: { from?: number; to?: number } = {}): Promise<ConsentStats> {
	return apiGet<ConsentStats>(`/sites/${siteID}/consent/stats${buildQuery({ ...params })}`);
}

export function exportCsvUrl(siteID: string, params: { from?: number; to?: number } = {}): string {
	return `/api/sites/${siteID}/consent/export.csv${buildQuery({ ...params })}`;
}
