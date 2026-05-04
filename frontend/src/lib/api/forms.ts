import { apiDelete, apiGet } from './client';

export interface FormSummary {
	id: string;
	name: string;
	action: string;
	action_config: string;
	fields_json: string;
	created_at: string;
	updated_at: string;
	submission_count: number;
}

export interface FormSubmission {
	id: string;
	form_id: string;
	site_id: string;
	data_json: string;
	ip_hash: string;
	created_at: string;
}

export function listForms(siteID: string): Promise<FormSummary[]> {
	return apiGet<FormSummary[]>(`/sites/${siteID}/forms`);
}

export function listSubmissions(siteID: string, formID: string): Promise<FormSubmission[]> {
	return apiGet<FormSubmission[]>(`/sites/${siteID}/forms/${formID}/submissions`);
}

export function deleteSubmission(siteID: string, formID: string, submissionID: string): Promise<unknown> {
	return apiDelete(`/sites/${siteID}/forms/${formID}/submissions/${submissionID}`);
}

export function csvExportURL(siteID: string, formID: string): string {
	return `/api/sites/${siteID}/forms/${formID}/submissions.csv`;
}
