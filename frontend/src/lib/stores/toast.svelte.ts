export type ToastKind = 'success' | 'error' | 'info';

export interface Toast {
	id: string;
	kind: ToastKind;
	message: string;
}

export interface ToastOptions {
	duration?: number;
}

const DEFAULT_DURATION = 4000;

let items = $state<Toast[]>([]);

export const toasts = {
	get value(): Toast[] {
		return items;
	}
};

function newId(): string {
	return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

function push(kind: ToastKind, message: string, opts?: ToastOptions): string {
	const id = newId();
	items = [...items, { id, kind, message }];
	const duration = opts?.duration ?? DEFAULT_DURATION;
	if (typeof window !== 'undefined' && duration > 0) {
		window.setTimeout(() => dismiss(id), duration);
	}
	return id;
}

export function dismiss(id: string): void {
	items = items.filter((t) => t.id !== id);
}

export const toast = {
	success: (message: string, opts?: ToastOptions) => push('success', message, opts),
	error: (message: string, opts?: ToastOptions) => push('error', message, opts),
	info: (message: string, opts?: ToastOptions) => push('info', message, opts)
};
