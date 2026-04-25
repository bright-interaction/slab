<script lang="ts">
	// Redirect handled in +page.ts. This shell renders only if JS-based navigation
	// has not yet kicked in.
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	onMount(() => {
		void goto('/sites/new/wizard/type', { replaceState: true });
	});
</script>

<div class="flex min-h-[40vh] items-center justify-center text-[13px] text-text-muted">
	Loading wizard...
</div>
