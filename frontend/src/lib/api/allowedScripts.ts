import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { AllowedScript, BoolInt } from './types';

export interface CreateAllowedScriptInput {
	domain: string;
	purpose?: string;
}

export interface UpdateAllowedScriptPatch {
	domain?: string;
	purpose?: string;
	is_active?: BoolInt;
}

export interface CreateAllowedScriptResponse {
	id: string;
	domain: string;
}

export function list(siteID: string): Promise<AllowedScript[]> {
	return apiGet<AllowedScript[]>(`/sites/${siteID}/allowed-scripts`);
}

export function create(
	siteID: string,
	input: CreateAllowedScriptInput
): Promise<CreateAllowedScriptResponse> {
	return apiPost<CreateAllowedScriptResponse>(`/sites/${siteID}/allowed-scripts`, input);
}

export function update(
	siteID: string,
	scriptID: string,
	patch: UpdateAllowedScriptPatch
): Promise<{ status: string }> {
	return apiPatch<{ status: string }>(`/sites/${siteID}/allowed-scripts/${scriptID}`, patch);
}

export function remove(siteID: string, scriptID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/allowed-scripts/${scriptID}`);
}
