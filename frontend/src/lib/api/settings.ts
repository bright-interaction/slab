import { apiDelete, apiGet, apiPut } from './client';
import type { SiteSetting } from './types';

export type SettingsCategory =
	| 'security'
	| 'seo'
	| 'analytics'
	| 'general'
	| 'wizard_state'
	| 'server';

export interface SettingUpsertInput {
	category: string;
	key: string;
	value: string;
}

export function list(siteID: string): Promise<SiteSetting[]> {
	return apiGet<SiteSetting[]>(`/sites/${siteID}/settings`);
}

export function listByCategory(siteID: string, category: string): Promise<SiteSetting[]> {
	return apiGet<SiteSetting[]>(`/sites/${siteID}/settings/${category}`);
}

export function upsert(siteID: string, input: SettingUpsertInput): Promise<SiteSetting> {
	return apiPut<SiteSetting>(`/sites/${siteID}/settings`, input);
}

export function bulkUpsert(siteID: string, items: SettingUpsertInput[]): Promise<SiteSetting[]> {
	return apiPut<SiteSetting[]>(`/sites/${siteID}/settings/bulk`, items);
}

export function remove(siteID: string, settingID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/settings/${settingID}`);
}
