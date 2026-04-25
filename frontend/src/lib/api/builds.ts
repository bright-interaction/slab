import { apiGet, apiPost } from './client';

export interface TriggerBuildResponse {
	build_id: string;
	status: string;
}

export interface BuildStatusResponse {
	status: string;
	build_log: string;
	pages_built: number;
	duration_ms: number;
	error: string;
	dist_dir?: string;
}

export function triggerBuild(siteID: string): Promise<TriggerBuildResponse> {
	return apiPost<TriggerBuildResponse>(`/sites/${siteID}/build`);
}

// TODO(commit-12): admin build status endpoint not yet wired in server.go.
// Today only the agent route /api/agent/build/{buildID}/status exists, which
// requires X-Agent-Key auth. Commit 12 should add an admin variant.
export function getBuildStatus(siteID: string, buildID: string): Promise<BuildStatusResponse> {
	return apiGet<BuildStatusResponse>(`/sites/${siteID}/builds/${buildID}/status`);
}
