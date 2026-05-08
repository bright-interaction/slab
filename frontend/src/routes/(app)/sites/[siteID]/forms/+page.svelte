<script lang="ts">
	import { ArrowLeft, FormInput, ExternalLink } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { ApiError } from '$lib/api/client';
	import * as formsApi from '$lib/api/forms';
	import type { FormSummary } from '$lib/api/forms';

	let { data }: { data: { site: { id: string; name: string } } } = $props();
	const siteID = $derived(data.site.id);

	let forms = $state<FormSummary[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	async function load() {
		loading = true;
		loadError = null;
		try {
			forms = await formsApi.listForms(siteID);
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load forms.';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	function fmt(ts: string): string {
		if (!ts) return '-';
		const d = new Date(ts.replace(' ', 'T') + 'Z');
		return Number.isNaN(d.getTime()) ? ts : d.toLocaleString();
	}
</script>

<div class="mx-auto max-w-6xl px-6 py-8">
	<a
		href={`/sites/${siteID}`}
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Site dashboard
	</a>
	<header class="mt-3 flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Forms</h1>
		<p class="text-[13px] text-text-secondary">
			Public submissions captured by form blocks on your built site.
		</p>
	</header>

	<div class="mt-8 flex flex-col gap-4">
		{#if loading}
			<p class="text-[12px] text-text-muted">Loading...</p>
		{:else if loadError}
			<Card padding="md" class="border-danger/30">
				<p class="text-[13px] text-danger">{loadError}</p>
			</Card>
		{:else if forms.length === 0}
			<EmptyState
				title="No forms yet"
				description="Forms appear here once you add a form block to a page or create one via the agent API."
			>
				{#snippet icon()}
					<FormInput size={20} strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{:else}
			{#each forms as form (form.id)}
				<Card padding="md">
					<div class="flex items-start justify-between gap-4">
						<div class="min-w-0 flex-1">
							<h2 class="text-[15px] font-medium text-text-primary">{form.name}</h2>
							<p class="mt-1 text-[12px] text-text-muted">
								<span class="font-mono">{form.id}</span>
								{#if form.action !== 'store'}
									<span class="ml-2 rounded bg-bg-elevated px-1.5 py-0.5 text-[10px] uppercase tracking-wider">action: {form.action}</span>
								{/if}
							</p>
							<p class="mt-2 text-[13px] text-text-primary">
								<strong class="font-medium">{form.submission_count}</strong>
								<span class="text-text-secondary">submissions</span>
								<span class="ml-2 text-[11px] text-text-muted">updated {fmt(form.updated_at)}</span>
							</p>
						</div>
						<Button variant="secondary" size="sm" onclick={() => (window.location.href = `/sites/${siteID}/forms/${form.id}/submissions`)}>
							View submissions <ExternalLink size={12} strokeWidth={1.75} />
						</Button>
					</div>
				</Card>
			{/each}
		{/if}
	</div>
</div>
