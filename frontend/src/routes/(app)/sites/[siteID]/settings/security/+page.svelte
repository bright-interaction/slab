<script lang="ts">
	import * as settingsApi from '$lib/api/settings';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { categoryMap } from '$lib/settings/nginxPreview';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	const xfoOptions = [
		{ value: 'DENY', label: 'DENY' },
		{ value: 'SAMEORIGIN', label: 'SAMEORIGIN' }
	];

	const referrerOptions = [
		{ value: 'no-referrer', label: 'no-referrer' },
		{ value: 'no-referrer-when-downgrade', label: 'no-referrer-when-downgrade' },
		{ value: 'origin', label: 'origin' },
		{ value: 'origin-when-cross-origin', label: 'origin-when-cross-origin' },
		{ value: 'same-origin', label: 'same-origin' },
		{ value: 'strict-origin', label: 'strict-origin' },
		{ value: 'strict-origin-when-cross-origin', label: 'strict-origin-when-cross-origin' },
		{ value: 'unsafe-url', label: 'unsafe-url' }
	];

	function toBool(v: string | undefined): boolean {
		if (!v) return false;
		const s = v.toLowerCase();
		return s === '1' || s === 'true' || s === 'yes' || s === 'on';
	}

	let loading = $state(true);
	let saving = $state(false);

	let hstsEnabled = $state(false);
	let hstsMaxAge = $state(31536000);
	let hstsPreload = $state(false);
	let cspEnabled = $state(false);
	let cspExtra = $state('');
	let xFrameOptions = $state('SAMEORIGIN');
	let xContentTypeOptions = $state(true);
	let referrerPolicy = $state('strict-origin-when-cross-origin');
	let permissionsPolicy = $state('camera=(), microphone=(), geolocation=()');
	let httpsRedirect = $state(true);

	type State = {
		hstsEnabled: boolean;
		hstsMaxAge: number;
		hstsPreload: boolean;
		cspEnabled: boolean;
		cspExtra: string;
		xFrameOptions: string;
		xContentTypeOptions: boolean;
		referrerPolicy: string;
		permissionsPolicy: string;
		httpsRedirect: boolean;
	};

	let initial: State = {
		hstsEnabled: false,
		hstsMaxAge: 31536000,
		hstsPreload: false,
		cspEnabled: false,
		cspExtra: '',
		xFrameOptions: 'SAMEORIGIN',
		xContentTypeOptions: true,
		referrerPolicy: 'strict-origin-when-cross-origin',
		permissionsPolicy: 'camera=(), microphone=(), geolocation=()',
		httpsRedirect: true
	};

	async function load() {
		loading = true;
		try {
			const rows = await settingsApi.listByCategory(siteID, 'security');
			const m = categoryMap(rows);

			hstsEnabled = toBool(m.hsts_enabled);
			hstsMaxAge = m.hsts_max_age ? parseInt(m.hsts_max_age, 10) || 31536000 : 31536000;
			hstsPreload = toBool(m.hsts_preload);
			cspEnabled = toBool(m.csp_enabled);
			cspExtra = m.csp_extra_directives || '';
			xFrameOptions = m.x_frame_options || 'SAMEORIGIN';
			xContentTypeOptions = m.x_content_type_options ? toBool(m.x_content_type_options) : true;
			referrerPolicy = m.referrer_policy || 'strict-origin-when-cross-origin';
			permissionsPolicy = m.permissions_policy || 'camera=(), microphone=(), geolocation=()';
			httpsRedirect = m.https_redirect ? toBool(m.https_redirect) : true;

			initial = {
				hstsEnabled,
				hstsMaxAge,
				hstsPreload,
				cspEnabled,
				cspExtra,
				xFrameOptions,
				xContentTypeOptions,
				referrerPolicy,
				permissionsPolicy,
				httpsRedirect
			};
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load settings.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const dirty = $derived(
		hstsEnabled !== initial.hstsEnabled ||
			hstsMaxAge !== initial.hstsMaxAge ||
			hstsPreload !== initial.hstsPreload ||
			cspEnabled !== initial.cspEnabled ||
			cspExtra !== initial.cspExtra ||
			xFrameOptions !== initial.xFrameOptions ||
			xContentTypeOptions !== initial.xContentTypeOptions ||
			referrerPolicy !== initial.referrerPolicy ||
			permissionsPolicy !== initial.permissionsPolicy ||
			httpsRedirect !== initial.httpsRedirect
	);

	function discard() {
		hstsEnabled = initial.hstsEnabled;
		hstsMaxAge = initial.hstsMaxAge;
		hstsPreload = initial.hstsPreload;
		cspEnabled = initial.cspEnabled;
		cspExtra = initial.cspExtra;
		xFrameOptions = initial.xFrameOptions;
		xContentTypeOptions = initial.xContentTypeOptions;
		referrerPolicy = initial.referrerPolicy;
		permissionsPolicy = initial.permissionsPolicy;
		httpsRedirect = initial.httpsRedirect;
	}

	function b(v: boolean): string {
		return v ? '1' : '0';
	}

	async function save() {
		if (saving || !dirty) return;
		saving = true;
		try {
			const items: settingsApi.SettingUpsertInput[] = [
				{ category: 'security', key: 'hsts_enabled', value: b(hstsEnabled) },
				{ category: 'security', key: 'hsts_max_age', value: String(hstsMaxAge) },
				{ category: 'security', key: 'hsts_preload', value: b(hstsPreload) },
				{ category: 'security', key: 'csp_enabled', value: b(cspEnabled) },
				{ category: 'security', key: 'csp_extra_directives', value: cspExtra },
				{ category: 'security', key: 'x_frame_options', value: xFrameOptions },
				{
					category: 'security',
					key: 'x_content_type_options',
					value: b(xContentTypeOptions)
				},
				{ category: 'security', key: 'referrer_policy', value: referrerPolicy },
				{ category: 'security', key: 'permissions_policy', value: permissionsPolicy },
				{ category: 'security', key: 'https_redirect', value: b(httpsRedirect) }
			];
			await settingsApi.bulkUpsert(siteID, items);

			initial = {
				hstsEnabled,
				hstsMaxAge,
				hstsPreload,
				cspEnabled,
				cspExtra,
				xFrameOptions,
				xContentTypeOptions,
				referrerPolicy,
				permissionsPolicy,
				httpsRedirect
			};
			toast.success('Security settings saved.');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save settings.';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}
</script>

<div class="mx-auto max-w-3xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Security
		</h1>
		<p class="text-[13px] text-text-secondary">
			HTTP response headers applied by nginx. Changes take effect on next build.
		</p>
	</header>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-8 flex flex-col gap-5">
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Transport</h2>
				<div class="mt-4 flex flex-col gap-3">
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">HTTPS redirect</span>
							<span class="text-[12px] text-text-muted">Force HTTP to HTTPS at the edge.</span>
						</div>
						<Switch bind:checked={httpsRedirect} />
					</div>
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">HSTS</span>
							<span class="text-[12px] text-text-muted">
								Strict-Transport-Security header.
							</span>
						</div>
						<Switch bind:checked={hstsEnabled} />
					</div>
					{#if hstsEnabled}
						<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<Input
								label="HSTS max age (seconds)"
								type="number"
								min="0"
								bind:value={hstsMaxAge}
								hint="31536000 = 1 year."
							/>
							<div class="flex items-end">
								<div class="flex items-center justify-between gap-4 w-full pb-2">
									<span class="text-[13px] text-text-primary">Preload</span>
									<Switch bind:checked={hstsPreload} />
								</div>
							</div>
						</div>
					{/if}
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Content Security Policy
				</h2>
				<div class="mt-4 flex flex-col gap-3">
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">Enable CSP</span>
							<span class="text-[12px] text-text-muted">
								Content-Security-Policy with strict defaults.
							</span>
						</div>
						<Switch bind:checked={cspEnabled} />
					</div>
					{#if cspEnabled}
						<Textarea
							label="Extra directives"
							rows={3}
							placeholder="connect-src 'self' https://api.example.com;"
							hint="Appended to base CSP. Keep semicolon-separated."
							bind:value={cspExtra}
						/>
					{/if}
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Headers
				</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-secondary">X-Frame-Options</span>
						<Select options={xfoOptions} bind:value={xFrameOptions} />
					</div>
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-secondary">Referrer-Policy</span>
						<Select options={referrerOptions} bind:value={referrerPolicy} />
					</div>
				</div>
				<div class="mt-4 flex items-center justify-between gap-4">
					<div class="flex flex-col">
						<span class="text-[13px] text-text-primary">X-Content-Type-Options</span>
						<span class="text-[12px] text-text-muted">Send nosniff. Recommended on.</span>
					</div>
					<Switch bind:checked={xContentTypeOptions} />
				</div>
				<div class="mt-4">
					<Textarea
						label="Permissions-Policy"
						rows={3}
						placeholder="camera=(), microphone=(), geolocation=()"
						hint="Comma-separated feature directives. Empty parens = block."
						bind:value={permissionsPolicy}
					/>
				</div>
			</Card>

			<div
				class="sticky bottom-0 -mx-6 flex items-center justify-between gap-3 border-t border-border-light bg-bg-surface/85 px-6 py-3 backdrop-blur"
			>
				<span class="text-[12px] text-text-muted">
					{dirty ? 'Unsaved changes. Apply on next build.' : 'Up to date.'}
				</span>
				<div class="flex items-center gap-2">
					<Button variant="ghost" onclick={discard} disabled={!dirty || saving}>Discard</Button>
					<Button variant="primary" onclick={save} loading={saving} disabled={!dirty}>
						Save
					</Button>
				</div>
			</div>
		</div>
	{/if}
</div>
