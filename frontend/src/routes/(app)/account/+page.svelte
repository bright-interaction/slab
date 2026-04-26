<script lang="ts">
	import { ArrowLeft, Check, Copy, Sun, Moon } from 'lucide-svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { ApiError } from '$lib/api/client';
	import * as authApi from '$lib/api/auth';
	import { auth, setUser } from '$lib/stores/auth.svelte';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { theme, toggleTheme } from '$lib/stores/theme.svelte';

	const user = $derived(auth.value);
	const isAdmin = $derived(user?.role === 'admin');

	let name = $state(user?.name ?? '');
	let savingProfile = $state(false);

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let savingPassword = $state(false);

	let signingOutAll = $state(false);
	let copiedId = $state(false);

	function formatDate(ts: string | undefined): string {
		if (!ts) return '-';
		const d = new Date(ts);
		if (Number.isNaN(d.getTime())) return ts;
		return d.toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	async function saveProfile(e: SubmitEvent) {
		e.preventDefault();
		if (savingProfile) return;
		savingProfile = true;
		try {
			const res = await authApi.updateProfile(name.trim());
			setUser(res.user);
			toast.success('Profile updated.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to update profile.');
		} finally {
			savingProfile = false;
		}
	}

	async function savePassword(e: SubmitEvent) {
		e.preventDefault();
		if (savingPassword) return;
		if (newPassword.length < 8) {
			toast.error('New password must be at least 8 characters.');
			return;
		}
		if (newPassword !== confirmPassword) {
			toast.error('New password and confirmation do not match.');
			return;
		}
		savingPassword = true;
		try {
			await authApi.changePassword(currentPassword, newPassword);
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
			toast.success('Password updated.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to update password.');
		} finally {
			savingPassword = false;
		}
	}

	async function signOutEverywhere() {
		if (signingOutAll) return;
		const ok = await confirm({
			title: 'Sign out everywhere else?',
			message:
				'Every other browser and device will be signed out immediately. You will stay signed in here.',
			confirmText: 'Sign out everywhere',
			variant: 'danger'
		});
		if (!ok) return;
		signingOutAll = true;
		try {
			await authApi.signOutEverywhere();
			toast.success('All other sessions signed out.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to sign out other sessions.');
		} finally {
			signingOutAll = false;
		}
	}

	async function copyId() {
		if (!user?.id) return;
		try {
			await navigator.clipboard.writeText(user.id);
			copiedId = true;
			setTimeout(() => (copiedId = false), 1500);
		} catch {
			toast.error('Could not copy. Select the ID and copy manually.');
		}
	}
</script>

<div class="mx-auto max-w-5xl px-6 py-8">
	<a
		href="/"
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Dashboard
	</a>
	<header class="mt-3 flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Account settings
		</h1>
		<p class="text-[13px] text-text-secondary">
			Your profile and login. Workspace-level settings live under each site.
		</p>
	</header>

	<div class="mt-8 flex flex-col gap-5">
		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Profile</h2>
			<form class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2" onsubmit={saveProfile}>
				<Input label="Email" value={user?.email ?? ''} disabled hint="Email cannot be changed yet." />
				<div class="flex flex-col gap-1.5">
					<span class="text-[12px] font-medium text-text-secondary">Role</span>
					<div
						class="rounded-lg border border-border-light bg-bg-elevated px-3 py-2 text-[13px] text-text-primary"
					>
						{user?.role ?? 'unknown'}
					</div>
				</div>
				<div class="sm:col-span-2">
					<Input label="Display name" bind:value={name} placeholder="Your name" />
				</div>
				<div class="sm:col-span-2 flex items-center justify-end gap-2">
					<Button type="submit" variant="primary" loading={savingProfile} disabled={savingProfile}>
						Save profile
					</Button>
				</div>
			</form>

			<dl
				class="mt-6 grid grid-cols-1 gap-3 border-t border-border-light pt-5 text-[12px] sm:grid-cols-3"
			>
				<div class="flex flex-col gap-0.5">
					<dt class="text-text-muted">Member since</dt>
					<dd class="text-text-primary">{formatDate(user?.created_at)}</dd>
				</div>
				<div class="flex flex-col gap-0.5">
					<dt class="text-text-muted">Last updated</dt>
					<dd class="text-text-primary">{formatDate(user?.updated_at)}</dd>
				</div>
				<div class="flex min-w-0 flex-col gap-0.5">
					<dt class="text-text-muted">Account ID</dt>
					<dd class="flex items-center gap-1.5">
						<code class="truncate font-mono text-[11px] text-text-secondary">{user?.id ?? '-'}</code>
						{#if user?.id}
							<button
								type="button"
								class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary"
								onclick={copyId}
								aria-label="Copy account ID"
							>
								{#if copiedId}
									<Check size={12} strokeWidth={2} />
								{:else}
									<Copy size={12} strokeWidth={1.75} />
								{/if}
							</button>
						{/if}
					</dd>
				</div>
			</dl>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Password</h2>
			<form class="mt-4 grid grid-cols-1 gap-4" onsubmit={savePassword}>
				<Input
					label="Current password"
					type="password"
					autocomplete="current-password"
					required
					bind:value={currentPassword}
				/>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<Input
						label="New password"
						type="password"
						autocomplete="new-password"
						required
						bind:value={newPassword}
						hint="Minimum 8 characters."
					/>
					<Input
						label="Confirm new password"
						type="password"
						autocomplete="new-password"
						required
						bind:value={confirmPassword}
					/>
				</div>
				<div class="flex items-center justify-end gap-2">
					<Button type="submit" variant="primary" loading={savingPassword} disabled={savingPassword}>
						Change password
					</Button>
				</div>
				<p class="text-[11px] text-text-muted">
					Changing your password signs out every other session. You'll stay signed in here.
				</p>
			</form>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Sessions</h2>
			<div class="mt-4 flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
				<div class="flex-1">
					<p class="text-[13px] text-text-primary">
						Signed in here as <span class="font-medium">{user?.email ?? ''}</span>.
					</p>
					<p class="mt-1 text-[12px] text-text-muted">
						Sign out of every other browser and device. You'll stay signed in here.
					</p>
				</div>
				<Button
					variant="secondary"
					onclick={signOutEverywhere}
					loading={signingOutAll}
					disabled={signingOutAll}
				>
					Sign out everywhere else
				</Button>
			</div>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Preferences</h2>
			<div class="mt-4 flex items-center justify-between gap-4">
				<div>
					<p class="text-[13px] font-medium text-text-primary">Theme</p>
					<p class="mt-0.5 text-[12px] text-text-muted">
						Currently using {theme.value === 'dark' ? 'dark' : 'light'} mode. Saved on this device.
					</p>
				</div>
				<button
					type="button"
					class="inline-flex h-9 items-center gap-2 rounded-lg border border-border-light bg-bg-elevated px-3 text-[12px] text-text-primary transition-colors hover:bg-bg-hover"
					onclick={toggleTheme}
					aria-label={`Switch to ${theme.value === 'dark' ? 'light' : 'dark'} mode`}
				>
					{#if theme.value === 'dark'}
						<Sun size={14} strokeWidth={1.75} />
						Switch to light
					{:else}
						<Moon size={14} strokeWidth={1.75} />
						Switch to dark
					{/if}
				</button>
			</div>
		</Card>

		{#if isAdmin}
			<Card padding="md">
				<div class="flex items-center justify-between gap-4">
					<div>
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							Workspace members
						</h2>
						<p class="mt-1 text-[13px] text-text-secondary">Invite teammates and manage roles.</p>
					</div>
					<Button variant="secondary" onclick={() => (window.location.href = '/members')}>
						Manage members
					</Button>
				</div>
			</Card>
		{/if}
	</div>
</div>
