<script lang="ts">
	import * as profileApi from '$lib/api/profile';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Site, SiteProfile } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	const countryOptions = [
		{ value: '', label: 'Select country' },
		{ value: 'SE', label: 'Sweden' },
		{ value: 'NO', label: 'Norway' },
		{ value: 'DK', label: 'Denmark' },
		{ value: 'FI', label: 'Finland' },
		{ value: 'DE', label: 'Germany' },
		{ value: 'NL', label: 'Netherlands' },
		{ value: 'FR', label: 'France' },
		{ value: 'GB', label: 'United Kingdom' },
		{ value: 'IE', label: 'Ireland' },
		{ value: 'US', label: 'United States' }
	];

	let loading = $state(true);
	let saving = $state(false);

	let businessName = $state('');
	let registrationNr = $state('');
	let country = $state('');
	let contactEmail = $state('');
	let contactPhone = $state('');
	let privacyEmail = $state('');
	let securityEmail = $state('');
	let addressLine1 = $state('');
	let addressLine2 = $state('');
	let city = $state('');
	let postalCode = $state('');

	type State = {
		businessName: string;
		registrationNr: string;
		country: string;
		contactEmail: string;
		contactPhone: string;
		privacyEmail: string;
		securityEmail: string;
		addressLine1: string;
		addressLine2: string;
		city: string;
		postalCode: string;
	};

	let initial: State = $state({
		businessName: '',
		registrationNr: '',
		country: '',
		contactEmail: '',
		contactPhone: '',
		privacyEmail: '',
		securityEmail: '',
		addressLine1: '',
		addressLine2: '',
		city: '',
		postalCode: ''
	});

	function isProfile(v: profileApi.ProfileResponse): v is SiteProfile {
		return 'business_name' in v;
	}

	async function load() {
		loading = true;
		try {
			const resp = await profileApi.get(siteID);
			if (isProfile(resp)) {
				businessName = resp.business_name || '';
				registrationNr = resp.registration_nr || '';
				country = resp.country || '';
				contactEmail = resp.contact_email || '';
				contactPhone = resp.contact_phone || '';
				privacyEmail = resp.privacy_email || '';
				securityEmail = resp.security_email || '';
				addressLine1 = resp.address_line1 || '';
				addressLine2 = resp.address_line2 || '';
				city = resp.city || '';
				postalCode = resp.postal_code || '';
			}
			initial = {
				businessName,
				registrationNr,
				country,
				contactEmail,
				contactPhone,
				privacyEmail,
				securityEmail,
				addressLine1,
				addressLine2,
				city,
				postalCode
			};
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load profile.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const dirty = $derived(
		businessName !== initial.businessName ||
			registrationNr !== initial.registrationNr ||
			country !== initial.country ||
			contactEmail !== initial.contactEmail ||
			contactPhone !== initial.contactPhone ||
			privacyEmail !== initial.privacyEmail ||
			securityEmail !== initial.securityEmail ||
			addressLine1 !== initial.addressLine1 ||
			addressLine2 !== initial.addressLine2 ||
			city !== initial.city ||
			postalCode !== initial.postalCode
	);

	function discard() {
		businessName = initial.businessName;
		registrationNr = initial.registrationNr;
		country = initial.country;
		contactEmail = initial.contactEmail;
		contactPhone = initial.contactPhone;
		privacyEmail = initial.privacyEmail;
		securityEmail = initial.securityEmail;
		addressLine1 = initial.addressLine1;
		addressLine2 = initial.addressLine2;
		city = initial.city;
		postalCode = initial.postalCode;
	}

	async function save() {
		if (saving || !dirty) return;
		saving = true;
		try {
			await profileApi.upsert(siteID, {
				business_name: businessName,
				registration_nr: registrationNr,
				country,
				contact_email: contactEmail,
				contact_phone: contactPhone,
				privacy_email: privacyEmail,
				security_email: securityEmail,
				address_line1: addressLine1,
				address_line2: addressLine2,
				city,
				postal_code: postalCode
			});
			initial = {
				businessName,
				registrationNr,
				country,
				contactEmail,
				contactPhone,
				privacyEmail,
				securityEmail,
				addressLine1,
				addressLine2,
				city,
				postalCode
			};
			toast.success('Profile saved.');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save profile.';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}
</script>

<div class="mx-auto max-w-3xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Profile
		</h1>
		<p class="text-[13px] text-text-secondary">
			Business identity and legal contacts. Used in footer, imprint and privacy templates.
		</p>
	</header>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-8 flex flex-col gap-5">
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Business
				</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<Input label="Business name" bind:value={businessName} />
					<Input
						label="Registration number"
						placeholder="556xxxxxxx"
						bind:value={registrationNr}
					/>
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-secondary">Country</span>
						<Select options={countryOptions} bind:value={country} />
					</div>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Contacts
				</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<Input
						label="Contact email"
						type="email"
						placeholder="hello@example.com"
						bind:value={contactEmail}
					/>
					<Input
						label="Contact phone"
						placeholder="+46 8 123 45 67"
						bind:value={contactPhone}
					/>
					<Input
						label="Privacy email"
						type="email"
						placeholder="privacy@example.com"
						bind:value={privacyEmail}
					/>
					<Input
						label="Security email"
						type="email"
						placeholder="security@example.com"
						bind:value={securityEmail}
					/>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Address</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<Input label="Address line 1" bind:value={addressLine1} class="sm:col-span-2" />
					<Input label="Address line 2" bind:value={addressLine2} class="sm:col-span-2" />
					<Input label="Postal code" bind:value={postalCode} />
					<Input label="City" bind:value={city} />
				</div>
			</Card>

			<div
				class="sticky bottom-0 -mx-6 flex items-center justify-between gap-3 border-t border-border-light bg-bg-surface/85 px-6 py-3 backdrop-blur"
			>
				<span class="text-[12px] text-text-muted">
					{dirty ? 'Unsaved changes.' : 'Up to date.'}
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
