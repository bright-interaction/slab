import { apiGet, apiPost } from './client';
import type { AuthUser } from './types';

export interface AuthResponse {
	user: AuthUser;
}

export interface StatusResponse {
	status: string;
}

export function login(email: string, password: string): Promise<AuthResponse> {
	return apiPost<AuthResponse>('/auth/login', { email, password });
}

export function logout(): Promise<StatusResponse> {
	return apiPost<StatusResponse>('/auth/logout');
}

export function me(): Promise<AuthResponse> {
	return apiGet<AuthResponse>('/auth/me');
}

export function changePassword(currentPassword: string, newPassword: string): Promise<StatusResponse> {
	return apiPost<StatusResponse>('/auth/change-password', {
		current_password: currentPassword,
		new_password: newPassword
	});
}
