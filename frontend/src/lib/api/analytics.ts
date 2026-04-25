// TODO: Backend routes ship with Phase 10-1 / 10-2.
// Until they land, GET /sites/{id}/analytics/* will 404 and the page will
// fall back to its empty state. The shape below is the contract we expect.

import { apiGet } from './client';

export type TopPage = {
	path: string;
	count: number;
};

export type TopReferer = {
	referer: string;
	count: number;
};

export type AnalyticsOverview = {
	visits: number;
	unique_visitors: number;
	identified_count: number;
	top_pages: TopPage[];
	top_referers: TopReferer[];
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

export type SinceRange = '7d' | '30d' | '90d' | 'all';

export interface ListSessionsOpts {
	identified?: boolean;
	limit?: number;
	offset?: number;
}

function sinceToDays(since: SinceRange): string {
	if (since === 'all') return 'all';
	return since;
}

export function getOverview(
	siteID: string,
	since: SinceRange = '7d'
): Promise<AnalyticsOverview> {
	const qs = new URLSearchParams({ since: sinceToDays(since) });
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
	return apiGet<{ sessions: VisitSession[] }>(
		`/sites/${siteID}/analytics/sessions${suffix}`
	);
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
