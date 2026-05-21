import { apiGet, apiPost } from './client';
import type { Timestamp } from './types';

export type EntityType = 'page' | 'block';

export interface Revision {
	id: string;
	site_id: string;
	entity_type: EntityType;
	entity_id: string;
	version_number: number;
	snapshot_json: string;
	change_summary: string;
	created_by: string;
	created_at: Timestamp;
}

interface ListResponse {
	revisions: Revision[];
	count: number;
}

interface RestoreResponse {
	entity_type: EntityType;
	entity_id: string;
	change_summary: string;
	entity: unknown;
	restored_from?: unknown;
}

export async function listRevisions(
	siteID: string,
	entityType: EntityType,
	entityID: string,
	limit = 50
): Promise<Revision[]> {
	const resp = await apiGet<ListResponse>(
		`/sites/${siteID}/revisions/${entityType}/${entityID}?limit=${limit}`
	);
	return resp.revisions;
}

export function getRevision(
	siteID: string,
	entityType: EntityType,
	entityID: string,
	version: number
): Promise<Revision> {
	return apiGet<Revision>(
		`/sites/${siteID}/revisions/${entityType}/${entityID}/${version}`
	);
}

export function restoreRevision(
	siteID: string,
	entityType: EntityType,
	entityID: string,
	version: number
): Promise<RestoreResponse> {
	return apiPost<RestoreResponse>(
		`/sites/${siteID}/revisions/${entityType}/${entityID}/${version}/restore`,
		{}
	);
}
