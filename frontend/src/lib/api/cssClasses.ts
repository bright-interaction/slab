import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { CssClass } from './types';

export interface CreateCssClassInput {
	name: string;
	css: string;
	category?: string;
	usage_note?: string;
	sort_order?: number;
}

export interface UpdateCssClassPatch {
	name?: string;
	category?: string;
	css?: string;
	usage_note?: string;
	sort_order?: number;
}

export function list(siteID: string): Promise<CssClass[]> {
	return apiGet<CssClass[]>(`/sites/${siteID}/css-classes`);
}

export function get(siteID: string, classID: string): Promise<CssClass> {
	return apiGet<CssClass>(`/sites/${siteID}/css-classes/${classID}`);
}

export function create(siteID: string, input: CreateCssClassInput): Promise<CssClass> {
	return apiPost<CssClass>(`/sites/${siteID}/css-classes`, input);
}

export function update(
	siteID: string,
	classID: string,
	patch: UpdateCssClassPatch
): Promise<CssClass> {
	return apiPatch<CssClass>(`/sites/${siteID}/css-classes/${classID}`, patch);
}

export function remove(siteID: string, classID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/css-classes/${classID}`);
}
