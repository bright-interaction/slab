<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import {
		ArrowLeft,
		Layers,
		LayoutGrid,
		Code2,
		Plug,
		Workflow
	} from 'lucide-svelte';
</script>

{#snippet sectionCard(
	href: string,
	title: string,
	subtitle: string,
	IconCmp: typeof Layers
)}
	<a {href} class="block">
		<Card padding="md" hoverable>
			<div class="flex items-start gap-3">
				<span
					class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-border-light bg-bg-elevated text-text-secondary"
				>
					<IconCmp size={16} strokeWidth={1.5} />
				</span>
				<div class="flex flex-col gap-0.5 min-w-0">
					<p class="text-[13px] font-medium text-text-primary">{title}</p>
					<p class="text-[12px] text-text-muted">{subtitle}</p>
				</div>
			</div>
		</Card>
	</a>
{/snippet}

<div class="mx-auto max-w-5xl px-6 py-8">
	<a
		href="/"
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Dashboard
	</a>
	<header class="mt-3 flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Documentation
		</h1>
		<p class="text-[13px] text-text-secondary">
			Atomic Site is a structured canvas any AI agent can paint on. These pages explain how the
			system works, what every feature does, and how to wire your own AI agent into a site.
		</p>
	</header>

	<section class="mt-8 grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
		{@render sectionCard(
			'/docs/architecture',
			'Architecture',
			'How the system works end to end.',
			Layers
		)}
		{@render sectionCard(
			'/docs/features',
			'Features',
			'Every admin feature, what it does, where to find it.',
			LayoutGrid
		)}
		{@render sectionCard(
			'/docs/agent-api',
			'Agent API',
			'Full endpoint catalogue for BYOAI integration.',
			Code2
		)}
		{@render sectionCard(
			'/docs/connect-your-agent',
			'Connect your agent',
			'Wire Claude CLI, Cursor, or any agent to a site.',
			Plug
		)}
		{@render sectionCard(
			'/docs/integrations',
			'Integrations',
			'External systems Atomic Site connects to.',
			Workflow
		)}
	</section>
</div>
