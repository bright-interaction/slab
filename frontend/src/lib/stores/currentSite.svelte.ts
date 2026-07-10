import type { Site } from '$lib/api/types';

const STORAGE_KEY = 'slab_last_site_id';

let active = $state<Site | null>(null);

export const currentSite = {
	get value(): Site | null {
		return active;
	}
};

function persistID(id: string | null): void {
	if (typeof window === 'undefined') return;
	try {
		if (id) {
			window.localStorage.setItem(STORAGE_KEY, id);
		} else {
			window.localStorage.removeItem(STORAGE_KEY);
		}
	} catch {
		// localStorage may be blocked; ignore.
	}
}

export function setSite(site: Site): void {
	active = site;
	persistID(site.id);
}

export function clearSite(): void {
	active = null;
	persistID(null);
}

export function lastSiteID(): string | null {
	if (typeof window === 'undefined') return null;
	try {
		return window.localStorage.getItem(STORAGE_KEY);
	} catch {
		return null;
	}
}
