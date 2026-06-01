<script lang="ts">
	import { goto } from '$app/navigation';
	import { CheckCircle2, XCircle, AlertTriangle, Info } from 'lucide-svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { fixRoute } from '$lib/evaluations/fixRoutes';
	import type { EvaluationCheck } from '$lib/api/types';

	let {
		check,
		siteID
	}: {
		check: EvaluationCheck;
		siteID: string;
	} = $props();

	const checkKey = $derived(check.id ?? check.name ?? '');
	const target = $derived(fixRoute(checkKey, siteID));

	const severity = $derived(check.severity ?? (check.passed ? 'info' : 'warning'));

	const severityVariant = $derived.by<'success' | 'warning' | 'danger' | 'info' | 'default'>(() => {
		if (check.passed) return 'success';
		if (severity === 'critical' || severity === 'error') return 'danger';
		if (severity === 'warning') return 'warning';
		return 'info';
	});

	const severityLabel = $derived.by(() => {
		if (check.passed === true) return 'pass';
		if (severity === 'critical') return 'critical';
		if (severity === 'error') return 'high';
		if (severity === 'warning') return 'medium';
		if (severity === 'info') return 'info';
		return 'low';
	});
</script>

<div
	class="flex items-start gap-3 rounded-lg border border-border-light bg-bg p-3 transition-colors hover:bg-bg-hover"
>
	<div class="mt-0.5 shrink-0">
		{#if check.passed}
			<CheckCircle2 class="h-4 w-4 text-success" />
		{:else if severity === 'critical' || severity === 'error'}
			<XCircle class="h-4 w-4 text-danger" />
		{:else if severity === 'warning'}
			<AlertTriangle class="h-4 w-4 text-warning" />
		{:else}
			<Info class="h-4 w-4 text-info" />
		{/if}
	</div>

	<div class="flex-1 min-w-0">
		<div class="flex flex-wrap items-center gap-2">
			<p class="text-[13px] font-medium text-text-primary truncate">{check.name}</p>
			<Badge variant={severityVariant}>{severityLabel}</Badge>
		</div>
		{#if check.message}
			<p class="mt-1 text-[12px] text-text-secondary">{check.message}</p>
		{/if}
		{#if check.details}
			<p class="mt-1 text-[12px] text-text-muted">{check.details}</p>
		{/if}
		{#if check.url}
			<p class="mt-1 font-mono text-[11px] text-text-muted truncate">{check.url}</p>
		{/if}
	</div>

	{#if !check.passed && target}
		<div class="shrink-0">
			<Button variant="secondary" size="sm" onclick={() => goto(target)}>Fix</Button>
		</div>
	{/if}
</div>
