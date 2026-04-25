import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { Component } from './types';

export interface CreateComponentInput {
	name: string;
	template: string;
	category?: string;
	props_schema?: unknown;
	css_classes?: string[];
	usage_note?: string;
}

export interface UpdateComponentPatch {
	name?: string;
	category?: string;
	template?: string;
	props_schema?: unknown;
	css_classes?: string[];
	usage_note?: string;
}

export function list(siteID: string): Promise<Component[]> {
	return apiGet<Component[]>(`/sites/${siteID}/components`);
}

export function get(siteID: string, componentID: string): Promise<Component> {
	return apiGet<Component>(`/sites/${siteID}/components/${componentID}`);
}

export function create(siteID: string, input: CreateComponentInput): Promise<Component> {
	return apiPost<Component>(`/sites/${siteID}/components`, input);
}

export function update(
	siteID: string,
	componentID: string,
	patch: UpdateComponentPatch
): Promise<Component> {
	return apiPatch<Component>(`/sites/${siteID}/components/${componentID}`, patch);
}

export function remove(siteID: string, componentID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/components/${componentID}`);
}
