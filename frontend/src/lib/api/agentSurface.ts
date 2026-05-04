// Admin Settings -> Agent inventory. Read-only.
//
// Backed by GET /api/sites/{siteID}/agent-surface (handlers/agent_surface.go),
// which delegates to mcp.Server.AgentSurface(). The payload mirrors the
// live MCP registry plus the embedded curriculum docs.

import { apiGet } from './client';

export interface AgentSurfaceCounts {
	tools: number;
	write_tools: number;
	resources: number;
	prompts: number;
	knowledge_docs: number;
}

export interface AgentToolMeta {
	name: string;
	description: string;
	requires_write: boolean;
}

export interface AgentResourceMeta {
	uri: string;
	name: string;
	description: string;
	mime_type: string;
}

export interface AgentPromptMeta {
	name: string;
	description: string;
}

export interface AgentKnowledgeMeta {
	slug: string;
	title: string;
	category: string;
	order: number;
	summary: string;
}

export interface AgentSurface {
	counts: AgentSurfaceCounts;
	tools: AgentToolMeta[];
	resources: AgentResourceMeta[];
	prompts: AgentPromptMeta[];
	knowledge: AgentKnowledgeMeta[];
	privacy_invariants: string[];
	endpoint: string;
	protocol_version: string;
	version: string;
}

export interface AgentSurfaceUnavailable {
	available: false;
	reason: string;
}

export type AgentSurfaceResponse = AgentSurface | AgentSurfaceUnavailable;

export function isAvailable(r: AgentSurfaceResponse): r is AgentSurface {
	return !('available' in r) || r.available !== false;
}

export function get(siteID: string): Promise<AgentSurfaceResponse> {
	return apiGet<AgentSurfaceResponse>(`/sites/${siteID}/agent-surface`);
}
