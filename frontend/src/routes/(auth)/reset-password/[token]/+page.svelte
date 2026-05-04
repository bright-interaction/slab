<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { ApiError } from '$lib/api/client';
	import * as authApi from '$lib/api/auth';

	const token = $derived(page.params.token ?? '');

	let maskedEmail = $state('');
	let loadingInfo = $state(true);
	let infoError = $state('');

	let password = $state('');
	let confirm = $state('');
	let submitting = $state(false);
	let submitError = $state('');
	let done = $state(false);

	$effect(() => {
		void loadInfo();
	});

	async function loadInfo() {
		if (!token) {
			infoError = 'Reset link is missing the token.';
			loadingInfo = false;
			return;
		}
		try {
			const info = await authApi.resetPasswordInfo(token);
			maskedEmail = info.email;
		} catch (err) {
			infoError =
				err instanceof ApiError
					? err.message
					: 'Reset link is invalid or has expired.';
		} finally {
			loadingInfo = false;
		}
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (submitting) return;
		submitError = '';
		if (password.length < 12) {
			submitError = 'Password must be at least 12 characters.';
			return;
		}
		if (password !== confirm) {
			submitError = 'Passwords do not match.';
			return;
		}
		submitting = true;
		try {
			await authApi.resetPassword(token, password);
			done = true;
			// Server has invalidated every other session via token_version
			// bump; route them to /login to re-authenticate with the new
			// password. Slight delay so the success state is visible.
			setTimeout(() => {
				void goto('/login');
			}, 1500);
		} catch (err) {
			submitError =
				err instanceof ApiError
					? err.message
					: 'Could not reset password.';
		} finally {
			submitting = false;
		}
	}
</script>

<div class="rounded-2xl border border-border-light bg-bg-surface p-8">
	<div class="mb-6 space-y-1">
		<h1 class="font-display text-2xl font-extralight tracking-tight text-text-primary">Reset password</h1>
		{#if maskedEmail && !infoError}
			<p class="text-[13px] text-text-secondary">
				For <span class="font-mono">{maskedEmail}</span>
			</p>
		{/if}
	</div>

	{#if loadingInfo}
		<p class="text-[13px] text-text-muted">Checking reset link...</p>
	{:else if infoError}
		<div class="rounded-xl border border-danger/30 bg-danger/5 p-4">
			<p class="text-[13px] text-danger">{infoError}</p>
			<a
				href="/forgot-password"
				class="mt-3 inline-block text-[12px] text-accent underline-offset-2 hover:underline"
			>
				Request a new link
			</a>
		</div>
	{:else if done}
		<div class="rounded-xl border border-border-light bg-bg-elevated p-4">
			<p class="text-[13px] text-text-primary">
				Password updated. Sending you to sign in.
			</p>
		</div>
	{:else}
		<form class="space-y-4" onsubmit={submit}>
			<Input
				label="New password"
				type="password"
				autocomplete="new-password"
				required
				bind:value={password}
				disabled={submitting}
				placeholder="At least 12 characters"
			/>
			<Input
				label="Confirm new password"
				type="password"
				autocomplete="new-password"
				required
				bind:value={confirm}
				disabled={submitting}
			/>

			{#if submitError}
				<p class="text-[12px] text-danger" role="alert">{submitError}</p>
			{/if}

			<Button type="submit" variant="primary" loading={submitting} disabled={submitting} class="w-full">
				{submitting ? 'Updating...' : 'Update password'}
			</Button>
		</form>
	{/if}
</div>
