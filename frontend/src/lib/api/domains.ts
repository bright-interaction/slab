import { apiDelete, apiGet, apiPost } from './client';

// Mirrors store.SiteDomain from the Go side. Status moves through:
//
//   pending -> verified -> cert_ready -> live
//                                     \-> error
//
// The admin UI shows status, last_check_at, and last_error verbatim
// so editors can self-debug DNS misconfigs without opening logs.
export interface SiteDomain {
	id: string;
	site_id: string;
	hostname: string;
	status: 'pending' | 'verified' | 'cert_ready' | 'live' | 'error';
	verify_token: string;
	cert_path: string;
	last_check_at: string;
	last_error: string;
	is_canonical: 0 | 1;
	created_at: string;
	updated_at: string;
}

export interface ListDomainsResponse {
	domains: SiteDomain[];
}

export function list(siteID: string): Promise<ListDomainsResponse> {
	return apiGet<ListDomainsResponse>(`/sites/${siteID}/domains`);
}

export function create(
	siteID: string,
	hostname: string,
	isCanonical = false
): Promise<SiteDomain> {
	return apiPost<SiteDomain>(`/sites/${siteID}/domains`, {
		hostname,
		is_canonical: isCanonical
	});
}

export function setCanonical(siteID: string, domainID: string): Promise<SiteDomain> {
	return apiPost<SiteDomain>(`/sites/${siteID}/domains/${domainID}/canonical`, {});
}

export function refresh(siteID: string, domainID: string): Promise<{ status: string }> {
	return apiPost<{ status: string }>(`/sites/${siteID}/domains/${domainID}/refresh`, {});
}

export function remove(siteID: string, domainID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/domains/${domainID}`);
}
