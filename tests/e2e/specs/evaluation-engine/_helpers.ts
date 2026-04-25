/**
 * Shared helpers for evaluation-engine specs.
 *
 * GET /api/sites/{siteID}/evaluations/{buildID} returns one row per category
 * (security, seo, performance, accessibility, privacy). Each row's checks_json
 * is a stringified array of {name, section, passed, weight, severity, ...}.
 */
import { type APIRequestContext } from '@playwright/test';
import { u } from '../../fixtures/auth';

export type EvaluationRow = {
	id: string;
	build_id: string;
	site_id: string;
	category: string;
	score: number;
	max_score: number;
	grade: string;
	checks_json: string;
	created_at: string;
};

export type CheckResult = {
	name: string;
	section: string;
	passed: boolean | null;
	weight: number;
	severity?: string;
	detail?: string;
	recommendation?: string;
	page?: string;
};

export async function getEvaluations(
	api: APIRequestContext,
	siteID: string,
	buildID: string
): Promise<EvaluationRow[]> {
	const res = await api.get(u(`/api/sites/${siteID}/evaluations/${buildID}`));
	if (!res.ok()) {
		throw new Error(`getEvaluations failed: ${res.status()} ${await res.text()}`);
	}
	const body = await res.json();
	return (Array.isArray(body) ? body : (body.evaluations ?? body.rows ?? [])) as EvaluationRow[];
}

export function findCategory(rows: EvaluationRow[], category: string): EvaluationRow {
	const row = rows.find((r) => r.category === category);
	if (!row) {
		throw new Error(`category ${category} not found in evaluations: ${rows.map((r) => r.category).join(',')}`);
	}
	return row;
}

export function parseChecks(row: EvaluationRow): CheckResult[] {
	return JSON.parse(row.checks_json) as CheckResult[];
}

export function checkNames(row: EvaluationRow): string[] {
	return parseChecks(row).map((c) => c.name);
}

export const VALID_GRADES = new Set([
	'A+',
	'A',
	'A-',
	'B+',
	'B',
	'B-',
	'C+',
	'C',
	'C-',
	'D+',
	'D',
	'D-',
	'F',
	'N/A'
]);
