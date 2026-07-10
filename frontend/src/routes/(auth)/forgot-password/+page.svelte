<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { ApiError } from '$lib/api/client';
	import * as authApi from '$lib/api/auth';

	let email = $state('');
	let loading = $state(false);
	// We deliberately render the same success state regardless of
	// whether the email matched a real user. Account enumeration is
	// defeated server-side (bare 200) and we mirror that here so the
	// UI doesn't leak via state divergence.
	let submitted = $state(false);
	let error = $state('');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (loading) return;
		error = '';
		loading = true;
		try {
			await authApi.forgotPassword(email);
			submitted = true;
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Could not request reset. Try again.';
		} finally {
			loading = false;
		}
	}
</script>

<div class="rounded-2xl border border-border-light bg-bg-surface p-8">
	<div class="mb-6 space-y-1">
		<h1 class="font-display text-2xl font-extralight tracking-tight text-text-primary">Forgot password</h1>
		<p class="text-[13px] text-text-secondary">
			Enter your email and we'll send a reset link valid for 30 minutes.
		</p>
	</div>

	{#if submitted}
		<div class="rounded-xl border border-border-light bg-bg-elevated p-4">
			<p class="text-[13px] text-text-primary">
				If <span class="font-mono">{email}</span> is registered, a reset link is on its way.
			</p>
			<p class="mt-2 text-[12px] text-text-muted">
				Don't see it? Check spam, then ask an operator to read the application log
				for the reset link (slab logs them when no mail integration is wired).
			</p>
			<a
				href="/login"
				class="mt-4 inline-block text-[12px] text-accent underline-offset-2 hover:underline"
			>
				Back to sign in
			</a>
		</div>
	{:else}
		<form class="space-y-4" onsubmit={submit}>
			<Input
				label="Email"
				type="email"
				autocomplete="email"
				required
				bind:value={email}
				disabled={loading}
				placeholder="you@company.com"
			/>

			{#if error}
				<p class="text-[12px] text-danger" role="alert">{error}</p>
			{/if}

			<Button type="submit" variant="primary" {loading} disabled={loading} class="w-full">
				{loading ? 'Sending...' : 'Send reset link'}
			</Button>

			<p class="text-center text-[12px] text-text-muted">
				<a href="/login" class="text-accent underline-offset-2 hover:underline">
					Back to sign in
				</a>
			</p>
		</form>
	{/if}
</div>
