import { apiGet, apiPatch } from './client';

export interface QuotaUsage {
	storage: { used_bytes: number; quota_bytes: number; pct: number };
	build_minutes: { used: number; quota: number; window_days: number; pct: number };
	blocked_on_overage: boolean;
}

export function getSiteQuota(siteID: string): Promise<QuotaUsage> {
	return apiGet<QuotaUsage>(`/sites/${siteID}/quota`);
}

export interface AdminMetrics {
	sqlite_open_conns: number;
	sqlite_in_use: number;
	sqlite_idle: number;
	analyticsdb: boolean;
	analyticsdb_healthy?: boolean;
	retention_last?: unknown;
	schema_version?: number;
	per_site_quotas?: PerSiteQuota[];
}

export interface PerSiteQuota {
	site_id: string;
	name: string;
	slug: string;
	storage_used_bytes: number;
	storage_quota_bytes: number;
	storage_pct: number;
	build_minutes_used: number;
	build_minutes_quota: number;
	build_minutes_pct: number;
	blocked_on_overage: boolean;
}

export function adminMetrics(): Promise<AdminMetrics> {
	return apiGet<AdminMetrics>('/admin/metrics');
}

export function patchAdminQuota(
	siteID: string,
	body: {
		storage_quota_bytes?: number;
		build_minutes_quota?: number;
		quota_overage_blocked?: boolean;
	}
): Promise<unknown> {
	return apiPatch(`/admin/sites/${siteID}/quota`, body);
}
