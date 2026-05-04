<script lang="ts">
	import { page } from '$app/stores';
	import { ArrowLeft, Trash2, Download } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { ApiError } from '$lib/api/client';
	import * as formsApi from '$lib/api/forms';
	import type { FormSubmission } from '$lib/api/forms';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { toast } from '$lib/stores/toast.svelte';

	let { data }: { data: { site: { id: string; name: string } } } = $props();
	const siteID = $derived(data.site.id);
	const formID = $derived($page.params.formID as string);

	let rows = $state<FormSubmission[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let expanded = $state<Record<string, boolean>>({});

	async function load() {
		loading = true;
		loadError = null;
		try {
			rows = await formsApi.listSubmissions(siteID, formID);
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load submissions.';
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

	function previewFields(dataJSON: string): string {
		try {
			const parsed = JSON.parse(dataJSON) as Record<string, unknown>;
			return Object.entries(parsed)
				.slice(0, 3)
				.map(([k, v]) => `${k}: ${String(v).slice(0, 40)}`)
				.join(' . ');
		} catch {
			return '(unparseable)';
		}
	}

	async function deleteRow(submissionID: string) {
		const ok = await confirm({
			title: 'Delete this submission?',
			message: 'This permanently removes the row. Cannot be undone.',
			confirmText: 'Delete',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await formsApi.deleteSubmission(siteID, formID, submissionID);
			rows = rows.filter((r) => r.id !== submissionID);
			toast.success('Submission deleted.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed.');
		}
	}

	function downloadCSV() {
		window.location.href = formsApi.csvExportURL(siteID, formID);
	}
</script>

<div class="mx-auto max-w-6xl px-6 py-8">
	<a
		href={`/sites/${siteID}/forms`}
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Forms
	</a>
	<header class="mt-3 flex items-start justify-between gap-4">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Submissions</h1>
			<p class="text-[13px] text-text-secondary">
				Form ID <code class="font-mono text-[12px]">{formID}</code>
			</p>
		</div>
		{#if rows.length > 0}
			<Button variant="secondary" onclick={downloadCSV}>
				<Download size={14} strokeWidth={1.75} />Export CSV
			</Button>
		{/if}
	</header>

	<div class="mt-8 flex flex-col gap-3">
		{#if loading}
			<p class="text-[12px] text-text-muted">Loading...</p>
		{:else if loadError}
			<Card padding="md" class="border-danger/30">
				<p class="text-[13px] text-danger">{loadError}</p>
			</Card>
		{:else if rows.length === 0}
			<EmptyState
				title="No submissions yet"
				description="Submissions appear here as visitors fill out the form on the built site."
			/>
		{:else}
			{#each rows as row (row.id)}
				<Card padding="md">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0 flex-1">
							<p class="text-[12px] text-text-muted">
								<span class="font-mono">{row.id}</span>
								<span class="mx-2">.</span>
								{fmt(row.created_at)}
								{#if row.ip_hash}
									<span class="mx-2">.</span>
									<span class="font-mono text-[10px]">ip:{row.ip_hash.slice(0, 8)}</span>
								{/if}
							</p>
							<p class="mt-1 text-[13px] text-text-primary">{previewFields(row.data_json)}</p>
							{#if expanded[row.id]}
								<pre class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-3 font-mono text-[11px] text-text-primary">{row.data_json}</pre>
							{/if}
						</div>
						<div class="flex flex-col items-end gap-2">
							<Button variant="secondary" size="sm" onclick={() => (expanded[row.id] = !expanded[row.id])}>
								{expanded[row.id] ? 'Hide' : 'View JSON'}
							</Button>
							<button
								type="button"
								class="inline-flex h-6 w-6 items-center justify-center rounded text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
								onclick={() => deleteRow(row.id)}
								aria-label="Delete submission"
							>
								<Trash2 size={12} strokeWidth={1.75} />
							</button>
						</div>
					</div>
				</Card>
			{/each}
		{/if}
	</div>
</div>
