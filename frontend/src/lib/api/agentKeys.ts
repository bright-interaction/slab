import { apiDelete, apiGet, apiPost } from './client';
import type { AgentKey } from './types';

export interface CreateAgentKeyInput {
	name: string;
	capabilities?: string[];
}

export interface CreateAgentKeyResponse {
	id: string;
	name: string;
	key: string;
	note: string;
}

export function list(siteID: string): Promise<AgentKey[]> {
	return apiGet<AgentKey[]>(`/sites/${siteID}/agent-keys`);
}

export function create(siteID: string, input: CreateAgentKeyInput): Promise<CreateAgentKeyResponse> {
	return apiPost<CreateAgentKeyResponse>(`/sites/${siteID}/agent-keys`, input);
}

export function revoke(siteID: string, keyID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/agent-keys/${keyID}`);
}

export interface BootstrapResponse {
	key_id: string;
	key: string;
	key_name: string;
	base_url: string;
	site_id: string;
	site_name: string;
	site_domain: string;
	claude_md: string;
	env_file: string;
	smoke_test: string;
	note: string;
}

export function bootstrap(siteID: string, name?: string): Promise<BootstrapResponse> {
	return apiPost<BootstrapResponse>(`/sites/${siteID}/agent-bootstrap`, {
		name: name || 'Quick start key'
	});
}
