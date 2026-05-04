<script lang="ts">
	import { goto } from '$app/navigation';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import { ApiError } from '$lib/api/client';
	import * as authApi from '$lib/api/auth';
	import { setUser } from '$lib/stores/auth.svelte';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (loading) return;
		error = '';
		loading = true;
		try {
			const res = await authApi.login(email, password);
			setUser(res.user);
			await goto('/');
		} catch (err) {
			if (err instanceof ApiError) {
				error = err.message;
			} else {
				error = 'Sign in failed. Try again.';
			}
		} finally {
			loading = false;
		}
	}
</script>

<div class="rounded-2xl border border-border-light bg-bg-surface p-8">
	<div class="mb-6 space-y-1">
		<h1 class="font-display text-2xl font-extralight tracking-tight text-text-primary">Admin sign in</h1>
		<p class="text-[13px] text-text-secondary">Atomicsite is invite-only. Ask your workspace admin for an invite link.</p>
	</div>

	<form class="space-y-4" onsubmit={handleSubmit}>
		<Input
			label="Email"
			type="email"
			autocomplete="email"
			required
			bind:value={email}
			disabled={loading}
			placeholder="you@company.com"
		/>
		<Input
			label="Password"
			type="password"
			autocomplete="current-password"
			required
			bind:value={password}
			disabled={loading}
		/>

		{#if error}
			<p class="text-[12px] text-danger" role="alert">{error}</p>
		{/if}

		<Button type="submit" variant="primary" {loading} disabled={loading} class="w-full">
			{loading ? 'Signing in.' : 'Sign in'}
		</Button>

		<p class="text-center text-[12px] text-text-muted">
			<a href="/forgot-password" class="text-accent underline-offset-2 hover:underline">
				Forgot password?
			</a>
		</p>
	</form>
</div>
