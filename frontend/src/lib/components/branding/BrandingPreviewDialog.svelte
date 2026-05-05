<script lang="ts">
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import BrandingPreview from './BrandingPreview.svelte';
	import { Smartphone, Monitor } from 'lucide-svelte';

	let {
		open = $bindable(false),
		primary,
		secondary,
		bg,
		text,
		surface,
		border,
		muted,
		accent,
		onPrimary,
		fontHeading,
		fontBody
	}: {
		open?: boolean;
		primary: string;
		secondary: string;
		bg: string;
		text: string;
		surface: string;
		border: string;
		muted: string;
		accent: string;
		onPrimary: string;
		fontHeading: string;
		fontBody: string;
	} = $props();

	// Mount on open, unmount on close so the preview only renders while
	// the dialog is on screen. Same pattern as BlockPreviewDialog.
	let mounted = $state(false);
	$effect(() => {
		if (open) mounted = true;
		else mounted = false;
	});

	let viewport = $state<'mobile' | 'desktop'>('desktop');
</script>

<Dialog
	bind:open
	title="Brand preview"
	description="Live render of headings, body, primary CTA, and surface colors. Mobile and desktop scaled into the same canvas."
	size="xl"
>
	<div class="flex items-center justify-end gap-3">
		<div
			role="group"
			aria-label="Preview viewport"
			class="inline-flex items-center gap-0.5 rounded-md border border-border-light bg-bg-elevated p-0.5"
		>
			<button
				type="button"
				aria-pressed={viewport === 'mobile'}
				aria-label="Mobile preview"
				onclick={() => (viewport = 'mobile')}
				class="inline-flex h-7 items-center gap-1.5 rounded px-2.5 text-[11.5px] font-medium transition-colors {viewport ===
				'mobile'
					? 'bg-bg-surface text-text-primary shadow-sm'
					: 'text-text-muted hover:text-text-primary'}"
			>
				<Smartphone size={13} strokeWidth={1.75} />
				Mobile
			</button>
			<button
				type="button"
				aria-pressed={viewport === 'desktop'}
				aria-label="Desktop preview"
				onclick={() => (viewport = 'desktop')}
				class="inline-flex h-7 items-center gap-1.5 rounded px-2.5 text-[11.5px] font-medium transition-colors {viewport ===
				'desktop'
					? 'bg-bg-surface text-text-primary shadow-sm'
					: 'text-text-muted hover:text-text-primary'}"
			>
				<Monitor size={13} strokeWidth={1.75} />
				Desktop
			</button>
		</div>
	</div>

	<div class="mt-4 overflow-hidden rounded-lg border border-border-light bg-white">
		{#if mounted}
			<BrandingPreview
				{primary}
				{secondary}
				{bg}
				{text}
				{surface}
				{border}
				{muted}
				{accent}
				{onPrimary}
				{fontHeading}
				{fontBody}
				{viewport}
			/>
		{/if}
	</div>
</Dialog>
