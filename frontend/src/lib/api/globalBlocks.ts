import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { GlobalBlock } from './types';

export interface GlobalBlocksListResponse {
	global_blocks: GlobalBlock[];
}

export interface CreateGlobalBlockInput {
	name: string;
	slot: 'header' | 'footer';
	block_type: string;
	data?: unknown;
	style?: unknown;
	sort_order?: number;
}

export interface UpdateGlobalBlockPatch {
	name?: string;
	slot?: 'header' | 'footer';
	block_type?: string;
	data?: unknown;
	style?: unknown;
	sort_order?: number;
	is_active?: boolean;
}

export function list(siteID: string): Promise<GlobalBlocksListResponse> {
	return apiGet<GlobalBlocksListResponse>(`/sites/${siteID}/global-blocks`);
}

export function get(siteID: string, blockID: string): Promise<GlobalBlock> {
	return apiGet<GlobalBlock>(`/sites/${siteID}/global-blocks/${blockID}`);
}

export function create(siteID: string, input: CreateGlobalBlockInput): Promise<GlobalBlock> {
	return apiPost<GlobalBlock>(`/sites/${siteID}/global-blocks`, input);
}

export function update(
	siteID: string,
	blockID: string,
	patch: UpdateGlobalBlockPatch
): Promise<GlobalBlock> {
	return apiPatch<GlobalBlock>(`/sites/${siteID}/global-blocks/${blockID}`, patch);
}

export function remove(siteID: string, blockID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/global-blocks/${blockID}`);
}
