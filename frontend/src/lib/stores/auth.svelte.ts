import type { AuthUser } from '$lib/api/types';

let current = $state<AuthUser | null>(null);
// MFA enrollment-required flag from the most recent /auth/login or
// /auth/me response. The dashboard renders a banner when this is on
// so the user can finish enrollment; the server-side
// EnforceMFAEnrollment middleware also blocks every write except the
// TOTP setup paths regardless of what the SPA shows.
let enrollRequired = $state<boolean>(false);

export const auth = {
	get value(): AuthUser | null {
		return current;
	},
	get enrollRequired(): boolean {
		return enrollRequired;
	}
};

export function setUser(user: AuthUser | null): void {
	current = user;
}

export function setEnrollRequired(v: boolean): void {
	enrollRequired = v;
}

export function clearUser(): void {
	current = null;
	enrollRequired = false;
}
