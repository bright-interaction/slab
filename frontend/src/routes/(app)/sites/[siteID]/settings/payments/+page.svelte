<script lang="ts">
	import * as settingsApi from '$lib/api/settings';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { categoryMap } from '$lib/settings/nginxPreview';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	function toBool(v: string | undefined): boolean {
		if (!v) return false;
		const s = v.toLowerCase();
		return s === '1' || s === 'true' || s === 'yes' || s === 'on';
	}

	let loading = $state(true);
	let saving = $state(false);

	let mollieApiKey = $state('');
	let mollieTestMode = $state(true);
	let revealKey = $state(false);

	type State = {
		mollieApiKey: string;
		mollieTestMode: boolean;
	};

	let initial: State = $state({
		mollieApiKey: '',
		mollieTestMode: true
	});

	async function load() {
		loading = true;
		try {
			const rows = await settingsApi.listByCategory(siteID, 'payments');
			const m = categoryMap(rows);
			mollieApiKey = m.mollie_api_key || '';
			const explicit = m.mollie_test_mode;
			if (explicit === undefined || explicit === '') {
				mollieTestMode = mollieApiKey.startsWith('live_') ? false : true;
			} else {
				mollieTestMode = toBool(explicit);
			}
			initial = { mollieApiKey, mollieTestMode };
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load payment settings.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const dirty = $derived(
		mollieApiKey !== initial.mollieApiKey || mollieTestMode !== initial.mollieTestMode
	);

	const keyPrefix = $derived(mollieApiKey.startsWith('live_') ? 'live' : mollieApiKey.startsWith('test_') ? 'test' : '');
	const keyMismatch = $derived(
		mollieApiKey !== '' &&
			keyPrefix !== '' &&
			((keyPrefix === 'live' && mollieTestMode) || (keyPrefix === 'test' && !mollieTestMode))
	);
	const configured = $derived(initial.mollieApiKey !== '');

	function discard() {
		mollieApiKey = initial.mollieApiKey;
		mollieTestMode = initial.mollieTestMode;
		revealKey = false;
	}

	function b(v: boolean): string {
		return v ? '1' : '0';
	}

	async function save() {
		if (saving || !dirty) return;
		const trimmed = mollieApiKey.trim();
		if (trimmed !== '' && !trimmed.startsWith('test_') && !trimmed.startsWith('live_')) {
			toast.error('Mollie API keys start with test_ or live_. Paste the full key from mollie.com -> Developers -> API keys.');
			return;
		}
		saving = true;
		try {
			const items: settingsApi.SettingUpsertInput[] = [
				{ category: 'payments', key: 'mollie_api_key', value: trimmed },
				{ category: 'payments', key: 'mollie_test_mode', value: b(mollieTestMode) }
			];
			await settingsApi.bulkUpsert(siteID, items);
			mollieApiKey = trimmed;
			initial = { mollieApiKey, mollieTestMode };
			toast.success(trimmed === '' ? 'Mollie API key cleared.' : 'Payment settings saved.');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save payment settings.';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	async function clearKey() {
		mollieApiKey = '';
		await save();
	}
</script>

<div class="mx-auto max-w-3xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Payments
		</h1>
		<p class="text-[13px] text-text-secondary">
			Mollie API key for the storefront checkout. One key per site; the same key handles
			payment creation, refunds, and webhook validation.
		</p>
	</header>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-8 flex flex-col gap-5">
			<Card padding="md">
				<div class="flex gap-3">
					<div
						class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent/10 text-accent"
					>
						<svg
							class="h-3.5 w-3.5"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z"
							/>
						</svg>
					</div>
					<div class="flex-1">
						<p class="text-[13px] text-text-primary">
							The checkout_form block POSTs to <code class="font-mono text-[11.5px]">/api/sites/{siteID}/checkout</code>;
							that endpoint reads this key, calls Mollie CreatePayment, and returns
							a hosted checkout URL the visitor is redirected to. No payment data
							ever touches your site directly. Mollie is the only provider wired in
							v1.
						</p>
						<p class="mt-1 text-[12px] text-text-muted">
							Get the API key at
							<a
								href="https://my.mollie.com/dashboard/developers/api-keys"
								class="text-accent underline-offset-2 hover:underline"
								target="_blank"
								rel="noopener noreferrer">mollie.com -> Developers -> API keys</a
							>. Use a test_ key while you author the storefront; switch to a live_ key
							before opening real orders.
						</p>
					</div>
				</div>
			</Card>

			<Card padding="md">
				<div class="flex items-start justify-between gap-4">
					<div>
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							Mollie API key
						</h2>
						<p class="mt-1 text-[12px] text-text-muted">
							Starts with <code class="font-mono text-[11.5px]">test_</code> or
							<code class="font-mono text-[11.5px]">live_</code>. Bearer-equivalent
							secret; treat like a password.
						</p>
					</div>
					{#if configured}
						<Badge variant="success">Configured</Badge>
					{:else}
						<Badge variant="warning">Not set</Badge>
					{/if}
				</div>
				<div class="mt-4 flex flex-col gap-3">
					<div class="flex gap-2">
						<Input
							type={revealKey ? 'text' : 'password'}
							placeholder={configured && !revealKey
								? mollieApiKey.slice(0, 5) + '••••••••'
								: 'test_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'}
							autocomplete="off"
							spellcheck={false}
							bind:value={mollieApiKey}
						/>
						<Button variant="secondary" onclick={() => (revealKey = !revealKey)}>
							{revealKey ? 'Hide' : 'Show'}
						</Button>
					</div>
					{#if keyMismatch}
						<p class="text-[12px] text-warning">
							The key prefix ({keyPrefix}_) and the test-mode switch disagree. Flip
							the switch to match the key you pasted, or paste the matching key for
							the mode you want.
						</p>
					{/if}
				</div>
			</Card>

			<Card padding="md">
				<div class="flex items-center justify-between gap-4">
					<div class="flex flex-col">
						<span class="text-[13px] text-text-primary">Test mode</span>
						<span class="text-[12px] text-text-muted">
							Off when you are accepting real payments. The flag is informational;
							Mollie routes based on the key prefix regardless. Use this to label
							the storefront in the Orders tab and to gate any future test-only UI.
						</span>
					</div>
					<Switch bind:checked={mollieTestMode} />
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Webhook
				</h2>
				<p class="mt-2 text-[13px] text-text-primary">
					Mollie posts payment status updates to:
				</p>
				<pre
					class="mt-2 overflow-x-auto rounded border border-border-light bg-bg-elevated p-3 font-mono text-[11.5px]"
				>https://{data.site.domain || '<your-domain>'}/api/sites/{siteID}/payments/mollie/webhook</pre>
				<p class="mt-2 text-[12px] text-text-muted">
					No webhook secret to paste; Mollie does not sign webhooks. Instead the
					handler fetches the payment via the API on every webhook hit, which
					verifies it belongs to this tenant's key. The endpoint is INSERT OR
					IGNORE idempotent so Mollie redeliveries are safe.
				</p>
			</Card>

			<div class="sticky bottom-6 mt-2 flex items-center justify-between gap-3 rounded-xl border border-border-light bg-bg-base/95 px-4 py-3 backdrop-blur">
				<div class="flex flex-col">
					<span class="text-[12px] text-text-secondary">
						{dirty ? 'Unsaved changes' : 'Up to date'}
					</span>
					{#if configured && !dirty}
						<button
							type="button"
							class="self-start text-[11px] text-text-muted underline-offset-2 hover:text-danger hover:underline"
							onclick={clearKey}
						>
							Remove the saved key
						</button>
					{/if}
				</div>
				<div class="flex gap-2">
					<Button variant="secondary" disabled={!dirty || saving} onclick={discard}>
						Discard
					</Button>
					<Button variant="primary" disabled={!dirty || saving} onclick={save}>
						{saving ? 'Saving...' : 'Save'}
					</Button>
				</div>
			</div>
		</div>
	{/if}
</div>
