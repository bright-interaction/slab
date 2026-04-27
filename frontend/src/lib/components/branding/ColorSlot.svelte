<script lang="ts">
	const HEX_RE = /^#[0-9a-fA-F]{6}$/;

	let {
		label,
		helper,
		value = $bindable(),
		onReset
	}: {
		label: string;
		helper?: string;
		value: string;
		onReset?: () => void;
	} = $props();

	function normalize(v: string): string {
		const trimmed = v.trim();
		const withHash = trimmed.startsWith('#') ? trimmed : `#${trimmed}`;
		return HEX_RE.test(withHash) ? withHash.toLowerCase() : withHash;
	}

	function onPicker(e: Event): void {
		const v = (e.currentTarget as HTMLInputElement).value;
		value = v.toLowerCase();
	}

	function onText(e: Event): void {
		const v = (e.currentTarget as HTMLInputElement).value;
		const n = normalize(v);
		if (HEX_RE.test(n)) value = n;
		else value = v;
	}

	const isValid = $derived(HEX_RE.test(value));
</script>

<div class="flex flex-col gap-1.5">
	<div class="flex items-baseline justify-between">
		<span class="text-[12px] font-medium text-text-primary">{label}</span>
		{#if onReset}
			<button
				type="button"
				class="text-[11px] text-text-muted hover:text-text-primary transition-colors"
				onclick={onReset}
			>
				Reset
			</button>
		{/if}
	</div>
	<p class="min-h-[2lh] text-[11px] text-text-muted">{helper ?? ''}</p>
	<div class="flex items-center gap-2">
		<label
			class="relative inline-flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-md ring-1 ring-border overflow-hidden"
			style:background-color={isValid ? value : 'transparent'}
		>
			<input
				type="color"
				class="absolute inset-0 h-full w-full cursor-pointer opacity-0"
				value={isValid ? value : '#000000'}
				oninput={onPicker}
				aria-label="{label} color picker"
			/>
		</label>
		<input
			type="text"
			class="h-9 flex-1 rounded-lg border bg-bg-elevated px-3 text-[13px] font-mono text-text-primary placeholder:text-text-muted focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent {isValid
				? 'border-border'
				: 'border-danger/50'}"
			{value}
			oninput={onText}
			spellcheck="false"
			aria-label="{label} hex value"
		/>
	</div>
</div>
