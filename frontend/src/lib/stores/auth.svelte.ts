import type { AuthUser } from '$lib/api/types';

let current = $state<AuthUser | null>(null);

export const auth = {
	get value(): AuthUser | null {
		return current;
	}
};

export function setUser(user: AuthUser | null): void {
	current = user;
}

export function clearUser(): void {
	current = null;
}
