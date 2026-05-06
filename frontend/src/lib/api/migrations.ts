import { apiGet, apiPost, apiDelete } from './client';
import type { BoolInt, Timestamp } from './types';

// Source CMS this manifest was crawled from. Mirrors the four importers
// in internal/migration/. Empty defaults to "sitemap".
export type MigrationSourceType = 'sitemap' | 'wordpress' | 'webflow' | 'ghost';

// Status flow on the migrations row: pending -> ready -> applied | failed.
// "ready" is the post-crawl pre-apply state every UI workflow starts in.
export type MigrationStatus = 'pending' | 'ready' | 'applied' | 'failed';

// Subset returned by GET /migrations (drops the manifest_json blob to
// keep listing payload small). The detail endpoint returns the full
// Migration with manifest.
export interface MigrationSummary {
	id: string;
	source_url: string;
	source_type: MigrationSourceType | string;
	status: MigrationStatus | string;
	error: string;
	created_at: Timestamp;
	updated_at: Timestamp;
	stats: { pages?: number; collections?: number; media?: number };
}

export interface MigrationStats {
	pages_found: number;
	collections_found: number;
	items_found: number;
	media_found: number;
}

// Per-page entry in the parsed manifest. The handler returns this
// straight from the importer; the planner reads SourcePath to compute
// NewSlug.
export interface MigrationPage {
	source_path: string;
	title: string;
	meta_description: string;
	canonical_url: string;
	og_image_url: string;
	no_index: boolean;
	html: string;
}

export interface MigrationCollection {
	name: string;
	slug: string;
	schema: unknown[];
	items: MigrationCollectionItem[];
}

export interface MigrationCollectionItem {
	slug: string;
	title: string;
	locale: string;
	data: Record<string, unknown>;
	published_at: string;
}

export interface MigrationMediaEntry {
	source_url: string;
	alt: string;
	caption: string;
}

export interface MigrationManifest {
	stats: MigrationStats;
	pages: MigrationPage[];
	collections: MigrationCollection[];
	media: MigrationMediaEntry[];
}

export interface Migration {
	id: string;
	site_id: string;
	source_url: string;
	source_type: MigrationSourceType | string;
	status: MigrationStatus | string;
	error: string;
	created_at: Timestamp;
	updated_at: Timestamp;
	manifest: MigrationManifest;
}

// URL planner output. PageRewrite carries source_path -> new_slug plus
// a Skipped flag the operator flips via the planner-options toolbar.
export interface PageRewrite {
	source_path: string;
	new_slug: string;
	title: string;
	skipped: boolean;
	skip_reason?: string;
}

export interface RedirectPlan {
	from_path: string;
	to_path: string;
	status_code: number;
}

// Reasons a planned page collides with live state. Mostly slug overlap
// against existing pages but the planner also reports redirect-vs-page
// shadowing.
export interface URLConflict {
	source_path: string;
	new_slug: string;
	reason: string;
}

export interface URLPlan {
	pages: PageRewrite[];
	redirects: RedirectPlan[];
	conflicts: URLConflict[];
}

// Planner override knobs the operator can tweak before re-computing.
// SkipPaths drops a manifest source_path entirely.
export interface PlannerOptions {
	skip_paths?: string[];
	collapse_date_archives?: boolean;
	preserve_locale_prefix?: boolean;
}

// ApplyResult mirrors migration.ApplyResult (Sprint 4: *Updated counts
// added so the UI can show "3 created, 2 updated" on a refresh).
export interface ApplyResult {
	collections_created: number;
	items_created: number;
	items_updated?: number;
	pages_created: number;
	pages_updated?: number;
	redirects_created: number;
	redirects_updated?: number;
	media_uploaded: number;
	media_failed: number;
	warnings?: string[];
}

export interface StartMigrationRequest {
	source_type: MigrationSourceType | string;
	source_url: string;
	wordpress_auth_header?: string;
	webflow_site_id?: string;
	webflow_auth_token?: string;
	ghost_content_key?: string;
}

export interface StartMigrationResponse {
	id: string;
	status: MigrationStatus | string;
	manifest: MigrationManifest;
}

export interface ApplyMigrationRequest extends PlannerOptions {
	upsert?: boolean;
	confirm_conflicts?: boolean;
}

export interface VerifyCoverageRequest {
	urls: string[];
}

export interface VerifyCoverageResponse {
	will_200: string[];
	will_301: string[];
	will_410: string[];
	will_404: string[];
	malformed: string[];
}

export interface VerifyLiveRequest {
	urls: string[];
	deployed_domain: string;
}

export interface VerifyLiveAcceptedResponse {
	job_id: string;
	status: string; // typically "queued"
	total_urls: number;
}

// Status flow for an async verify-live run. Mirrors verify_jobs.status.
// Reaper transitions queued/running -> failed (error="server restart")
// on boot if a previous process crashed mid-run.
export type VerifyJobStatus = 'queued' | 'running' | 'done' | 'failed' | 'cancelled';

export interface VerifyJob {
	id: string;
	site_id: string;
	migration_id: string;
	status: VerifyJobStatus | string;
	total_urls: number;
	processed_urls: number;
	ok_count: number;
	fail_count: number;
	deployed_domain: string;
	error: string;
	created_at: Timestamp;
	started_at: Timestamp;
	completed_at: Timestamp;
}

export interface MigrationVerification {
	id: string;
	source_url: string;
	status_code: number;
	final_url: string;
	hops: number;
	ok: boolean;
	error: string;
	checked_at: Timestamp;
}

export interface ListVerificationsResponse {
	items: MigrationVerification[];
	ok: number;
	failed: number;
}

// ----------------------------------------------------------------
// API surface

export function listMigrations(siteID: string): Promise<MigrationSummary[]> {
	return apiGet<MigrationSummary[]>(`/sites/${siteID}/migrations`);
}

export function getMigration(siteID: string, migrationID: string): Promise<Migration> {
	return apiGet<Migration>(`/sites/${siteID}/migrations/${migrationID}`);
}

export function startMigration(
	siteID: string,
	body: StartMigrationRequest
): Promise<StartMigrationResponse> {
	return apiPost<StartMigrationResponse>(`/sites/${siteID}/migrations`, body);
}

export function planMigration(
	siteID: string,
	migrationID: string,
	body: PlannerOptions
): Promise<URLPlan> {
	return apiPost<URLPlan>(`/sites/${siteID}/migrations/${migrationID}/plan`, body);
}

export function applyMigration(
	siteID: string,
	migrationID: string,
	body: ApplyMigrationRequest
): Promise<ApplyResult> {
	return apiPost<ApplyResult>(`/sites/${siteID}/migrations/${migrationID}/apply`, body);
}

export function deleteMigration(siteID: string, migrationID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/migrations/${migrationID}`);
}

export function verifyCoverage(
	siteID: string,
	migrationID: string,
	body: VerifyCoverageRequest
): Promise<VerifyCoverageResponse> {
	return apiPost<VerifyCoverageResponse>(
		`/sites/${siteID}/migrations/${migrationID}/verify-coverage`,
		body
	);
}

export function verifyLive(
	siteID: string,
	migrationID: string,
	body: VerifyLiveRequest
): Promise<VerifyLiveAcceptedResponse> {
	return apiPost<VerifyLiveAcceptedResponse>(
		`/sites/${siteID}/migrations/${migrationID}/verify-live`,
		body
	);
}

export function listVerifyJobs(siteID: string, migrationID: string): Promise<{ items: VerifyJob[] }> {
	return apiGet<{ items: VerifyJob[] }>(
		`/sites/${siteID}/migrations/${migrationID}/verify-jobs`
	);
}

export function getVerifyJob(
	siteID: string,
	migrationID: string,
	jobID: string
): Promise<VerifyJob> {
	return apiGet<VerifyJob>(
		`/sites/${siteID}/migrations/${migrationID}/verify-jobs/${jobID}`
	);
}

export function cancelVerifyJob(
	siteID: string,
	migrationID: string,
	jobID: string
): Promise<{ status: string }> {
	return apiPost<{ status: string }>(
		`/sites/${siteID}/migrations/${migrationID}/verify-jobs/${jobID}/cancel`
	);
}

export function listVerifications(
	siteID: string,
	migrationID: string
): Promise<ListVerificationsResponse> {
	return apiGet<ListVerificationsResponse>(
		`/sites/${siteID}/migrations/${migrationID}/verifications`
	);
}

// Helper for UI affordances: a job is "active" if it can still progress.
// done | failed | cancelled are terminal.
export function isVerifyJobTerminal(status: string): boolean {
	return status === 'done' || status === 'failed' || status === 'cancelled';
}

// Used by ApplyButton to surface (BoolInt -> boolean) for upsert + such
// when reading existing rows, kept here so the type is colocated with
// the migrations API.
export type { BoolInt };
