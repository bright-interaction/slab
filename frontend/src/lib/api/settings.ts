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

// CookieDeclaration mirrors internal/builder/cookieproof.go CookieDeclaration.
// Used by the Cookies admin page to render the merged preset + user-edited
// declaration table.
export interface CookieDeclaration {
	category: string;
	name: string;
	provider?: string;
	purpose?: string;
	expiry?: string;
}

// CookieTranslation is one language's worth of overridable banner copy.
// All fields optional; the widget falls back to its built-in translations
// for any field left empty. Mirrors the en.ts/sv.ts/etc. shape in
// CookieProof/src/i18n/.
export interface CookieTranslation {
	title?: string;
	description?: string;
	accept?: string;
	reject?: string;
	customize?: string;
}

// COOKIE_LANGUAGES is the list of locales the embedded CookieProof widget
// ships translations for (CookieProof/src/i18n/index.ts). The admin lets
// the operator pick a subset for the language-selector dropdown and
// override the strings per language.
export const COOKIE_LANGUAGES: { code: string; label: string }[] = [
	{ code: 'en', label: 'English' },
	{ code: 'sv', label: 'Svenska' },
	{ code: 'de', label: 'Deutsch' },
	{ code: 'fr', label: 'Français' },
	{ code: 'es', label: 'Español' },
	{ code: 'nl', label: 'Nederlands' },
	{ code: 'no', label: 'Norsk' },
	{ code: 'da', label: 'Dansk' },
	{ code: 'fi', label: 'Suomi' },
	{ code: 'pt', label: 'Português' },
	{ code: 'it', label: 'Italiano' },
	{ code: 'pl', label: 'Polski' },
	{ code: 'ja', label: '日本語' }
];

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

// getCookiePresets returns the cookie declarations atomicsite auto-derives
// from the site's enabled trackers (GA4, language settings, etc.). The
// admin Cookies page renders these alongside the user-edited list so
// the operator sees the full disclosure surface without re-entering
// preset data. Computed on the fly server-side from current settings.
export function getCookiePresets(siteID: string): Promise<CookieDeclaration[]> {
	return apiGet<CookieDeclaration[]>(`/sites/${siteID}/settings/cookie-presets`);
}
