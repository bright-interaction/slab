<script lang="ts">
	import * as mediaApi from '$lib/api/media';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Medium, MediaVariant } from '$lib/api/types';

	let {
		open = $bindable(false),
		siteID,
		media,
		onUpdated,
		onDeleted
	}: {
		open?: boolean;
		siteID: string;
		media: Medium | null;
		onUpdated?: (m: Medium) => void;
		onDeleted?: (m: Medium) => void;
	} = $props();

	let altDraft = $state('');
	let saving = $state(false);
	let deleting = $state(false);

	$effect(() => {
		if (media) {
			altDraft = media.alt_text ?? '';
		}
	});

	const variants = $derived<MediaVariant[]>(
		media ? mediaApi.parseMediaVariants(media.variants_json) : []
	);

	const sortedVariants = $derived(
		[...variants].sort((a, b) => (a.width || 0) - (b.width || 0))
	);

	const previewSrc = $derived.by(() => {
		if (!media) return '';
		const widths = variants
			.filter((v) => v.format === 'webp')
			.map((v) => v.width)
			.sort((a, b) => b - a);
		if (widths.length > 0) {
			return mediaApi.mediaUrl(siteID, media.id, `${widths[0]}.webp`);
		}
		return mediaApi.mediaUrl(siteID, media.id, 'original');
	});

	const fullUrl = $derived.by(() => {
		if (!media) return '';
		if (typeof window === 'undefined') return mediaApi.mediaUrl(siteID, media.id, 'original');
		return `${window.location.origin}${mediaApi.mediaUrl(siteID, media.id, 'original')}`;
	});

	function formatBytes(n: number | undefined): string {
		if (!n) return '';
		if (n < 1024) return `${n} B`;
		if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
		return `${(n / (1024 * 1024)).toFixed(2)} MB`;
	}

	async function copy(text: string, label = 'URL') {
		try {
			await navigator.clipboard.writeText(text);
			toast.success(`${label} copied`);
		} catch {
			toast.error('Could not copy');
		}
	}

	async function saveAlt() {
		if (!media) return;
		if (altDraft === media.alt_text) return;
		saving = true;
		try {
			const updated = await mediaApi.update(siteID, media.id, { alt_text: altDraft });
			onUpdated?.(updated);
			toast.success('Alt text saved');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save');
			altDraft = media.alt_text ?? '';
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!media) return;
		const ok = await confirm({
			title: 'Delete media?',
			message: `${media.filename} will be removed permanently. This cannot be undone.`,
			confirmText: 'Delete',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		deleting = true;
		const target = media;
		try {
			await mediaApi.remove(siteID, target.id);
			onDeleted?.(target);
			open = false;
			toast.success('Media deleted');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete');
		} finally {
			deleting = false;
		}
	}

	function variantUrl(v: MediaVariant): string {
		if (!media) return '';
		const ext = v.format === 'webp' ? 'webp' : v.format;
		const variant = v.name === 'original' ? 'original' : `${v.name}.${ext}`;
		return mediaApi.mediaUrl(siteID, media.id, variant);
	}

	function fullVariantUrl(v: MediaVariant): string {
		const path = variantUrl(v);
		if (!path) return '';
		if (typeof window === 'undefined') return path;
		return `${window.location.origin}${path}`;
	}
</script>

<Dialog bind:open size="lg" title={media?.filename ?? 'Media'}>
	{#if media}
		<div class="flex flex-col gap-5">
			<div class="overflow-hidden rounded-xl border border-border-light bg-bg-elevated">
				<img
					src={previewSrc}
					alt={media.alt_text || media.filename}
					class="block max-h-[60vh] w-full object-contain"
				/>
			</div>

			<div class="grid grid-cols-2 gap-4 text-[12px]">
				<div>
					<p class="text-text-muted">Dimensions</p>
					<p class="text-text-primary">{media.width} x {media.height}</p>
				</div>
				<div>
					<p class="text-text-muted">Type</p>
					<p class="text-text-primary">{media.mime_type}</p>
				</div>
				<div>
					<p class="text-text-muted">Size</p>
					<p class="text-text-primary">{formatBytes(media.file_size)}</p>
				</div>
				<div>
					<p class="text-text-muted">Variants</p>
					<p class="text-text-primary">{variants.length}</p>
				</div>
			</div>

			<div>
				<Input
					label="Alt text"
					value={altDraft}
					oninput={(e) => (altDraft = (e.currentTarget as HTMLInputElement).value)}
					onblur={saveAlt}
					hint={saving ? 'Saving...' : 'Saves on blur'}
					placeholder="Describe this image"
				/>
			</div>

			<div>
				<div class="flex items-center justify-between">
					<h3 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Variants
					</h3>
					<button
						type="button"
						class="text-[12px] text-text-muted hover:text-text-primary transition-colors"
						onclick={() => copy(fullUrl, 'Original URL')}
					>
						Copy original URL
					</button>
				</div>
				<div class="mt-2 rounded-lg border border-border-light divide-y divide-border-light">
					{#each sortedVariants as v (v.name + v.format)}
						<div class="flex items-center justify-between gap-3 px-3 py-2 text-[12px]">
							<div class="flex items-center gap-3 min-w-0">
								<span class="font-mono text-text-primary">{v.name}</span>
								<span class="text-text-muted">{v.width} x {v.height}</span>
								<span class="text-text-muted uppercase">{v.format}</span>
								{#if v.size}
									<span class="text-text-muted">{formatBytes(v.size)}</span>
								{/if}
							</div>
							<button
								type="button"
								class="text-text-muted hover:text-text-primary transition-colors"
								onclick={() => copy(fullVariantUrl(v), `${v.name} URL`)}
							>
								Copy URL
							</button>
						</div>
					{:else}
						<div class="px-3 py-2 text-[12px] text-text-muted">No variants generated</div>
					{/each}
				</div>
			</div>

			<div class="flex items-center justify-between border-t border-border-light pt-4">
				<Button variant="danger" loading={deleting} onclick={handleDelete}>
					Delete
				</Button>
				<Button variant="secondary" onclick={() => (open = false)}>Close</Button>
			</div>
		</div>
	{/if}
</Dialog>
