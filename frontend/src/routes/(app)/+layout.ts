import { redirect } from '@sveltejs/kit';
import * as authApi from '$lib/api/auth';
import { ApiError } from '$lib/api/client';
import { setUser, setEnrollRequired } from '$lib/stores/auth.svelte';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async () => {
	try {
		const res = await authApi.me();
		setUser(res.user);
		setEnrollRequired(Boolean(res.enroll_required));
		return { user: res.user, enrollRequired: Boolean(res.enroll_required) };
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) {
			throw redirect(303, '/login');
		}
		throw err;
	}
};
