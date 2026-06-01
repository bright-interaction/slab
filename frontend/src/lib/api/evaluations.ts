import { apiGet } from './client';
import type { Evaluation, EvaluationCheck } from './types';

export function listBySite(siteID: string, limit?: number): Promise<Evaluation[]> {
	const qs = limit !== undefined ? `?limit=${limit}` : '';
	return apiGet<Evaluation[]>(`/sites/${siteID}/evaluations${qs}`);
}

export function listByBuild(siteID: string, buildID: string): Promise<Evaluation[]> {
	return apiGet<Evaluation[]>(`/sites/${siteID}/evaluations/${buildID}`);
}

// The Go eval package serializes checks with field names that differ
// slightly from the frontend EvaluationCheck shape: Go emits `detail` and
// `recommendation`, the frontend renders `details` and `message`. Map at
// parse time so every CheckRow shows the fix hint instead of just a badge.
// A check whose `passed` is null in Go (informational only) is normalized
// to false here so the frontend's boolean-typed field stays sound, with
// severity "info" carrying the meaning.
interface RawCheck {
	name?: string;
	passed?: boolean | null;
	severity?: 'info' | 'warning' | 'error' | 'critical';
	detail?: string;
	recommendation?: string;
	page?: string;
	id?: string;
	score?: number;
	max_score?: number;
	message?: string;
	details?: string;
	url?: string;
}

export function parseEvaluationChecks(checks_json: string): EvaluationCheck[] {
	if (!checks_json) return [];
	try {
		const parsed = JSON.parse(checks_json);
		if (!Array.isArray(parsed)) return [];
		return (parsed as RawCheck[]).map((c) => ({
			id: c.id,
			name: c.name ?? '',
			passed: c.passed === undefined ? null : c.passed,
			score: c.score,
			max_score: c.max_score,
			severity: c.passed === null || c.passed === undefined ? 'info' : c.severity,
			message: c.message ?? c.recommendation,
			details: c.details ?? c.detail,
			url: c.url ?? c.page
		}));
	} catch {
		return [];
	}
}
