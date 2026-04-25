import { apiDelete, apiGet, apiPatch, apiPost, apiPut } from './client';
import type { Block } from './types';

export interface BlocksListResponse {
	blocks: Block[];
}

export interface CreateBlockInput {
	block_type: string;
	data?: unknown;
	style?: unknown;
	sort_order?: number;
}

export interface UpdateBlockPatch {
	block_type?: string;
	data?: unknown;
	style?: unknown;
	sort_order?: number;
	is_visible?: boolean;
}

export interface BulkSaveBlock {
	id?: string;
	block_type: string;
	sort_order: number;
	data?: unknown;
	style?: unknown;
	is_visible: boolean;
}

export function list(siteID: string, pageID: string): Promise<BlocksListResponse> {
	return apiGet<BlocksListResponse>(`/sites/${siteID}/pages/${pageID}/blocks`);
}

export function create(siteID: string, pageID: string, input: CreateBlockInput): Promise<Block> {
	return apiPost<Block>(`/sites/${siteID}/pages/${pageID}/blocks`, input);
}

export function update(
	siteID: string,
	pageID: string,
	blockID: string,
	patch: UpdateBlockPatch
): Promise<Block> {
	return apiPatch<Block>(`/sites/${siteID}/pages/${pageID}/blocks/${blockID}`, patch);
}

export function remove(siteID: string, pageID: string, blockID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/pages/${pageID}/blocks/${blockID}`);
}

export function reorder(
	siteID: string,
	pageID: string,
	sortedBlockIDs: string[]
): Promise<{ status: string }> {
	const order = sortedBlockIDs.map((id, idx) => ({ id, sort_order: idx }));
	return apiPost<{ status: string }>(`/sites/${siteID}/pages/${pageID}/blocks/reorder`, {
		order
	});
}

export function bulkSave(
	siteID: string,
	pageID: string,
	blocks: BulkSaveBlock[]
): Promise<BlocksListResponse> {
	return apiPut<BlocksListResponse>(`/sites/${siteID}/pages/${pageID}/blocks/bulk`, { blocks });
}
