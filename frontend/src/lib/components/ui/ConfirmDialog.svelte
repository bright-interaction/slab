<script lang="ts" module>
	type ConfirmOptions = {
		title: string;
		message: string;
		confirmText?: string;
		cancelText?: string;
		variant?: 'danger' | 'primary';
	};

	type State = ConfirmOptions & {
		open: boolean;
		resolve: (v: boolean) => void;
	};

	let state = $state<State | null>(null);

	export function confirm(opts: ConfirmOptions): Promise<boolean> {
		return new Promise((resolve) => {
			state = { ...opts, open: true, resolve };
		});
	}

	export function _state() {
		return state;
	}

	export function _close(result: boolean) {
		if (state) {
			state.resolve(result);
			state = null;
		}
	}
</script>

<script lang="ts">
	import Dialog from './Dialog.svelte';
	import Button from './Button.svelte';

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

	const ambient = $derived(state);
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
			_close(true);
		}
	}

	function handleCancel() {
		if (inline) {
			oncancel?.();
			open = false;
		} else {
			_close(false);
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
