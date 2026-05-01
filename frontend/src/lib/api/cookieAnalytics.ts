// Frontend API helper for the stitched cookie-analytics surface that the
// Go server exposes via DuckDB ATTACH on the SQLite file. Each function
// matches one /api/sites/{siteID}/cookies/analytics/* endpoint.
//
// All endpoints accept an optional time window via from/to (RFC3339 or
// unix-millis). Defaults to the last 30 days when omitted.

import { apiGet } from './client';

function buildQuery(params: { [key: string]: unknown }): string {
	const usp = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v === undefined || v === null || v === '') continue;
		usp.set(k, String(v));
	}
	const s = usp.toString();
	return s ? `?${s}` : '';
}

export interface FunnelKPIs {
	unique_visitors: number;
	consenting_visitors: number;
	accept_all_visitors: number;
	reject_all_visitors: number;
	custom_visitors: number;
	gpc_visitors: number;
	do_not_sell_visitors: number;
	identified_visitors: number;
	abandoned_banner_visitors: number;
}

export interface FunnelResponse {
	site_id: string;
	from: string;
	to: string;
	kpis: FunnelKPIs;
}

export function funnel(siteID: string, params: { from?: string; to?: string } = {}): Promise<FunnelResponse> {
	return apiGet<FunnelResponse>(`/sites/${siteID}/cookies/analytics/funnel${buildQuery({ ...params })}`);
}

export interface PreConsentShare {
	total_pageviews: number;
	pageviews_before_or_no_consent: number;
	pageviews_after_consent: number;
	pageviews_after_reject: number;
}

export function preConsent(siteID: string, params: { from?: string; to?: string } = {}): Promise<PreConsentShare> {
	return apiGet<PreConsentShare>(`/sites/${siteID}/cookies/analytics/pre-consent${buildQuery({ ...params })}`);
}

export interface TimeToConsent {
	sample_size: number;
	p50_seconds: number;
	p90_seconds: number;
	mean_seconds: number;
}

export function timeToConsent(siteID: string, params: { from?: string; to?: string } = {}): Promise<TimeToConsent> {
	return apiGet<TimeToConsent>(`/sites/${siteID}/cookies/analytics/time-to-consent${buildQuery({ ...params })}`);
}

export interface EngagedAcceptRate {
	engaged_visitors: number;
	engaged_accepted: number;
	engaged_accept_rate_pct: number;
	not_engaged_visitors: number;
	not_engaged_accepted: number;
	not_engaged_accept_rate_pct: number;
}

export function engagedAcceptRate(siteID: string, params: { from?: string; to?: string } = {}): Promise<EngagedAcceptRate> {
	return apiGet<EngagedAcceptRate>(`/sites/${siteID}/cookies/analytics/engaged-accept-rate${buildQuery({ ...params })}`);
}

export interface AcceptingPage {
	path: string;
	visitors: number;
	accepted: number;
	accept_rate_pct: number;
}

export interface TopAcceptingPagesResponse {
	limit: number;
	rows: AcceptingPage[] | null;
}

export function topAcceptingPages(siteID: string, params: { from?: string; to?: string; limit?: number } = {}): Promise<TopAcceptingPagesResponse> {
	return apiGet<TopAcceptingPagesResponse>(`/sites/${siteID}/cookies/analytics/top-accepting-pages${buildQuery({ ...params })}`);
}

export interface ConsentDayRow {
	day: string;
	accept_all: number;
	reject_all: number;
	custom: number;
	gpc: number;
	dns: number;
	do_not_sell: number;
}

export interface DailySplitResponse {
	rows: ConsentDayRow[] | null;
}

export function dailySplit(siteID: string, params: { from?: string; to?: string } = {}): Promise<DailySplitResponse> {
	return apiGet<DailySplitResponse>(`/sites/${siteID}/cookies/analytics/daily${buildQuery({ ...params })}`);
}

export interface VisitorJourneyStep {
	ts_ms: number;
	kind: 'pageview' | 'engagement' | 'consent' | 'identified';
	path?: string;
	method?: string;
	categories?: string;
	gpc_active?: boolean;
	scroll_pct?: number;
	time_on_page_seconds?: number;
}

export interface VisitorJourneyResponse {
	site_id: string;
	fingerprint: string;
	steps: VisitorJourneyStep[];
}

export function visitorJourney(siteID: string, fingerprint: string, params: { from?: string; to?: string } = {}): Promise<VisitorJourneyResponse> {
	return apiGet<VisitorJourneyResponse>(`/sites/${siteID}/cookies/analytics/journey/${encodeURIComponent(fingerprint)}${buildQuery({ ...params })}`);
}
