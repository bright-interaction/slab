import { apiGet } from './client';

export type StarterKit = {
	id: string;
	name: string;
	description: string;
	target_site_types: string[];
};

export function list(): Promise<StarterKit[]> {
	return apiGet<StarterKit[]>('/starter-kits');
}
