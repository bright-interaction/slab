import { apiGet, apiPost, apiPatch, apiDelete } from './client';
import type { Timestamp, BoolInt } from './types';

export type ProductStatus = 'draft' | 'active' | 'archived';
export type InventoryReason =
	| 'restock'
	| 'sale'
	| 'manual'
	| 'damage'
	| 'return'
	| 'correction';

export interface Product {
	id: string;
	site_id: string;
	name: string;
	slug: string;
	description: string;
	category: string;
	status: ProductStatus;
	images_json: string;
	base_price_cents: number;
	currency: string;
	requires_shipping: BoolInt;
	tax_class: string;
	weight_grams: number;
	sort_order: number;
	created_at: Timestamp;
	updated_at: Timestamp;
}

export interface ProductVariant {
	id: string;
	product_id: string;
	sku: string;
	name: string;
	price_cents: number;
	compare_at_price_cents: number;
	inventory_count: number;
	allow_backorder: BoolInt;
	sort_order: number;
	attributes_json: string;
	created_at: Timestamp;
	updated_at: Timestamp;
}

export interface InventoryAdjustment {
	id: string;
	variant_id: string;
	site_id: string;
	delta: number;
	new_count: number;
	reason: InventoryReason;
	note: string;
	created_by: string;
	created_at: Timestamp;
}

export interface ProductInput {
	name: string;
	slug: string;
	description?: string;
	category?: string;
	status?: ProductStatus;
	images?: string[];
	base_price_cents?: number;
	currency?: string;
	requires_shipping?: boolean;
	tax_class?: string;
	weight_grams?: number;
	sort_order?: number;
}

export interface VariantInput {
	sku?: string;
	name?: string;
	price_cents?: number;
	compare_at_price_cents?: number;
	inventory_count?: number;
	allow_backorder?: boolean;
	sort_order?: number;
	attributes?: Record<string, unknown>;
}

export async function listProducts(siteID: string): Promise<Product[]> {
	const resp = await apiGet<{ products: Product[]; count: number }>(
		`/sites/${siteID}/products`
	);
	return resp.products ?? [];
}

export function getProduct(siteID: string, id: string): Promise<Product> {
	return apiGet<Product>(`/sites/${siteID}/products/${id}`);
}

export function createProduct(siteID: string, body: ProductInput): Promise<Product> {
	return apiPost<Product>(`/sites/${siteID}/products`, body);
}

export function updateProduct(
	siteID: string,
	id: string,
	body: Partial<ProductInput>
): Promise<Product> {
	return apiPatch<Product>(`/sites/${siteID}/products/${id}`, body);
}

export function deleteProduct(siteID: string, id: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/products/${id}`);
}

export async function listVariants(siteID: string, productID: string): Promise<ProductVariant[]> {
	const resp = await apiGet<{ variants: ProductVariant[]; count: number }>(
		`/sites/${siteID}/products/${productID}/variants`
	);
	return resp.variants ?? [];
}

export function createVariant(
	siteID: string,
	productID: string,
	body: VariantInput
): Promise<ProductVariant> {
	return apiPost<ProductVariant>(`/sites/${siteID}/products/${productID}/variants`, body);
}

export function updateVariant(
	siteID: string,
	variantID: string,
	body: Partial<VariantInput>
): Promise<ProductVariant> {
	return apiPatch<ProductVariant>(`/sites/${siteID}/variants/${variantID}`, body);
}

export function deleteVariant(siteID: string, variantID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/sites/${siteID}/variants/${variantID}`);
}

export function setInventory(
	siteID: string,
	variantID: string,
	body: { new_count?: number; delta?: number; reason?: InventoryReason; note?: string }
): Promise<{ variant: ProductVariant; delta: number; reason: string }> {
	return apiPost(`/sites/${siteID}/variants/${variantID}/inventory`, body);
}

export async function listInventoryAdjustments(
	siteID: string,
	variantID: string
): Promise<InventoryAdjustment[]> {
	const resp = await apiGet<{ adjustments: InventoryAdjustment[]; count: number }>(
		`/sites/${siteID}/variants/${variantID}/inventory`
	);
	return resp.adjustments ?? [];
}
