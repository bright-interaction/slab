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

	const xXssOptions = [
		{ value: '0', label: '0 (off, OWASP recommendation)' },
		{ value: '1', label: '1' },
		{ value: '1; mode=block', label: '1; mode=block (default; scanner parity)' }
	];

	const xPermittedOptions = [
		{ value: 'none', label: 'none (default)' },
		{ value: 'master-only', label: 'master-only' },
		{ value: 'by-content-type', label: 'by-content-type' },
		{ value: 'all', label: 'all' }
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
	let frameAncestors = $state("'none'");
	let xFrameOptions = $state('SAMEORIGIN');
	let xContentTypeOptions = $state(true);
	let referrerPolicy = $state('strict-origin-when-cross-origin');
	let permissionsPolicy = $state('camera=(), microphone=(), geolocation=()');
	let coop = $state('same-origin');
	let corp = $state('same-origin');
	let coep = $state('');
	let xXssProtection = $state('1; mode=block');
	let xPermittedCrossDomain = $state('none');
	let httpsRedirect = $state(true);

	type State = {
		hstsEnabled: boolean;
		hstsMaxAge: number;
		hstsPreload: boolean;
		cspEnabled: boolean;
		cspExtra: string;
		frameAncestors: string;
		xFrameOptions: string;
		xContentTypeOptions: boolean;
		referrerPolicy: string;
		permissionsPolicy: string;
		coop: string;
		corp: string;
		coep: string;
		xXssProtection: string;
		xPermittedCrossDomain: string;
		httpsRedirect: boolean;
	};

	let initial: State = $state({
		hstsEnabled: false,
		hstsMaxAge: 31536000,
		hstsPreload: false,
		cspEnabled: false,
		cspExtra: '',
		frameAncestors: "'none'",
		xFrameOptions: 'SAMEORIGIN',
		xContentTypeOptions: true,
		referrerPolicy: 'strict-origin-when-cross-origin',
		permissionsPolicy: 'camera=(), microphone=(), geolocation=()',
		coop: 'same-origin',
		corp: 'same-origin',
		coep: '',
		xXssProtection: '1; mode=block',
		xPermittedCrossDomain: 'none',
		httpsRedirect: true
	});

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
			frameAncestors = m.frame_ancestors || "'none'";
			xFrameOptions = m.x_frame_options || 'SAMEORIGIN';
			xContentTypeOptions = m.x_content_type_options ? toBool(m.x_content_type_options) : true;
			referrerPolicy = m.referrer_policy || 'strict-origin-when-cross-origin';
			permissionsPolicy = m.permissions_policy || 'camera=(), microphone=(), geolocation=()';
			coop = m.coop || 'same-origin';
			corp = m.corp || 'same-origin';
			coep = m.coep || '';
			xXssProtection = m.x_xss_protection || '1; mode=block';
			xPermittedCrossDomain = m.x_permitted_cross_domain_policies || 'none';
			httpsRedirect = m.https_redirect ? toBool(m.https_redirect) : true;

			initial = {
				hstsEnabled,
				hstsMaxAge,
				hstsPreload,
				cspEnabled,
				cspExtra,
				frameAncestors,
				xFrameOptions,
				xContentTypeOptions,
				referrerPolicy,
				permissionsPolicy,
				coop,
				corp,
				coep,
				xXssProtection,
				xPermittedCrossDomain,
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
			frameAncestors !== initial.frameAncestors ||
			xFrameOptions !== initial.xFrameOptions ||
			xContentTypeOptions !== initial.xContentTypeOptions ||
			referrerPolicy !== initial.referrerPolicy ||
			permissionsPolicy !== initial.permissionsPolicy ||
			coop !== initial.coop ||
			corp !== initial.corp ||
			coep !== initial.coep ||
			xXssProtection !== initial.xXssProtection ||
			xPermittedCrossDomain !== initial.xPermittedCrossDomain ||
			httpsRedirect !== initial.httpsRedirect
	);

	function discard() {
		hstsEnabled = initial.hstsEnabled;
		hstsMaxAge = initial.hstsMaxAge;
		hstsPreload = initial.hstsPreload;
		cspEnabled = initial.cspEnabled;
		cspExtra = initial.cspExtra;
		frameAncestors = initial.frameAncestors;
		xFrameOptions = initial.xFrameOptions;
		xContentTypeOptions = initial.xContentTypeOptions;
		referrerPolicy = initial.referrerPolicy;
		permissionsPolicy = initial.permissionsPolicy;
		coop = initial.coop;
		corp = initial.corp;
		coep = initial.coep;
		xXssProtection = initial.xXssProtection;
		xPermittedCrossDomain = initial.xPermittedCrossDomain;
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
				{ category: 'security', key: 'frame_ancestors', value: frameAncestors },
				{ category: 'security', key: 'x_frame_options', value: xFrameOptions },
				{
					category: 'security',
					key: 'x_content_type_options',
					value: b(xContentTypeOptions)
				},
				{ category: 'security', key: 'referrer_policy', value: referrerPolicy },
				{ category: 'security', key: 'permissions_policy', value: permissionsPolicy },
				{ category: 'security', key: 'coop', value: coop },
				{ category: 'security', key: 'corp', value: corp },
				{ category: 'security', key: 'coep', value: coep },
				{ category: 'security', key: 'x_xss_protection', value: xXssProtection },
				{
					category: 'security',
					key: 'x_permitted_cross_domain_policies',
					value: xPermittedCrossDomain
				},
				{ category: 'security', key: 'https_redirect', value: b(httpsRedirect) }
			];
			await settingsApi.bulkUpsert(siteID, items);

			initial = {
				hstsEnabled,
				hstsMaxAge,
				hstsPreload,
				cspEnabled,
				cspExtra,
				frameAncestors,
				xFrameOptions,
				xContentTypeOptions,
				referrerPolicy,
				permissionsPolicy,
				coop,
				corp,
				coep,
				xXssProtection,
				xPermittedCrossDomain,
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

<div class="mx-auto max-w-7xl px-6 py-8">
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
						<Switch bind:checked={httpsRedirect} ariaLabel="Force HTTPS redirect" />
					</div>
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">HSTS</span>
							<span class="text-[12px] text-text-muted">
								Strict-Transport-Security header.
							</span>
						</div>
						<Switch bind:checked={hstsEnabled} ariaLabel="Enable HSTS" />
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
									<Switch bind:checked={hstsPreload} ariaLabel="Enable HSTS preload" />
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
						<Switch bind:checked={cspEnabled} ariaLabel="Enable CSP" />
					</div>
					{#if cspEnabled}
						<Input
							label="frame-ancestors"
							hint="Who can embed THIS site in an iframe. 'none' (default) is anti-clickjacking. Use 'self' or a host list to allow embedding (e.g. for a customer portal)."
							placeholder="'none' or 'self' or https://app.example.com"
							bind:value={frameAncestors}
						/>
						<Textarea
							label="Extra directives"
							rows={3}
							placeholder="report-uri /csp-report"
							hint="Appended to the auto-built CSP. Use this for report-uri, sandbox, or anything atomicsite doesn't expose as its own field. Trailing semicolons are stripped."
							bind:value={cspExtra}
						/>
						<p class="rounded-lg border border-border-light bg-bg-elevated/50 px-3 py-2 text-[12px] text-text-muted">
							Need to whitelist an iframe (cal.com, YouTube), an image CDN, or
							a payment script? Use the
							<a href="./allowed-scripts" class="text-accent underline-offset-2 hover:underline">
								Trusted external domains
							</a>
							page. Each entry there picks the kind (script / iframe / image /
							media / connect / all), and the builder routes it into the right
							CSP directive automatically.
						</p>
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
					<Switch bind:checked={xContentTypeOptions} ariaLabel="Send X-Content-Type-Options nosniff" />
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

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Cross-Origin Isolation
				</h2>
				<p class="mt-2 text-[12px] text-text-muted">
					Controls how this page shares a browsing context with documents on
					other origins. Same-origin defaults are safe; loosen only if a
					third-party iframe or popup needs to interact with the site.
				</p>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-secondary">
							Cross-Origin-Opener-Policy (COOP)
						</span>
						<Select
							options={[
								{ value: 'same-origin', label: 'same-origin (default)' },
								{ value: 'same-origin-allow-popups', label: 'same-origin-allow-popups' },
								{ value: 'unsafe-none', label: 'unsafe-none' }
							]}
							bind:value={coop}
						/>
					</div>
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-secondary">
							Cross-Origin-Resource-Policy (CORP)
						</span>
						<Select
							options={[
								{ value: 'same-origin', label: 'same-origin (default)' },
								{ value: 'same-site', label: 'same-site' },
								{ value: 'cross-origin', label: 'cross-origin' }
							]}
							bind:value={corp}
						/>
					</div>
					<div class="flex flex-col gap-1.5 sm:col-span-2">
						<span class="text-[12px] font-medium text-text-secondary">
							Cross-Origin-Embedder-Policy (COEP)
						</span>
						<Select
							options={[
								{ value: '', label: 'Off (default; preserves third-party embeds)' },
								{ value: 'require-corp', label: 'require-corp (enables SharedArrayBuffer)' },
								{ value: 'credentialless', label: 'credentialless' }
							]}
							bind:value={coep}
						/>
						<p class="text-[11px] text-text-muted">
							require-corp blocks iframes without Cross-Origin-Resource-Policy.
							Use only if you need SharedArrayBuffer / cross-origin isolation.
						</p>
					</div>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Legacy headers (scanner parity)
				</h2>
				<p class="mt-2 text-[12px] text-text-muted">
					Modern browsers ignore both of these but Site Inspector and
					securityheaders.com still grade for them. Defaults score A+; flip
					only if you have a specific reason.
				</p>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-secondary">
							X-XSS-Protection
						</span>
						<Select options={xXssOptions} bind:value={xXssProtection} />
						<p class="text-[11px] text-text-muted">
							Real XSS protection comes from the CSP. OWASP recommends 0; scanners
							expect 1; mode=block.
						</p>
					</div>
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-secondary">
							X-Permitted-Cross-Domain-Policies
						</span>
						<Select options={xPermittedOptions} bind:value={xPermittedCrossDomain} />
						<p class="text-[11px] text-text-muted">
							Locks down legacy Adobe Flash / Silverlight crossdomain.xml lookup.
							none has zero downside on modern stacks.
						</p>
					</div>
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
