<script lang="ts" module>
	// Re-export from the shared store so existing imports keep working:
	//   import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	export { confirm } from '$lib/stores/confirm.svelte';
</script>

<script lang="ts">
	import Dialog from './Dialog.svelte';
	import Button from './Button.svelte';
	import { getConfirmState, closeConfirm } from '$lib/stores/confirm.svelte';

	let {
		open = $bindable(false),
		title,
		message,
		confirmText = 'Confirm',
		cancelText = 'Cancel',
		variant = 'primary',
		onconfirm,
		oncancel
	}: {
		open?: boolean;
		title?: string;
		message?: string;
		confirmText?: string;
		cancelText?: string;
		variant?: 'danger' | 'primary';
		onconfirm?: () => void;
		oncancel?: () => void;
	} = $props();

	const inline = $derived(title !== undefined || message !== undefined);

	const ambient = $derived(getConfirmState());
	const showAmbient = $derived(!inline && ambient !== null && ambient.open);

	const finalTitle = $derived(inline ? title : (ambient?.title ?? ''));
	const finalMessage = $derived(inline ? message : (ambient?.message ?? ''));
	const finalConfirmText = $derived(
		inline ? confirmText : (ambient?.confirmText ?? 'Confirm')
	);
	const finalCancelText = $derived(inline ? cancelText : (ambient?.cancelText ?? 'Cancel'));
	const finalVariant = $derived(inline ? variant : (ambient?.variant ?? 'primary'));

	let isOpen = $derived(inline ? open : showAmbient);

	function handleConfirm() {
		if (inline) {
			onconfirm?.();
			open = false;
		} else {
			closeConfirm(true);
		}
	}

	function handleCancel() {
		if (inline) {
			oncancel?.();
			open = false;
		} else {
			closeConfirm(false);
		}
	}
</script>

<Dialog
	open={isOpen}
	onOpenChange={(v: boolean) => {
		if (!v) handleCancel();
	}}
	title={finalTitle}
	description={finalMessage}
	size="sm"
>
	{#snippet footer()}
		<Button variant="secondary" onclick={handleCancel}>{finalCancelText}</Button>
		<Button variant={finalVariant === 'danger' ? 'danger' : 'primary'} onclick={handleConfirm}>
			{finalConfirmText}
		</Button>
	{/snippet}
</Dialog>
