import { error } from '@sveltejs/kit';
import * as sitesApi from '$lib/api/sites';
import { ApiError } from '$lib/api/client';
import { setSite } from '$lib/stores/currentSite.svelte';
import type { LayoutLoad } from './$types';

export const ssr = false;

export const load: LayoutLoad = async ({ params }) => {
	try {
		const site = await sitesApi.get(params.siteID);
		setSite(site);
		return { site };
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) {
			throw error(404, 'Site not found');
		}
		throw err;
	}
};
