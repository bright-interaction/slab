// GitHub design references. Backend lives in internal/handlers/design_refs.go.
import { apiGet, apiPost, apiDelete } from './client';

export interface DesignReference {
	id: string;
	site_id: string;
	url: string;
	label: string;
	ref_type: 'design-system' | 'component-library' | 'reference-site';
	fetched_json: string;
	fetched_at: string;
	created_at: string;
}

export interface ListDesignReferencesResponse {
	references: DesignReference[];
}

export function list(siteID: string): Promise<ListDesignReferencesResponse> {
	return apiGet<ListDesignReferencesResponse>(`/sites/${siteID}/design-references`);
}

export function create(
	siteID: string,
	url: string,
	label: string,
	refType: DesignReference['ref_type']
): Promise<DesignReference> {
	return apiPost<DesignReference>(`/sites/${siteID}/design-references`, {
		url,
		label,
		ref_type: refType
	});
}

export function refresh(siteID: string, refID: string): Promise<DesignReference> {
	return apiPost<DesignReference>(`/sites/${siteID}/design-references/${refID}/refresh`);
}

export function remove(siteID: string, refID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/design-references/${refID}`);
}
