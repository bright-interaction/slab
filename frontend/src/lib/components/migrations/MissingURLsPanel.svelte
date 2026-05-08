<script lang="ts">
	import { onMount } from 'svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as missing from '$lib/api/missingUrls';
	import { ApiError } from '$lib/api/client';
	import type { MissingURL, MissingURLRedirectStatus } from '$lib/api/missingUrls';

	let { siteID }: { siteID: string } = $props();

	let rows = $state<MissingURL[]>([]);
	let loading = $state(true);
	let limit = $state<50 | 200 | 500>(50);

	let redirectOpen = $state(false);
	let dismissOpen = $state(false);
	let selected = $state<MissingURL | null>(null);
	let toPath = $state('/');
	let statusCode = $state<MissingURLRedirectStatus>(301);
	let submitting = $state(false);

	const statusOptions = [
		{ value: '301', label: '301 Moved Permanently' },
		{ value: '302', label: '302 Found' },
		{ value: '307', label: '307 Temporary' },
		{ value: '308', label: '308 Permanent' },
		{ value: '410', label: '410 Gone' }
	];

	let statusCodeStr = $state('301');

	$effect(() => {
		const n = parseInt(statusCodeStr, 10);
		if (n === 301 || n === 302 || n === 307 || n === 308 || n === 410) {
			statusCode = n;
		}
	});

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			rows = await missing.listMissingURLs(siteID, limit);
		} catch {
			rows = [];
		} finally {
			loading = false;
		}
	}

	function openRedirect(row: MissingURL) {
		selected = row;
		toPath = '/';
		statusCodeStr = '301';
		statusCode = 301;
		redirectOpen = true;
	}

	function openDismiss(row: MissingURL) {
		selected = row;
		dismissOpen = true;
	}

	async function submitRedirect() {
		if (!selected) return;
		submitting = true;
		try {
			const resp = await missing.createRedirectFromMissing(siteID, selected.id, {
				to_path: toPath.trim(),
				status_code: statusCode
			});
			if (resp.status === 'already_existed') {
				toast.info(
					`A redirect already covered ${selected.path} → ${resp.to}; cleared the missing-url row`
				);
			} else {
				toast.success(`Redirect created: ${resp.from} → ${resp.to} (${resp.status_code})`);
			}
			rows = rows.filter((r) => r.id !== selected!.id);
			redirectOpen = false;
			selected = null;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Create redirect failed');
		} finally {
			submitting = false;
		}
	}

	async function submitDismiss() {
		if (!selected) return;
		submitting = true;
		try {
			await missing.dismissMissingURL(siteID, selected.id);
			toast.success(`Dismissed ${selected.path}`);
			rows = rows.filter((r) => r.id !== selected!.id);
			dismissOpen = false;
			selected = null;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Dismiss failed');
		} finally {
			submitting = false;
		}
	}

	async function setLimit(n: 50 | 200 | 500) {
		limit = n;
		await load();
	}
</script>

<Card>
	<div class="flex items-baseline justify-between mb-4">
		<div>
			<h2 class="text-sm font-medium text-text-primary">Missing URLs (404 inbox)</h2>
			<p class="text-xs text-text-muted mt-1">
				404s captured in real time by the analytics parser. One-click any row into a 301/302/307/308/410 redirect.
			</p>
		</div>
		<div class="flex items-center gap-1 text-xs">
			<button
				class="px-2 py-1 rounded text-text-muted hover:text-text-primary {limit === 50 ? 'text-text-primary font-medium' : ''}"
				onclick={() => setLimit(50)}
				type="button"
			>50</button>
			<button
				class="px-2 py-1 rounded text-text-muted hover:text-text-primary {limit === 200 ? 'text-text-primary font-medium' : ''}"
				onclick={() => setLimit(200)}
				type="button"
			>200</button>
			<button
				class="px-2 py-1 rounded text-text-muted hover:text-text-primary {limit === 500 ? 'text-text-primary font-medium' : ''}"
				onclick={() => setLimit(500)}
				type="button"
			>500</button>
		</div>
	</div>

	{#if loading}
		<div class="space-y-2">
			<Skeleton class="h-8" />
			<Skeleton class="h-8" />
			<Skeleton class="h-8" />
		</div>
	{:else if rows.length === 0}
		<EmptyState
			title="No missing URLs"
			description="404s logged by the analytics parser will appear here. Visit a 404 path on the live site to seed an entry."
		/>
	{:else}
		<div class="overflow-x-auto rounded-lg border border-border-light">
			<table class="w-full text-xs">
				<thead class="bg-bg-elevated text-text-muted">
					<tr>
						<th class="text-left font-normal px-3 py-2">Path</th>
						<th class="text-left font-normal px-3 py-2 tabular-nums">Hits</th>
						<th class="text-left font-normal px-3 py-2">First seen</th>
						<th class="text-left font-normal px-3 py-2">Last seen</th>
						<th class="text-left font-normal px-3 py-2">Referer</th>
						<th class="text-right font-normal px-3 py-2">Action</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-light">
					{#each rows as row (row.id)}
						<tr>
							<td class="px-3 py-1.5 font-mono truncate max-w-[32ch]">{row.path}</td>
							<td class="px-3 py-1.5 tabular-nums">
								<Badge variant={row.hit_count >= 10 ? 'warning' : 'default'}>{row.hit_count}</Badge>
							</td>
							<td class="px-3 py-1.5 text-text-muted">{row.first_seen}</td>
							<td class="px-3 py-1.5 text-text-muted">{row.last_seen}</td>
							<td class="px-3 py-1.5 text-text-muted truncate max-w-[24ch]">{row.referer || '-'}</td>
							<td class="px-3 py-1.5 text-right">
								<button
									type="button"
									class="text-accent hover:underline mr-3"
									onclick={() => openRedirect(row)}
								>Redirect</button>
								<button
									type="button"
									class="text-text-muted hover:text-danger"
									onclick={() => openDismiss(row)}
								>Dismiss</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</Card>

<Dialog
	bind:open={redirectOpen}
	title="Create redirect"
	description={selected ? `From: ${selected.path}` : ''}
>
	<div class="space-y-3">
		<div>
			<Input
				label="Destination path"
				bind:value={toPath}
				placeholder="/new-location"
				hint="Must start with /. Off-site, open-redirect, and no-op redirects are rejected."
			/>
		</div>
		<div>
			<label for="status_code" class="block text-xs text-text-muted mb-1.5">Status code</label>
			<Select bind:value={statusCodeStr} options={statusOptions} />
		</div>
	</div>
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (redirectOpen = false)} disabled={submitting}>
				Cancel
			</Button>
			<Button onclick={submitRedirect} loading={submitting} disabled={!toPath.trim()}>
				Create redirect
			</Button>
		</div>
	{/snippet}
</Dialog>

<ConfirmDialog
	bind:open={dismissOpen}
	title="Dismiss this 404?"
	message={selected
		? `${selected.path} (${selected.hit_count} hits). The row will reappear if the path 404s again. To make a dismissal permanent, create a 410 redirect.`
		: ''}
	confirmText="Dismiss"
	variant="danger"
	onconfirm={submitDismiss}
/>
