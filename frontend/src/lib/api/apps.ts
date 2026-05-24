import { apiGet, apiPost, apiDelete } from './client';

// Sprint 4 slice A (MCP-as-Apps platform, 2026-05-24): API wrappers
// for the marketplace + per-site install ledger.

export interface CredentialField {
	key: string;
	label: string;
	kind: 'secret' | 'string' | 'url';
	required: boolean;
	placeholder?: string;
	help?: string;
}

export interface App {
	id: string;
	slug: string;
	name: string;
	description: string;
	category: string;
	publisher: string;
	icon_url: string;
	docs_url: string;
	version: string;
	credential_fields: CredentialField[];
	is_curated: boolean;
}

export interface SiteInstall {
	site_id: string;
	app_id: string;
	status: 'active' | 'revoked' | 'error';
	installed_by: string;
	last_used_at: string;
	notes: string;
	updated_at: string;
	app: App;
	credentials_set: string[];
}

export interface InstallInput {
	credentials: Record<string, string>;
	notes?: string;
}

export async function listMarketplace(): Promise<App[]> {
	const r = await apiGet<{ apps: App[]; count: number }>(`/api/apps`);
	return r.apps;
}

export function getApp(appID: string): Promise<App> {
	return apiGet(`/api/apps/${appID}`);
}

export async function listSiteInstalls(siteID: string): Promise<SiteInstall[]> {
	const r = await apiGet<{ installs: SiteInstall[]; count: number }>(`/api/sites/${siteID}/apps`);
	return r.installs;
}

export function install(siteID: string, appID: string, input: InstallInput): Promise<SiteInstall> {
	return apiPost(`/api/sites/${siteID}/apps/${appID}`, input);
}

export function revoke(siteID: string, appID: string): Promise<void> {
	return apiDelete(`/api/sites/${siteID}/apps/${appID}`);
}
