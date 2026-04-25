import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { BoolInt, KnowledgebaseEntry } from './types';

export interface CreateKnowledgebaseInput {
	category: string;
	title: string;
	content: string;
	sort_order?: number;
}

export interface UpdateKnowledgebasePatch {
	category?: string;
	title?: string;
	content?: string;
	is_active?: BoolInt;
	sort_order?: number;
}

export function list(siteID: string): Promise<KnowledgebaseEntry[]> {
	return apiGet<KnowledgebaseEntry[]>(`/sites/${siteID}/knowledgebase`);
}

export function get(siteID: string, entryID: string): Promise<KnowledgebaseEntry> {
	return apiGet<KnowledgebaseEntry>(`/sites/${siteID}/knowledgebase/${entryID}`);
}

export function create(siteID: string, input: CreateKnowledgebaseInput): Promise<KnowledgebaseEntry> {
	return apiPost<KnowledgebaseEntry>(`/sites/${siteID}/knowledgebase`, input);
}

export function update(
	siteID: string,
	entryID: string,
	patch: UpdateKnowledgebasePatch
): Promise<KnowledgebaseEntry> {
	return apiPatch<KnowledgebaseEntry>(`/sites/${siteID}/knowledgebase/${entryID}`, patch);
}

export function remove(siteID: string, entryID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/knowledgebase/${entryID}`);
}
