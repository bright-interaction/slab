// Self-hosted woff2 fonts. Backend lives in internal/handlers/fonts.go.
// Endpoints under /api/sites/{siteID}/fonts.

import { apiGet, apiDelete } from './client';

export interface SiteFont {
	id: string;
	site_id: string;
	family_name: string;
	weight: number;
	style: string;
	file_path: string;
	file_size: number;
	original_name: string;
	created_at: string;
}

export interface ListFontsResponse {
	fonts: SiteFont[];
}

export function list(siteID: string): Promise<ListFontsResponse> {
	return apiGet<ListFontsResponse>(`/sites/${siteID}/fonts`);
}

export function remove(siteID: string, fontID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/fonts/${fontID}`);
}

// Upload bypasses the JSON apiPost helper because it needs multipart form
// encoding. Keeps the same credentials posture as the rest of the SPA
// (cookie-based JWT).
export async function upload(
	siteID: string,
	file: File,
	familyName: string,
	weight: number,
	style: 'normal' | 'italic'
): Promise<SiteFont> {
	const fd = new FormData();
	fd.append('file', file);
	fd.append('family_name', familyName);
	fd.append('weight', String(weight));
	fd.append('style', style);
	const res = await fetch(`/api/sites/${siteID}/fonts`, {
		method: 'POST',
		credentials: 'same-origin',
		body: fd
	});
	if (!res.ok) {
		const text = await res.text();
		let detail = text;
		try {
			const j = JSON.parse(text);
			if (j.error) detail = j.error;
		} catch {
			// not JSON; keep raw text
		}
		throw new Error(detail || `Upload failed (${res.status})`);
	}
	return res.json();
}
