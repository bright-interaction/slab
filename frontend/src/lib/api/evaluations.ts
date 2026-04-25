import { apiGet } from './client';
import type { Evaluation, EvaluationCheck } from './types';

export function listBySite(siteID: string, limit?: number): Promise<Evaluation[]> {
	const qs = limit !== undefined ? `?limit=${limit}` : '';
	return apiGet<Evaluation[]>(`/sites/${siteID}/evaluations${qs}`);
}

export function listByBuild(siteID: string, buildID: string): Promise<Evaluation[]> {
	return apiGet<Evaluation[]>(`/sites/${siteID}/evaluations/${buildID}`);
}

export function parseEvaluationChecks(checks_json: string): EvaluationCheck[] {
	if (!checks_json) return [];
	try {
		const parsed = JSON.parse(checks_json);
		return Array.isArray(parsed) ? (parsed as EvaluationCheck[]) : [];
	} catch {
		return [];
	}
}
