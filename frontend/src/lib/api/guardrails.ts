import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type { BoolInt, GuardrailRule } from './types';

export interface CreateGuardrailInput {
	rule_type: string;
	target: string;
	value: string;
	severity?: string;
}

export interface UpdateGuardrailPatch {
	rule_type?: string;
	target?: string;
	value?: string;
	severity?: string;
	is_active?: BoolInt;
}

export function list(siteID: string): Promise<GuardrailRule[]> {
	return apiGet<GuardrailRule[]>(`/sites/${siteID}/guardrails`);
}

export function get(siteID: string, ruleID: string): Promise<GuardrailRule> {
	return apiGet<GuardrailRule>(`/sites/${siteID}/guardrails/${ruleID}`);
}

export function create(siteID: string, input: CreateGuardrailInput): Promise<GuardrailRule> {
	return apiPost<GuardrailRule>(`/sites/${siteID}/guardrails`, input);
}

export function update(
	siteID: string,
	ruleID: string,
	patch: UpdateGuardrailPatch
): Promise<GuardrailRule> {
	return apiPatch<GuardrailRule>(`/sites/${siteID}/guardrails/${ruleID}`, patch);
}

export function remove(siteID: string, ruleID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/guardrails/${ruleID}`);
}
