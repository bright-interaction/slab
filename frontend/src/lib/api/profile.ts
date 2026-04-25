import { apiGet, apiPut } from './client';
import type { SiteProfile } from './types';

export interface ProfileUpsertInput {
	business_name?: string;
	registration_nr?: string;
	country?: string;
	contact_email?: string;
	contact_phone?: string;
	privacy_email?: string;
	security_email?: string;
	address_line1?: string;
	address_line2?: string;
	city?: string;
	postal_code?: string;
}

export type ProfileResponse = SiteProfile | { site_id: string };

export function get(siteID: string): Promise<ProfileResponse> {
	return apiGet<ProfileResponse>(`/sites/${siteID}/profile`);
}

export function upsert(siteID: string, profile: ProfileUpsertInput): Promise<SiteProfile> {
	return apiPut<SiteProfile>(`/sites/${siteID}/profile`, profile);
}
