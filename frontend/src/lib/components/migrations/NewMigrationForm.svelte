<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as migrations from '$lib/api/migrations';
	import type { MigrationSourceType } from '$lib/api/migrations';
	import { ApiError } from '$lib/api/client';

	let {
		siteID,
		onSuccess
	}: {
		siteID: string;
		onSuccess: (migrationID: string) => void;
	} = $props();

	let sourceType = $state<MigrationSourceType>('sitemap');
	let sourceURL = $state('');
	let wpAuthHeader = $state('');
	let webflowSiteID = $state('');
	let webflowAuthToken = $state('');
	let ghostContentKey = $state('');
	let submitting = $state(false);
	let lastError = $state<string | null>(null);

	const sourceTypes = [
		{ value: 'sitemap', label: 'Sitemap (XML)' },
		{ value: 'wordpress', label: 'WordPress (REST API)' },
		{ value: 'webflow', label: 'Webflow (CMS API v2)' },
		{ value: 'ghost', label: 'Ghost (Content API)' }
	];

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (!sourceURL.trim()) return;
		submitting = true;
		lastError = null;
		try {
			const resp = await migrations.startMigration(siteID, {
				source_type: sourceType,
				source_url: sourceURL.trim(),
				wordpress_auth_header: wpAuthHeader.trim() || undefined,
				webflow_site_id: webflowSiteID.trim() || undefined,
				webflow_auth_token: webflowAuthToken.trim() || undefined,
				ghost_content_key: ghostContentKey.trim() || undefined
			});
			toast.success(
				`Crawl complete: ${resp.manifest?.stats?.pages_found ?? 0} pages, ${resp.manifest?.stats?.media_found ?? 0} media`
			);
			onSuccess(resp.id);
		} catch (err) {
			lastError = err instanceof ApiError ? err.message : 'Crawl failed';
			toast.error(lastError);
		} finally {
			submitting = false;
		}
	}
</script>

<Card>
	<h2 class="text-sm font-medium text-text-primary mb-1">Start a new migration</h2>
	<p class="text-xs text-text-muted mb-4">
		Crawls run synchronously. Large sites (1000+ posts) take minutes; keep this tab open until
		the manifest lands.
	</p>

	<form onsubmit={submit} class="space-y-4">
		<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
			<div>
				<label for="source_type" class="block text-xs text-text-muted mb-1.5">Source type</label>
				<Select bind:value={sourceType} options={sourceTypes} />
			</div>
			<div class="md:col-span-2">
				<label for="source_url" class="block text-xs text-text-muted mb-1.5">Source URL</label>
				<Input

					bind:value={sourceURL}
					placeholder={sourceType === 'sitemap'
						? 'https://example.com/sitemap.xml'
						: sourceType === 'wordpress'
							? 'https://example.com (root, /wp-json appended)'
							: sourceType === 'ghost'
								? 'https://example.com (root, /ghost/api/content/ appended)'
								: 'https://api.webflow.com (or override)'}
					required
				/>
			</div>
		</div>

		{#if sourceType === 'wordpress'}
			<div>
				<label for="wp_auth" class="block text-xs text-text-muted mb-1.5">
					WordPress Application Password (optional, for drafts + private posts)
				</label>
				<Input

					bind:value={wpAuthHeader}
					placeholder="Basic base64(user:apppass)"
					autocomplete="off"
				/>
			</div>
		{/if}

		{#if sourceType === 'webflow'}
			<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
				<div>
					<label for="wf_site" class="block text-xs text-text-muted mb-1.5">Webflow site_id</label>
					<Input bind:value={webflowSiteID} required />
				</div>
				<div>
					<label for="wf_token" class="block text-xs text-text-muted mb-1.5">
						Webflow auth token
					</label>
					<Input

						bind:value={webflowAuthToken}
						type="password"
						autocomplete="off"
						required
					/>
				</div>
			</div>
		{/if}

		{#if sourceType === 'ghost'}
			<div>
				<label for="ghost_key" class="block text-xs text-text-muted mb-1.5">
					Ghost Content API key
				</label>
				<Input

					bind:value={ghostContentKey}
					type="password"
					autocomplete="off"
					required
				/>
			</div>
		{/if}

		{#if lastError}
			<p class="text-xs text-danger">{lastError}</p>
		{/if}

		<div class="flex justify-end">
			<Button type="submit" loading={submitting} disabled={!sourceURL.trim()}>
				{submitting ? 'Crawling...' : 'Start crawl'}
			</Button>
		</div>
	</form>
</Card>
