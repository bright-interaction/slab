import { apiGet, apiPost, apiPatch, apiDelete } from './client';
import type { Timestamp, BoolInt } from './types';

export type DiscountKind = 'percent' | 'fixed';
export type DiscountAppliesTo = 'all' | 'categories' | 'products';

export interface DiscountCode {
	id: string;
	site_id: string;
	code: string;
	kind: DiscountKind;
	value: number;
	min_subtotal_cents: number;
	max_uses: number;
	used_count: number;
	starts_at: string;
	ends_at: string;
	applies_to: DiscountAppliesTo;
	applies_to_ids_json: string;
	is_active: BoolInt;
	created_at: Timestamp;
	updated_at: Timestamp;
}

export interface DiscountCodeInput {
	code: string;
	kind: DiscountKind;
	value: number;
	min_subtotal_cents?: number;
	max_uses?: number;
	starts_at?: string;
	ends_at?: string;
	applies_to?: DiscountAppliesTo;
	applies_to_ids?: string[];
	is_active?: boolean;
}

export async function listDiscountCodes(siteID: string): Promise<DiscountCode[]> {
	const resp = await apiGet<{ discount_codes: DiscountCode[]; count: number }>(
		`/sites/${siteID}/discount-codes`
	);
	return resp.discount_codes ?? [];
}

export function createDiscountCode(
	siteID: string,
	body: DiscountCodeInput
): Promise<DiscountCode> {
	return apiPost<DiscountCode>(`/sites/${siteID}/discount-codes`, body);
}

export function updateDiscountCode(
	siteID: string,
	id: string,
	body: Partial<DiscountCodeInput>
): Promise<DiscountCode> {
	return apiPatch<DiscountCode>(`/sites/${siteID}/discount-codes/${id}`, body);
}

export function deleteDiscountCode(siteID: string, id: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/discount-codes/${id}`);
}
