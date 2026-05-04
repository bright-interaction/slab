import { apiDelete, apiGet, apiPatch, apiPost } from './client';

export type FieldType =
	| 'text'
	| 'textarea'
	| 'richtext'
	| 'number'
	| 'boolean'
	| 'date'
	| 'datetime'
	| 'url'
	| 'email'
	| 'image'
	| 'gallery'
	| 'select'
	| 'multiselect'
	| 'reference'
	| 'color'
	| 'json';

export interface FieldDef {
	name: string;
	type: FieldType;
	label: string;
	required?: boolean;
	options?: string[];
	max_length?: number;
	ref_collection?: string;
}

export interface CollectionSettings {
	render_as_pages?: boolean;
	schema_org_type?: string;
	list_page_template?: string;
	detail_page_template?: string;
	items_per_index_page?: number;
	requires_member_role?: string;
	[k: string]: unknown;
}

export interface Collection {
	id: string;
	site_id: string;
	name: string;
	slug: string;
	schema: FieldDef[];
	settings: CollectionSettings;
	sort_order: number;
	item_count: number;
	created_at: string;
	updated_at: string;
}

export interface CollectionItem {
	id: string;
	collection_id: string;
	site_id: string;
	slug: string;
	title: string;
	data_json: string;
	locale: string;
	status: 'draft' | 'published' | 'scheduled';
	published_at: string;
	sort_order: number;
	created_at: string;
	updated_at: string;
}

export function listCollections(siteID: string): Promise<Collection[]> {
	return apiGet<Collection[]>(`/sites/${siteID}/collections`);
}

export function getCollection(siteID: string, collectionID: string): Promise<Collection> {
	return apiGet<Collection>(`/sites/${siteID}/collections/${collectionID}`);
}

export function createCollection(
	siteID: string,
	body: { name: string; slug: string; schema: FieldDef[]; settings?: CollectionSettings; sort_order?: number }
): Promise<Collection> {
	return apiPost<Collection>(`/sites/${siteID}/collections`, body);
}

export function updateCollection(
	siteID: string,
	collectionID: string,
	body: Partial<{ name: string; slug: string; schema: FieldDef[]; settings: CollectionSettings; sort_order: number }>
): Promise<Collection> {
	return apiPatch<Collection>(`/sites/${siteID}/collections/${collectionID}`, body);
}

export function deleteCollection(siteID: string, collectionID: string): Promise<unknown> {
	return apiDelete(`/sites/${siteID}/collections/${collectionID}`);
}

export function listItems(siteID: string, collectionID: string): Promise<CollectionItem[]> {
	return apiGet<CollectionItem[]>(`/sites/${siteID}/collections/${collectionID}/items`);
}

export function getItem(siteID: string, collectionID: string, itemID: string): Promise<CollectionItem> {
	return apiGet<CollectionItem>(`/sites/${siteID}/collections/${collectionID}/items/${itemID}`);
}

export interface ItemInput {
	slug: string;
	title: string;
	data?: Record<string, unknown>;
	locale?: string;
	status?: 'draft' | 'published' | 'scheduled';
	published_at?: string;
	sort_order?: number;
}

export function createItem(
	siteID: string,
	collectionID: string,
	body: ItemInput
): Promise<CollectionItem> {
	return apiPost<CollectionItem>(`/sites/${siteID}/collections/${collectionID}/items`, body);
}

export function updateItem(
	siteID: string,
	collectionID: string,
	itemID: string,
	body: Partial<ItemInput>
): Promise<CollectionItem> {
	return apiPatch<CollectionItem>(
		`/sites/${siteID}/collections/${collectionID}/items/${itemID}`,
		body
	);
}

export function deleteItem(siteID: string, collectionID: string, itemID: string): Promise<unknown> {
	return apiDelete(`/sites/${siteID}/collections/${collectionID}/items/${itemID}`);
}

export interface BulkImportResult {
	created: number;
	ids: string[];
}

export function bulkImport(
	siteID: string,
	collectionID: string,
	items: ItemInput[]
): Promise<BulkImportResult> {
	return apiPost<BulkImportResult>(
		`/sites/${siteID}/collections/${collectionID}/items/import`,
		{ items }
	);
}
