export interface ApiErrorBody {
	error?: string;
	message?: string;
	fields?: Record<string, string>;
}

export class ApiError extends Error {
	status: number;
	fields?: Record<string, string>;

	constructor(status: number, message: string, fields?: Record<string, string>) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.fields = fields;
	}
}

const BASE = '/api';

async function parseBody<T>(res: Response): Promise<T> {
	const text = await res.text();
	if (!text) return undefined as unknown as T;
	try {
		return JSON.parse(text) as T;
	} catch {
		return text as unknown as T;
	}
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
	const url = path.startsWith('http') ? path : `${BASE}${path.startsWith('/') ? path : `/${path}`}`;

	const headers = new Headers(init.headers || {});
	if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}
	headers.set('Accept', 'application/json');

	const res = await fetch(url, {
		...init,
		credentials: 'include',
		headers
	});

	if (res.status === 401) {
		if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
			window.location.assign('/login');
		}
		throw new ApiError(401, 'Not authenticated');
	}

	if (!res.ok) {
		const body = await parseBody<ApiErrorBody>(res).catch(() => ({}) as ApiErrorBody);
		const message = body?.error || body?.message || `Request failed with status ${res.status}`;
		throw new ApiError(res.status, message, body?.fields);
	}

	if (res.status === 204) {
		return undefined as unknown as T;
	}

	return parseBody<T>(res);
}

export function apiGet<T>(path: string, init?: RequestInit): Promise<T> {
	return apiFetch<T>(path, { ...init, method: 'GET' });
}

export function apiPost<T>(path: string, body?: unknown, init?: RequestInit): Promise<T> {
	return apiFetch<T>(path, {
		...init,
		method: 'POST',
		body: body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined
	});
}

export function apiPatch<T>(path: string, body?: unknown, init?: RequestInit): Promise<T> {
	return apiFetch<T>(path, {
		...init,
		method: 'PATCH',
		body: body !== undefined ? JSON.stringify(body) : undefined
	});
}

export function apiPut<T>(path: string, body?: unknown, init?: RequestInit): Promise<T> {
	return apiFetch<T>(path, {
		...init,
		method: 'PUT',
		body: body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined
	});
}

export function apiDelete<T>(path: string, init?: RequestInit): Promise<T> {
	return apiFetch<T>(path, { ...init, method: 'DELETE' });
}
