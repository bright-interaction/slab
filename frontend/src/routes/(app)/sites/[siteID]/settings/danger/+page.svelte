<script lang="ts">
	import { goto } from '$app/navigation';
	import * as sitesApi from '$lib/api/sites';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { AlertTriangle } from 'lucide-svelte';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);
	const slug = $derived(data.site.slug);

	let dialogOpen = $state(false);
	let confirmText = $state('');
	let deleting = $state(false);

	const matches = $derived(confirmText.trim() === slug);

	async function deleteSite() {
		if (!matches || deleting) return;
		deleting = true;
		try {
			await sitesApi.remove(siteID);
			toast.success(`Site "${data.site.name}" deleted.`);
			dialogOpen = false;
			void goto('/sites');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to delete site.';
			toast.error(msg);
		} finally {
			deleting = false;
		}
	}

	function openDialog() {
		confirmText = '';
		dialogOpen = true;
	}
</script>

<div class="mx-auto max-w-3xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Danger zone
		</h1>
		<p class="text-[13px] text-text-secondary">
			Irreversible actions. Read carefully before proceeding.
		</p>
	</header>

	<div class="mt-8">
		<Card padding="md" class="border-danger/30">
			<div class="flex items-start gap-4">
				<span
					class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-danger/10 text-danger"
				>
					<AlertTriangle size={16} strokeWidth={1.5} />
				</span>
				<div class="flex-1">
					<h2 class="text-[14px] font-medium text-text-primary">Delete this site</h2>
					<p class="mt-1 text-[13px] text-text-secondary">
						Permanently removes the site, all pages, blocks, media files, settings,
						evaluations and build history. This cannot be undone. Live deployments at the
						domain will stop serving on next propagation.
					</p>
				</div>
				<Button variant="danger" onclick={openDialog}>Delete site</Button>
			</div>
		</Card>
	</div>
</div>

<Dialog
	bind:open={dialogOpen}
	title="Delete this site?"
	description="Type the site slug to confirm. This action cannot be reversed."
	size="md"
>
	<div class="flex flex-col gap-4">
		<div class="rounded-lg border border-danger/30 bg-danger/5 px-3 py-2.5">
			<p class="text-[12px] text-text-secondary">
				You are about to delete <span class="font-medium text-text-primary">{data.site.name}</span
				>. Slug:
				<code class="font-mono text-[12px] text-text-primary">{slug}</code>
			</p>
		</div>
		<Input
			label={`Type "${slug}" to confirm`}
			placeholder={slug}
			bind:value={confirmText}
			autocomplete="off"
		/>
	</div>
	{#snippet footer()}
		<Button variant="secondary" onclick={() => (dialogOpen = false)} disabled={deleting}>
			Cancel
		</Button>
		<Button variant="danger" onclick={deleteSite} disabled={!matches} loading={deleting}>
			Delete forever
		</Button>
	{/snippet}
</Dialog>
