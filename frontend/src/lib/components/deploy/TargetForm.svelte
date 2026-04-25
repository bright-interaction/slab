<script lang="ts">
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import type {
		DeployKind,
		DeployTarget,
		DeployTargetDockyardConfig,
		DeployTargetLocalConfig,
		DeployTargetRsyncConfig
	} from '$lib/api/types';

	let {
		value = $bindable(),
		mode,
		showDefaultToggle = true,
		errors = {}
	}: {
		value: Partial<DeployTarget>;
		mode: 'create' | 'edit';
		showDefaultToggle?: boolean;
		errors?: Record<string, string>;
	} = $props();

	const kindOptions = [
		{ value: 'local', label: 'Local' },
		{ value: 'rsync', label: 'Rsync' },
		{ value: 'dockyard', label: 'Dockyard' }
	];

	function asKind(v: string | undefined): DeployKind {
		return v === 'rsync' || v === 'dockyard' ? v : 'local';
	}

	const initialKind = asKind(value.kind);
	const initialConfig = (value.config ?? {}) as Record<string, unknown>;
	const initialLocal = initialKind === 'local' ? initialConfig : {};
	const initialRsync = initialKind === 'rsync' ? initialConfig : {};
	const initialDockyard = initialKind === 'dockyard' ? initialConfig : {};

	let nameLocal = $state<string>(value.name ?? '');
	let kindLocal = $state<string>(initialKind);
	let isDefaultLocal = $state<boolean>(Boolean(value.is_default));

	// Local config fields. Strings only, with port handled as string for Input.
	let localPath = $state<string>(String(initialLocal.path ?? ''));
	let localPublicURL = $state<string>(String(initialLocal.public_url ?? ''));

	let rsyncHost = $state<string>(String(initialRsync.host ?? ''));
	let rsyncUser = $state<string>(String(initialRsync.user ?? ''));
	let rsyncPort = $state<string>(
		initialRsync.port !== undefined ? String(initialRsync.port) : '22'
	);
	let rsyncPath = $state<string>(String(initialRsync.path ?? ''));
	let rsyncPublicURL = $state<string>(String(initialRsync.public_url ?? ''));
	let rsyncKey = $state<string>(String(initialRsync.private_key_pem ?? ''));

	let dyURL = $state<string>(String(initialDockyard.dockyard_url ?? ''));
	let dyAPIKey = $state<string>(String(initialDockyard.api_key ?? ''));
	let dyServerID = $state<string>(String(initialDockyard.server_id ?? ''));
	let dyDomain = $state<string>(String(initialDockyard.domain ?? ''));
	let dyHost = $state<string>(String(initialDockyard.host ?? ''));
	let dyUser = $state<string>(String(initialDockyard.user ?? ''));
	let dyPort = $state<string>(
		initialDockyard.port !== undefined ? String(initialDockyard.port) : '22'
	);
	let dyHostPath = $state<string>(String(initialDockyard.host_path ?? ''));
	let dyKey = $state<string>(String(initialDockyard.private_key_pem ?? ''));

	function buildLocalConfig(): DeployTargetLocalConfig {
		const out: DeployTargetLocalConfig = {};
		if (localPath !== '') out.path = localPath;
		if (localPublicURL !== '') out.public_url = localPublicURL;
		return out;
	}

	function buildRsyncConfig(): DeployTargetRsyncConfig {
		const out: DeployTargetRsyncConfig = {};
		if (rsyncHost !== '') out.host = rsyncHost;
		if (rsyncUser !== '') out.user = rsyncUser;
		const portN = Number(rsyncPort);
		out.port = Number.isFinite(portN) && portN > 0 ? portN : 22;
		if (rsyncPath !== '') out.path = rsyncPath;
		if (rsyncPublicURL !== '') out.public_url = rsyncPublicURL;
		if (rsyncKey !== '') out.private_key_pem = rsyncKey;
		return out;
	}

	function buildDockyardConfig(): DeployTargetDockyardConfig {
		const out: DeployTargetDockyardConfig = {};
		if (dyURL !== '') out.dockyard_url = dyURL;
		if (dyAPIKey !== '') out.api_key = dyAPIKey;
		if (dyServerID !== '') out.server_id = dyServerID;
		if (dyDomain !== '') out.domain = dyDomain;
		if (dyHost !== '') out.host = dyHost;
		if (dyUser !== '') out.user = dyUser;
		const portN = Number(dyPort);
		out.port = Number.isFinite(portN) && portN > 0 ? portN : 22;
		if (dyHostPath !== '') out.host_path = dyHostPath;
		if (dyKey !== '') out.private_key_pem = dyKey;
		return out;
	}

	// Sync everything into the bindable `value`.
	$effect(() => {
		const kind = asKind(kindLocal);
		let config: Record<string, unknown>;
		if (kind === 'local') config = { ...buildLocalConfig() } as Record<string, unknown>;
		else if (kind === 'rsync') config = { ...buildRsyncConfig() } as Record<string, unknown>;
		else config = { ...buildDockyardConfig() } as Record<string, unknown>;

		value = {
			...value,
			name: nameLocal,
			kind,
			config,
			is_default: isDefaultLocal
		};
	});

	const currentKind = $derived(asKind(kindLocal));
</script>

<div class="flex flex-col gap-4">
	<Input label="Name" placeholder="Production" bind:value={nameLocal} error={errors.name} required />

	<div class="flex flex-col gap-1.5">
		<span class="text-[12px] font-medium text-text-secondary">Kind</span>
		<Select
			options={kindOptions}
			bind:value={kindLocal}
			placeholder="Select a kind"
			disabled={mode === 'edit'}
		/>
		{#if mode === 'edit'}
			<p class="text-[12px] text-text-muted">Kind cannot be changed after creation.</p>
		{:else}
			<p class="text-[12px] text-text-muted">
				Local copies the build to a path on the server. Rsync pushes via SSH. Dockyard publishes to
				a Dockyard-managed container.
			</p>
		{/if}
	</div>

	{#if currentKind === 'local'}
		<Input
			label="Path"
			placeholder="/var/www/mysite"
			bind:value={localPath}
			hint="Absolute path on the server"
			error={errors['config.path']}
			required
		/>
		<Input
			label="Public URL"
			placeholder="https://example.com"
			bind:value={localPublicURL}
			hint="Optional. Shown after deploy succeeds."
		/>
	{/if}

	{#if currentKind === 'rsync'}
		<Input
			label="Host"
			placeholder="server.example.com"
			bind:value={rsyncHost}
			error={errors['config.host']}
			required
		/>
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
			<div class="sm:col-span-2">
				<Input
					label="User"
					placeholder="deploy"
					bind:value={rsyncUser}
					error={errors['config.user']}
					required
				/>
			</div>
			<Input label="Port" type="number" placeholder="22" bind:value={rsyncPort} min={1} max={65535} />
		</div>
		<Input
			label="Path"
			placeholder="/var/www/mysite"
			bind:value={rsyncPath}
			hint="Remote absolute path"
			error={errors['config.path']}
			required
		/>
		<Input label="Public URL" placeholder="https://example.com" bind:value={rsyncPublicURL} />
		<Textarea
			label="Private key (PEM)"
			rows={6}
			class="font-mono"
			placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
			bind:value={rsyncKey}
			hint="PEM-formatted SSH private key. Will be stored encrypted at rest in v1.1."
			error={errors['config.private_key_pem']}
			required
		/>
	{/if}

	{#if currentKind === 'dockyard'}
		<Input
			label="Dockyard URL"
			placeholder="https://dockyard.example.com"
			bind:value={dyURL}
			error={errors['config.dockyard_url']}
			required
		/>
		<Input
			label="API key"
			type="password"
			placeholder="dy_live_..."
			bind:value={dyAPIKey}
			error={errors['config.api_key']}
			required
		/>
		<Input
			label="Server ID"
			placeholder="srv_abc123"
			bind:value={dyServerID}
			error={errors['config.server_id']}
			required
		/>
		<Input
			label="Domain"
			placeholder="mysite.slab.example.com"
			bind:value={dyDomain}
			error={errors['config.domain']}
			required
		/>
		<Input label="Host" placeholder="server.example.com" bind:value={dyHost} />
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
			<div class="sm:col-span-2">
				<Input label="User" placeholder="deploy" bind:value={dyUser} />
			</div>
			<Input label="Port" type="number" placeholder="22" bind:value={dyPort} min={1} max={65535} />
		</div>
		<Input label="Host path" placeholder="/srv/sites/mysite" bind:value={dyHostPath} />
		<Textarea
			label="Private key (PEM)"
			rows={6}
			class="font-mono"
			placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
			bind:value={dyKey}
			hint="PEM-formatted SSH private key. Will be stored encrypted at rest in v1.1."
		/>
	{/if}

	{#if showDefaultToggle}
		<div
			class="flex items-center justify-between rounded-lg border border-border-light bg-bg-elevated px-3 py-2"
		>
			<div class="flex flex-col">
				<span class="text-[13px] font-medium text-text-primary">Default target</span>
				<span class="text-[12px] text-text-muted">
					Used by the Deploy button on the build page.
				</span>
			</div>
			<Switch bind:checked={isDefaultLocal} />
		</div>
	{/if}
</div>
