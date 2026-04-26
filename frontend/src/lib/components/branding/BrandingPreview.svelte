<script lang="ts">
	let {
		primary,
		secondary,
		bg,
		text,
		surface = '#ffffff',
		border = '',
		muted = '',
		accent = '',
		onPrimary = '#ffffff',
		fontHeading,
		fontBody
	}: {
		primary: string;
		secondary: string;
		bg: string;
		text: string;
		surface?: string;
		border?: string;
		muted?: string;
		accent?: string;
		onPrimary?: string;
		fontHeading: string;
		fontBody: string;
	} = $props();

	// Empty optional slots fall back to derived values so the preview keeps
	// rendering sensibly when a site only has the four core colours filled in.
	const effectiveBorder = $derived(
		border && /^#[0-9a-fA-F]{6}$/.test(border)
			? border
			: 'color-mix(in srgb, ' + text + ' 12%, transparent)'
	);
	const effectiveMuted = $derived(
		muted && /^#[0-9a-fA-F]{6}$/.test(muted)
			? muted
			: 'color-mix(in srgb, ' + text + ' 70%, transparent)'
	);
	const effectiveAccent = $derived(
		accent && /^#[0-9a-fA-F]{6}$/.test(accent) ? accent : primary
	);

	const style = $derived(
		[
			`--bp-primary:${primary}`,
			`--bp-secondary:${secondary}`,
			`--bp-bg:${bg}`,
			`--bp-text:${text}`,
			`--bp-surface:${surface}`,
			`--bp-border:${effectiveBorder}`,
			`--bp-muted:${effectiveMuted}`,
			`--bp-accent:${effectiveAccent}`,
			`--bp-on-primary:${onPrimary}`,
			`--bp-font-heading:'${fontHeading}'`,
			`--bp-font-body:'${fontBody}'`
		].join(';') + ';'
	);
</script>

<div
	class="overflow-hidden rounded-2xl border shadow-sm"
	style={style +
		"background:var(--bp-bg);color:var(--bp-text);border-color:var(--bp-border);font-feature-settings:'cv11','ss01';"}
>
	<div
		class="flex items-center justify-between px-6 py-4 border-b"
		style="border-color:var(--bp-border);"
	>
		<span
			class="text-[12px] font-semibold tracking-tight"
			style="font-family:var(--bp-font-heading);"
		>
			Acme
		</span>
		<nav
			class="hidden gap-5 sm:flex"
			style="font-family:var(--bp-font-body);font-size:12px;"
		>
			<span style="color:var(--bp-muted);">Product</span>
			<span style="color:var(--bp-muted);">Pricing</span>
			<span style="color:var(--bp-muted);">About</span>
		</nav>
	</div>

	<div class="px-8 pt-12 pb-10">
		<span
			class="text-[10px] font-medium uppercase tracking-[0.18em]"
			style="color:var(--bp-secondary);font-family:var(--bp-font-body);"
		>
			Atomic, by design
		</span>
		<h1
			class="mt-4 max-w-md"
			style="font-family:var(--bp-font-heading);font-weight:200;letter-spacing:-0.025em;font-size:34px;line-height:1.08;color:var(--bp-text);"
		>
			A calm site that earns the click.
		</h1>
		<p
			class="mt-4 max-w-md"
			style="font-family:var(--bp-font-body);font-size:13px;line-height:1.65;color:var(--bp-muted);"
		>
			Restrained type, accessible color, real content. Everything else is decoration.
		</p>
		<div class="mt-6">
			<button
				type="button"
				class="inline-flex h-10 items-center rounded-lg px-4 text-[13px] font-medium transition-opacity hover:opacity-90"
				style="background:var(--bp-primary);color:var(--bp-on-primary);font-family:var(--bp-font-body);"
			>
				Get started
			</button>
		</div>
	</div>

	<div
		class="grid grid-cols-1 gap-3 px-8 pb-10 sm:grid-cols-2"
		style="font-family:var(--bp-font-body);"
	>
		<div
			class="rounded-xl border p-5"
			style="border-color:var(--bp-border);background:var(--bp-surface);"
		>
			<h3
				style="font-family:var(--bp-font-heading);font-weight:300;font-size:15px;letter-spacing:-0.01em;color:var(--bp-text);"
			>
				Composable blocks
			</h3>
			<p
				class="mt-2"
				style="font-size:12px;line-height:1.6;color:var(--bp-muted);"
			>
				Sections snap together cleanly without breaking your grid.
			</p>
			<span
				class="mt-3 inline-block text-[12px] font-medium"
				style="color:var(--bp-accent);"
			>
				Learn more
			</span>
		</div>
		<div
			class="rounded-xl border p-5"
			style="border-color:var(--bp-border);background:var(--bp-surface);"
		>
			<h3
				style="font-family:var(--bp-font-heading);font-weight:300;font-size:15px;letter-spacing:-0.01em;color:var(--bp-text);"
			>
				Built for review
			</h3>
			<p
				class="mt-2"
				style="font-size:12px;line-height:1.6;color:var(--bp-muted);"
			>
				Every page passes WCAG, SEO, and performance gates before it ships.
			</p>
			<span
				class="mt-3 inline-block text-[12px] font-medium"
				style="color:var(--bp-accent);"
			>
				See checks
			</span>
		</div>
	</div>

	<div
		class="flex flex-wrap items-center gap-4 border-t px-8 py-4 text-[11px]"
		style="border-color:var(--bp-border);font-family:var(--bp-font-body);color:var(--bp-secondary);"
	>
		<span>Privacy</span>
		<span>Terms</span>
		<span>Status</span>
		<span>Contact</span>
	</div>
</div>
