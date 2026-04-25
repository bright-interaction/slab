<script lang="ts" module>
	export type BlockTypeOption = {
		type: string;
		label: string;
		description: string;
	};

	export const BLOCK_TYPE_OPTIONS: BlockTypeOption[] = [
		{ type: 'hero', label: 'Hero', description: 'Top-of-page banner with headline and CTA.' },
		{ type: 'text', label: 'Text', description: 'Rich paragraph or markdown body.' },
		{ type: 'image', label: 'Image', description: 'Single image with alt and caption.' },
		{ type: 'cta', label: 'Call to action', description: 'Headline, body, primary button.' },
		{
			type: 'feature_grid',
			label: 'Feature grid',
			description: 'Repeating cards with icon and copy.'
		}
	];
</script>

<script lang="ts">
	import { Popover } from 'bits-ui';
	import { Plus, Sparkles, Type, Image as ImageIcon, MousePointerClick, LayoutGrid } from 'lucide-svelte';

	let {
		onPick,
		disabled = false
	}: {
		onPick: (type: string) => void;
		disabled?: boolean;
	} = $props();

	let open = $state(false);

	function pick(type: string) {
		open = false;
		onPick(type);
	}

	function iconFor(type: string) {
		switch (type) {
			case 'hero':
				return Sparkles;
			case 'text':
				return Type;
			case 'image':
				return ImageIcon;
			case 'cta':
				return MousePointerClick;
			case 'feature_grid':
				return LayoutGrid;
			default:
				return Plus;
		}
	}
</script>

<Popover.Root bind:open>
	<Popover.Trigger
		{disabled}
		class="inline-flex h-9 items-center justify-center gap-2 rounded-lg border border-dashed border-border bg-bg-elevated px-4 text-[13px] text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg disabled:cursor-not-allowed disabled:opacity-50"
	>
		<Plus class="h-3.5 w-3.5" />
		Add block
	</Popover.Trigger>
	<Popover.Portal>
		<Popover.Content
			sideOffset={8}
			align="center"
			class="z-[110] w-[28rem] rounded-xl border border-border bg-bg-surface p-3 shadow-xl focus:outline-none data-[state=open]:animate-fadeIn"
		>
			<p class="px-1 pb-2 text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Choose block type
			</p>
			<div class="grid grid-cols-2 gap-2">
				{#each BLOCK_TYPE_OPTIONS as opt (opt.type)}
					{@const Icon = iconFor(opt.type)}
					<button
						type="button"
						class="flex items-start gap-3 rounded-lg border border-border-light bg-bg p-3 text-left transition-colors hover:bg-bg-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
						onclick={() => pick(opt.type)}
					>
						<span
							class="mt-0.5 inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-bg-elevated text-text-secondary"
						>
							<Icon class="h-3.5 w-3.5" />
						</span>
						<span class="flex min-w-0 flex-col gap-0.5">
							<span class="text-[13px] font-medium text-text-primary">{opt.label}</span>
							<span class="text-[11px] leading-snug text-text-muted">{opt.description}</span>
						</span>
					</button>
				{/each}
			</div>
		</Popover.Content>
	</Popover.Portal>
</Popover.Root>
