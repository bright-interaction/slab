import { apiGet, apiPatch, apiPost } from './client';
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

export function updateProfile(name: string): Promise<AuthResponse> {
	return apiPatch<AuthResponse>('/auth/me', { name });
}

export function signOutEverywhere(): Promise<StatusResponse> {
	return apiPost<StatusResponse>('/auth/sign-out-everywhere');
}

export interface InviteInfo {
	email: string;
	role: string;
}

export function inviteInfo(token: string): Promise<InviteInfo> {
	return apiGet<InviteInfo>(`/auth/signup/${token}`);
}

export function redeemInvite(token: string, name: string, password: string): Promise<AuthResponse> {
	return apiPost<AuthResponse>(`/auth/signup/${token}`, { name, password });
}
