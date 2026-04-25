import { apiDelete, apiGet, apiPatch, apiPost } from './client';
import type {
	DeployKind,
	DeployResult,
	DeployTarget,
	DeployTargetConfig
} from './types';

interface ListTargetsResponse {
	targets: DeployTarget[];
}

export interface CreateDeployTargetInput {
	name: string;
	kind: DeployKind;
	config: DeployTargetConfig;
	is_default?: boolean;
}

export interface UpdateDeployTargetPatch {
	name?: string;
	kind?: DeployKind;
	config?: DeployTargetConfig;
}

export interface TriggerDeployRequest {
	build_id: string;
	target_id: string;
}

export async function listTargets(siteID: string): Promise<DeployTarget[]> {
	const resp = await apiGet<ListTargetsResponse>(`/sites/${siteID}/deploy-targets`);
	return resp.targets ?? [];
}

export function createTarget(
	siteID: string,
	input: CreateDeployTargetInput
): Promise<DeployTarget> {
	return apiPost<DeployTarget>(`/sites/${siteID}/deploy-targets`, input);
}

export function getTarget(siteID: string, targetID: string): Promise<DeployTarget> {
	return apiGet<DeployTarget>(`/sites/${siteID}/deploy-targets/${targetID}`);
}

export function updateTarget(
	siteID: string,
	targetID: string,
	patch: UpdateDeployTargetPatch
): Promise<DeployTarget> {
	return apiPatch<DeployTarget>(`/sites/${siteID}/deploy-targets/${targetID}`, patch);
}

export async function removeTarget(siteID: string, targetID: string): Promise<void> {
	await apiDelete<{ status: string }>(`/sites/${siteID}/deploy-targets/${targetID}`);
}

export async function setDefault(siteID: string, targetID: string): Promise<void> {
	await apiPost<DeployTarget>(`/sites/${siteID}/deploy-targets/${targetID}/default`);
}

export function triggerDeploy(
	siteID: string,
	body: TriggerDeployRequest
): Promise<DeployResult> {
	return apiPost<DeployResult>(`/sites/${siteID}/deploy`, body);
}
