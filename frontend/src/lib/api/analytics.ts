// Backend lives in internal/handlers/analytics.go.
// Endpoints:
//   GET /sites/{id}/analytics/overview?since=1d|7d|30d|90d|all
//   GET /sites/{id}/analytics/sessions?identified=true&limit=25
//   GET /sites/{id}/analytics/conversion-paths?limit=5
//   GET /sites/{id}/analytics/tracked-fields

import { apiGet } from './client';

export type TopPage = {
	path: string;
	count: number;
};

export type TopReferer = {
	referer: string;
	count: number;
};

export type NameCount = {
	name: string;
	count: number;
};

export type TimePoint = {
	bucket: string;
	count: number;
	unique_count: number;
};

export type EngagementRow = {
	path: string;
	samples: number;
	avg_time_on_page_ms: number;
	avg_scroll_pct: number;
};

export type EngagementSummary = {
	samples: number;
	avg_time_on_page_ms: number;
	avg_scroll_pct: number;
	dark_pct: number;
	reduced_motion_pct: number;
	by_path: EngagementRow[];
	viewport_buckets: NameCount[];
};

export type AnalyticsOverview = {
	since: string;
	visits: number;
	unique_visitors: number;
	identified_count: number;
	live_visitors: number;
	top_pages: TopPage[];
	top_referers: TopReferer[];
	top_browsers: NameCount[];
	top_os: NameCount[];
	top_devices: NameCount[];
	top_countries: NameCount[];
	top_languages: NameCount[];
	top_utm_sources: NameCount[];
	top_utm_campaigns: NameCount[];
	time_series: TimePoint[];
	bucket_unit: 'day' | 'hour';
	engagement: EngagementSummary;
};

export type VisitSession = {
	id: string;
	site_id: string;
	fingerprint: string;
	visitor_id: string;
	email: string;
	consent_method: string;
	started_at: string;
	last_seen_at: string;
	page_count: number;
	identified_at: string;
};

export type ConversionStep = {
	path: string;
	ts: string;
};

export type ConversionPath = {
	steps: ConversionStep[];
	converted_at: string;
	email: string;
};

export type SinceRange = '1d' | '7d' | '30d' | '90d' | 'all';

export interface ListSessionsOpts {
	identified?: boolean;
	limit?: number;
	offset?: number;
}

export type TrackedField = {
	field: string;
	stored: string;
	source: string;
	purpose: string;
};

export type NotTrackedField = {
	field: string;
	reason: string;
};

export type TrackedFieldsResponse = {
	tracked: TrackedField[];
	not_tracked: NotTrackedField[];
};

export function getOverview(siteID: string, since: SinceRange = '7d'): Promise<AnalyticsOverview> {
	const qs = new URLSearchParams({ since });
	return apiGet<AnalyticsOverview>(`/sites/${siteID}/analytics/overview?${qs.toString()}`);
}

export function listSessions(
	siteID: string,
	opts: ListSessionsOpts = {}
): Promise<{ sessions: VisitSession[] }> {
	const qs = new URLSearchParams();
	if (typeof opts.identified === 'boolean') qs.set('identified', opts.identified ? 'true' : 'false');
	if (typeof opts.limit === 'number') qs.set('limit', String(opts.limit));
	if (typeof opts.offset === 'number') qs.set('offset', String(opts.offset));
	const suffix = qs.toString() ? `?${qs.toString()}` : '';
	return apiGet<{ sessions: VisitSession[] }>(`/sites/${siteID}/analytics/sessions${suffix}`);
}

export function listConversionPaths(
	siteID: string,
	limit = 20
): Promise<{ paths: ConversionPath[] }> {
	const qs = new URLSearchParams({ limit: String(limit) });
	return apiGet<{ paths: ConversionPath[] }>(
		`/sites/${siteID}/analytics/conversion-paths?${qs.toString()}`
	);
}

export function getTrackedFields(siteID: string): Promise<TrackedFieldsResponse> {
	return apiGet<TrackedFieldsResponse>(`/sites/${siteID}/analytics/tracked-fields`);
}
