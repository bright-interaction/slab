// Module-level state for the ambient ConfirmDialog. Lives in a .svelte.ts
// module (not a .svelte component) so the state is guaranteed to be a single
// shared instance regardless of how many places import the trigger.

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

export function getConfirmState(): State | null {
	return state;
}

export function closeConfirm(result: boolean): void {
	if (state) {
		state.resolve(result);
		state = null;
	}
}
