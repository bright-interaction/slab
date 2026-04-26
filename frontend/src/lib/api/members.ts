import { apiDelete, apiGet, apiPatch, apiPost } from './client';

export interface Member {
	id: string;
	email: string;
	name: string;
	role: 'admin' | 'editor';
	created_at: string;
	is_self: boolean;
}

export interface Invite {
	id: string;
	email: string;
	role: 'admin' | 'editor';
	token: string;
	url: string;
	expires_at: string;
	created_at: string;
}

export function listMembers(): Promise<Member[]> {
	return apiGet<Member[]>('/admin/members');
}

export function updateMemberRole(userID: string, role: 'admin' | 'editor'): Promise<{ status: string }> {
	return apiPatch<{ status: string }>(`/admin/members/${userID}/role`, { role });
}

export function deleteMember(userID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/admin/members/${userID}`);
}

export function listInvites(): Promise<Invite[]> {
	return apiGet<Invite[]>('/admin/invites');
}

export function createInvite(email: string, role: 'admin' | 'editor'): Promise<Invite> {
	return apiPost<Invite>('/admin/invites', { email, role });
}

export function deleteInvite(inviteID: string): Promise<{ status: string }> {
	return apiDelete<{ status: string }>(`/admin/invites/${inviteID}`);
}
