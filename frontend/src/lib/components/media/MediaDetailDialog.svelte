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

	// Sprint 5 quick-win (2026-05-24): per-image focal point. Click the
	// preview to set the focal point; the marker shows the current
	// position; save persists focal_x / focal_y as percentages.
	let focalX = $state(50);
	let focalY = $state(50);
	let focalSaving = $state(false);

	$effect(() => {
		if (media) {
			altDraft = media.alt_text ?? '';
			focalX = typeof media.focal_x === 'number' ? media.focal_x : 50;
			focalY = typeof media.focal_y === 'number' ? media.focal_y : 50;
		}
	});

	async function saveFocal(nextX: number, nextY: number) {
		if (!media) return;
		const clamp = (n: number) => Math.max(0, Math.min(100, Math.round(n)));
		const x = clamp(nextX);
		const y = clamp(nextY);
		if (x === media.focal_x && y === media.focal_y) return;
		focalSaving = true;
		try {
			const updated = await mediaApi.update(siteID, media.id, { focal_x: x, focal_y: y });
			focalX = updated.focal_x ?? x;
			focalY = updated.focal_y ?? y;
			onUpdated?.(updated);
			toast.success('Focal point saved');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save focal point');
		} finally {
			focalSaving = false;
		}
	}

	function handlePreviewClick(e: MouseEvent) {
		const target = e.currentTarget as HTMLDivElement;
		const rect = target.getBoundingClientRect();
		const x = ((e.clientX - rect.left) / rect.width) * 100;
		const y = ((e.clientY - rect.top) / rect.height) * 100;
		focalX = Math.round(x);
		focalY = Math.round(y);
		void saveFocal(x, y);
	}

	function resetFocal() {
		focalX = 50;
		focalY = 50;
		void saveFocal(50, 50);
	}

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
			<div
				role="button"
				tabindex="0"
				class="relative overflow-hidden rounded-xl border border-border-light bg-bg-elevated cursor-crosshair"
				onclick={handlePreviewClick}
				onkeydown={(e) => {
					if (e.key === 'Enter') resetFocal();
				}}
				title="Click to set focal point"
			>
				<img
					src={previewSrc}
					alt={media.alt_text || media.filename}
					class="block max-h-[60vh] w-full object-contain pointer-events-none"
				/>
				<span
					class="absolute -translate-x-1/2 -translate-y-1/2 h-6 w-6 rounded-full border-2 border-accent bg-accent/30 pointer-events-none transition-all"
					style:left={`${focalX}%`}
					style:top={`${focalY}%`}
				></span>
			</div>
			<div class="flex items-center justify-between text-[11px] text-text-muted">
				<span>Focal point: {focalX}% x {focalY}% {focalSaving ? '(saving...)' : ''}</span>
				<button
					type="button"
					class="text-accent hover:underline"
					onclick={resetFocal}
				>
					Reset to centered
				</button>
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
