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
	target_id?: string;
	deploy_url?: string;
	deployed_at?: string;
}

export function triggerBuild(siteID: string): Promise<TriggerBuildResponse> {
	return apiPost<TriggerBuildResponse>(`/sites/${siteID}/build`);
}

export function getBuildStatus(siteID: string, buildID: string): Promise<BuildStatusResponse> {
	return apiGet<BuildStatusResponse>(`/sites/${siteID}/builds/${buildID}/status`);
}
