export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'slab_theme';

function readInitial(): Theme {
	if (typeof window === 'undefined') return 'light';
	try {
		const stored = window.localStorage.getItem(STORAGE_KEY);
		if (stored === 'dark' || stored === 'light') return stored;
	} catch {
		// localStorage may be blocked, fall through.
	}
	return 'light';
}

let current = $state<Theme>(readInitial());

export const theme = {
	get value(): Theme {
		return current;
	}
};

export function setTheme(next: Theme): void {
	current = next;
	if (typeof document !== 'undefined') {
		document.documentElement.setAttribute('data-theme', next);
	}
	if (typeof window !== 'undefined') {
		try {
			window.localStorage.setItem(STORAGE_KEY, next);
		} catch {
			// Ignore quota or privacy-mode failures.
		}
	}
}

export function toggleTheme(): void {
	setTheme(current === 'dark' ? 'light' : 'dark');
}
