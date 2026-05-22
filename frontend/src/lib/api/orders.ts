import { apiGet, apiPost } from './client';
import type { Timestamp } from './types';

export type OrderStatus =
	| 'pending'
	| 'paid'
	| 'fulfilled'
	| 'cancelled'
	| 'refunded'
	| 'failed';

export interface Order {
	id: string;
	site_id: string;
	order_number: string;
	status: OrderStatus;
	customer_name: string;
	customer_email: string;
	customer_phone: string;
	shipping_address_json: string;
	billing_address_json: string;
	subtotal_cents: number;
	discount_cents: number;
	shipping_cents: number;
	tax_cents: number;
	total_cents: number;
	currency: string;
	discount_code_id: string;
	payment_provider: string;
	payment_id: string;
	payment_status: string;
	payment_checkout_url: string;
	refund_id: string;
	notes: string;
	metadata_json: string;
	created_at: Timestamp;
	updated_at: Timestamp;
	paid_at: string;
	fulfilled_at: string;
	cancelled_at: string;
	refunded_at: string;
}

export interface OrderItem {
	id: string;
	order_id: string;
	variant_id: string;
	product_id: string;
	product_name: string;
	variant_name: string;
	sku: string;
	quantity: number;
	unit_price_cents: number;
	total_cents: number;
	created_at: Timestamp;
}

export interface OrderDetail {
	order: Order;
	items: OrderItem[];
}

export async function listOrders(siteID: string, status?: OrderStatus | 'all'): Promise<Order[]> {
	const qs = status && status !== 'all' ? `?status=${status}` : '';
	const resp = await apiGet<{ orders: Order[]; count: number }>(`/sites/${siteID}/orders${qs}`);
	return resp.orders ?? [];
}

export function getOrder(siteID: string, id: string): Promise<OrderDetail> {
	return apiGet<OrderDetail>(`/sites/${siteID}/orders/${id}`);
}

export function updateOrderStatus(
	siteID: string,
	id: string,
	body: { status: OrderStatus; notes?: string }
): Promise<OrderDetail> {
	return apiPost<OrderDetail>(`/sites/${siteID}/orders/${id}/status`, body);
}

export function refundOrder(siteID: string, id: string): Promise<OrderDetail> {
	return apiPost<OrderDetail>(`/sites/${siteID}/orders/${id}/refund`, {});
}
