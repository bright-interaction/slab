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

// Forgot-password flow. Always returns 200 (the bare ack defeats
// account enumeration). Operator log surfaces the link when no
// MailSender is wired.
export interface ForgotPasswordResponse {
	status: string;
	message?: string;
}

export function forgotPassword(email: string): Promise<ForgotPasswordResponse> {
	return apiPost<ForgotPasswordResponse>('/auth/forgot-password', { email });
}

export interface ResetTokenInfo {
	email: string; // masked, e.g. a***@example.com
}

export function resetPasswordInfo(token: string): Promise<ResetTokenInfo> {
	return apiGet<ResetTokenInfo>(`/auth/reset-password/${token}`);
}

export function resetPassword(token: string, password: string): Promise<StatusResponse> {
	return apiPost<StatusResponse>(`/auth/reset-password/${token}`, { password });
}
