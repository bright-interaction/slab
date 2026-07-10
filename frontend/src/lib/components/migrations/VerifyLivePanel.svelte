<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as migrations from '$lib/api/migrations';
	import { ApiError } from '$lib/api/client';
	import {
		pollVerifyJob,
		getVerifyJobState,
		clearVerifyJob,
		type VerifyJobView
	} from '$lib/stores/verifyJob.svelte';
	import type { VerifyJob, MigrationVerification } from '$lib/api/migrations';

	let { siteID, migrationID }: { siteID: string; migrationID: string } = $props();

	let urlsText = $state('');
	let deployedDomain = $state('');
	let activeJobID = $state<string | null>(null);
	let stopFn: (() => void) | null = null;
	let triggering = $state(false);
	let cancelling = $state(false);

	let history = $state<VerifyJob[]>([]);
	let historyLoading = $state(true);

	let verifications = $state<MigrationVerification[]>([]);
	let verificationsLoaded = $state(false);

	const liveState = $derived(activeJobID ? getVerifyJobState(activeJobID) : null);
	const status = $derived<VerifyJobView>(liveState?.status ?? 'queued');
	const isTerminal = $derived(
		status === 'done' || status === 'failed' || status === 'cancelled' || status === 'timeout'
	);

	const total = $derived(liveState?.total_urls ?? 0);
	const processed = $derived(liveState?.processed_urls ?? 0);
	const okCount = $derived(liveState?.ok_count ?? 0);
	const failCount = $derived(liveState?.fail_count ?? 0);
	const progressPct = $derived(total > 0 ? Math.round((processed / total) * 100) : 0);

	$effect(() => {
		// Load failures table when a job hits "done" so the operator
		// sees per-URL detail without an explicit click.
		if (status === 'done' && activeJobID && !verificationsLoaded) {
			void loadVerifications();
		}
	});

	onMount(() => {
		void loadHistory();
	});

	onDestroy(() => {
		stopFn?.();
		stopFn = null;
	});

	async function loadHistory() {
		historyLoading = true;
		try {
			const resp = await migrations.listVerifyJobs(siteID, migrationID);
			history = resp.items ?? [];
		} catch {
			history = [];
		} finally {
			historyLoading = false;
		}
	}

	async function loadVerifications() {
		try {
			const resp = await migrations.listVerifications(siteID, migrationID);
			verifications = (resp.items ?? []).filter((v) => !v.ok);
			verificationsLoaded = true;
		} catch {
			verifications = [];
		}
	}

	function parseURLs(): string[] {
		return urlsText
			.split(/\r?\n/)
			.map((s) => s.trim())
			.filter(Boolean);
	}

	async function trigger() {
		const urls = parseURLs();
		if (!deployedDomain.trim() || urls.length === 0) return;
		triggering = true;
		try {
			const resp = await migrations.verifyLive(siteID, migrationID, {
				deployed_domain: deployedDomain.trim(),
				urls
			});
			toast.success(`Verify job queued: ${resp.total_urls} URLs`);
			stopFn?.();
			activeJobID = resp.job_id;
			verifications = [];
			verificationsLoaded = false;
			stopFn = pollVerifyJob(siteID, migrationID, resp.job_id, {
				total_urls: resp.total_urls,
				deployed_domain: deployedDomain.trim()
			});
			await loadHistory();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Verify-live failed to start');
		} finally {
			triggering = false;
		}
	}

	async function cancel() {
		if (!activeJobID) return;
		cancelling = true;
		try {
			await migrations.cancelVerifyJob(siteID, migrationID, activeJobID);
			toast.info('Cancellation signal sent');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Cancel failed');
		} finally {
			cancelling = false;
		}
	}

	function reset() {
		if (activeJobID) {
			stopFn?.();
			clearVerifyJob(activeJobID);
		}
		activeJobID = null;
		verifications = [];
		verificationsLoaded = false;
	}

	function statusVariant(s: string): 'default' | 'success' | 'danger' | 'info' {
		if (s === 'done') return 'success';
		if (s === 'failed' || s === 'cancelled' || s === 'timeout') return 'danger';
		if (s === 'running' || s === 'queued') return 'info';
		return 'default';
	}
</script>

<Card>
	<div class="mb-4">
		<h2 class="text-sm font-medium text-text-primary">Post-launch verification</h2>
		<p class="text-xs text-text-muted mt-1">
			Crawl old URLs against the deployed slab. Each URL gets a fresh GET, redirects
			follow up to 3 hops, terminal status is recorded. Cap: 25,000 URLs / run.
		</p>
	</div>

	{#if !activeJobID}
		<div class="space-y-3">
			<div>
				<label for="deployed_domain" class="block text-xs text-text-muted mb-1.5">
					Deployed domain (where DNS now points)
				</label>
				<Input
					id="deployed_domain"
					bind:value={deployedDomain}
					placeholder="www.example.com or http://127.0.0.1:8080 (tests)"
				/>
			</div>
			<div>
				<label for="verify_urls" class="block text-xs text-text-muted mb-1.5">
					URLs to verify (one per line; typically a Search Console export)
				</label>
				<Textarea
					id="verify_urls"
					bind:value={urlsText}
					rows={8}
					placeholder={'/about\n/old/post\nhttps://old.example.com/contact'}
				/>
				<p class="text-xs text-text-muted mt-1.5">{parseURLs().length} URL(s) to check</p>
			</div>
			<div class="flex justify-end">
				<Button
					onclick={trigger}
					loading={triggering}
					disabled={!deployedDomain.trim() || parseURLs().length === 0 || parseURLs().length > 25000}
				>
					Run verification
				</Button>
			</div>
		</div>
	{:else}
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-3">
					<Badge variant={statusVariant(status)}>{status}</Badge>
					<span class="text-xs text-text-muted font-mono">{activeJobID.slice(0, 8)}</span>
					{#if liveState?.deployed_domain}
						<span class="text-xs text-text-muted">→ {liveState.deployed_domain}</span>
					{/if}
				</div>
				<div class="flex items-center gap-2">
					{#if !isTerminal}
						<Button size="sm" variant="danger" onclick={cancel} loading={cancelling}>
							Cancel
						</Button>
					{:else}
						<Button size="sm" variant="ghost" onclick={reset}>Run another</Button>
					{/if}
				</div>
			</div>

			<div>
				<div class="flex items-center justify-between text-xs text-text-muted mb-1.5 tabular-nums">
					<span>{processed} / {total} checked</span>
					<span>{progressPct}%</span>
				</div>
				<div class="h-2 w-full rounded-full bg-bg-elevated overflow-hidden">
					<div
						class="h-full bg-accent transition-all duration-300"
						style="width: {progressPct}%"
					></div>
				</div>
				<div class="mt-2 flex items-center gap-3 text-xs">
					<span class="text-success">{okCount} ok</span>
					<span class="text-danger">{failCount} failed</span>
				</div>
			</div>

			{#if status === 'failed' && liveState?.error}
				<div class="rounded-lg border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">
					<strong>Job failed:</strong> {liveState.error}
				</div>
			{/if}

			{#if status === 'done' && verifications.length > 0}
				<details class="group" open={failCount <= 25}>
					<summary class="cursor-pointer text-xs text-text-secondary hover:text-text-primary list-none flex items-center gap-1.5">
						<svg class="h-3 w-3 transition-transform group-open:rotate-90" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
						</svg>
						{verifications.length} failure{verifications.length === 1 ? '' : 's'}
					</summary>
					<div class="mt-3 overflow-x-auto rounded-lg border border-border-light">
						<table class="w-full text-xs">
							<thead class="bg-bg-elevated text-text-muted">
								<tr>
									<th class="text-left font-normal px-3 py-2">Source URL</th>
									<th class="text-left font-normal px-3 py-2">Status</th>
									<th class="text-left font-normal px-3 py-2">Hops</th>
									<th class="text-left font-normal px-3 py-2">Error</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-border-light">
								{#each verifications.slice(0, 200) as v (v.id)}
									<tr>
										<td class="px-3 py-1.5 font-mono truncate max-w-[40ch]">{v.source_url}</td>
										<td class="px-3 py-1.5 tabular-nums">{v.status_code || '-'}</td>
										<td class="px-3 py-1.5 tabular-nums">{v.hops}</td>
										<td class="px-3 py-1.5 text-danger truncate max-w-[40ch]">{v.error}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					{#if verifications.length > 200}
						<p class="mt-2 text-xs text-text-muted">Showing first 200 failures.</p>
					{/if}
				</details>
			{/if}
		</div>
	{/if}

	{#if !historyLoading && history.length > 0}
		<details class="mt-6 group">
			<summary class="cursor-pointer text-xs text-text-secondary hover:text-text-primary list-none flex items-center gap-1.5">
				<svg class="h-3 w-3 transition-transform group-open:rotate-90" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				Past runs ({history.length})
			</summary>
			<div class="mt-3 overflow-x-auto rounded-lg border border-border-light">
				<table class="w-full text-xs">
					<thead class="bg-bg-elevated text-text-muted">
						<tr>
							<th class="text-left font-normal px-3 py-2">Job</th>
							<th class="text-left font-normal px-3 py-2">Status</th>
							<th class="text-left font-normal px-3 py-2">Total</th>
							<th class="text-left font-normal px-3 py-2">OK</th>
							<th class="text-left font-normal px-3 py-2">Failed</th>
							<th class="text-left font-normal px-3 py-2">Started</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border-light">
						{#each history as job (job.id)}
							<tr>
								<td class="px-3 py-1.5 font-mono">{job.id.slice(0, 8)}</td>
								<td class="px-3 py-1.5">
									<Badge variant={statusVariant(job.status)}>{job.status}</Badge>
								</td>
								<td class="px-3 py-1.5 tabular-nums">{job.total_urls}</td>
								<td class="px-3 py-1.5 tabular-nums text-success">{job.ok_count}</td>
								<td class="px-3 py-1.5 tabular-nums text-danger">{job.fail_count}</td>
								<td class="px-3 py-1.5 text-text-muted">{job.started_at || job.created_at}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</details>
	{/if}
</Card>
