<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import GradeBadge from '$lib/components/ui/GradeBadge.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import ThemeToggle from '$lib/components/ui/ThemeToggle.svelte';
	import { toast } from '$lib/stores/toast.svelte';

	let name = $state('');
</script>

<div class="min-h-screen px-8 py-12">
	<div class="mx-auto max-w-3xl space-y-10">
		<header class="flex items-start justify-between">
			<div>
				<p class="font-mono text-[11px] uppercase tracking-[0.2em] text-text-muted">atomicsite</p>
				<h1 class="mt-2 font-display text-5xl font-extralight tracking-tight">UI primitives.</h1>
			</div>
			<ThemeToggle />
		</header>

		<section class="flex flex-wrap items-center gap-3">
			<Button variant="primary">Primary</Button>
			<Button variant="secondary">Secondary</Button>
			<Button variant="ghost">Ghost</Button>
			<Button variant="danger">Danger</Button>
		</section>

		<section class="grid grid-cols-2 gap-6">
			<Input label="Site name" placeholder="bright.studio" bind:value={name} hint="What you'll publish at." />
			<Card>
				<h3 class="text-sm font-medium text-text-primary">Card title</h3>
				<p class="mt-1 text-[13px] text-text-secondary">Surface block with a thin border.</p>
			</Card>
		</section>

		<section class="space-y-2">
			<Skeleton width="60%" height="1rem" />
			<Skeleton width="40%" height="0.75rem" />
		</section>

		<section class="flex items-end gap-4">
			<GradeBadge grade="A+" size="md" />
			<GradeBadge grade="B" size="md" />
			<GradeBadge grade="F" size="md" />
		</section>

		<section class="flex items-center gap-6 text-[13px]">
			<span class="inline-flex items-center gap-2"><span class="status-dot-success"></span>Online</span>
			<span class="inline-flex items-center gap-2"><span class="status-dot-warning"></span>Pending</span>
			<span class="inline-flex items-center gap-2"><span class="status-dot-danger"></span>Offline</span>
			<Spinner />
		</section>

		<section>
			<Button onclick={() => toast.success('Saved successfully')}>Trigger toast</Button>
		</section>
	</div>
</div>
