<script lang="ts">
	import * as mediaApi from '$lib/api/media';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Medium } from '$lib/api/types';

	const MAX_BYTES = 20 * 1024 * 1024;

	let {
		siteID,
		folder,
		onUploaded
	}: {
		siteID: string;
		folder?: string;
		onUploaded?: (media: Medium) => void;
	} = $props();

	let inputEl: HTMLInputElement | null = $state(null);
	let uploading = $state(false);

	export function open(): void {
		inputEl?.click();
	}

	export function isUploading(): boolean {
		return uploading;
	}

	export async function uploadFiles(files: File[] | FileList): Promise<void> {
		const list = Array.from(files);
		const accepted: File[] = [];
		const rejected: { name: string; reason: string }[] = [];
		for (const f of list) {
			if (!f.type.startsWith('image/')) {
				rejected.push({ name: f.name, reason: 'not an image' });
				continue;
			}
			if (f.size > MAX_BYTES) {
				rejected.push({ name: f.name, reason: 'over 20 MB' });
				continue;
			}
			accepted.push(f);
		}
		for (const r of rejected) {
			toast.error(`Skipped ${r.name}: ${r.reason}`);
		}
		if (accepted.length === 0) return;

		uploading = true;
		let successes = 0;
		const tasks = accepted.map(async (file) => {
			try {
				const result = await mediaApi.upload(siteID, file, undefined, folder);
				return { ok: true as const, file, result };
			} catch (err) {
				const msg = err instanceof Error ? err.message : 'upload failed';
				return { ok: false as const, file, msg };
			}
		});
		const results = await Promise.all(tasks);
		for (const r of results) {
			if (r.ok) {
				successes += 1;
				onUploaded?.(r.result);
			} else {
				toast.error(`Failed ${r.file.name}: ${r.msg}`);
			}
		}
		if (successes > 0) {
			toast.success(`Uploaded ${successes} ${successes === 1 ? 'file' : 'files'}`);
		}
		uploading = false;
	}

	async function handleChange(event: Event): Promise<void> {
		const target = event.currentTarget as HTMLInputElement;
		if (!target.files || target.files.length === 0) return;
		await uploadFiles(target.files);
		target.value = '';
	}
</script>

<input
	bind:this={inputEl}
	type="file"
	multiple
	accept="image/*"
	class="hidden"
	onchange={handleChange}
	aria-hidden="true"
	tabindex="-1"
/>
