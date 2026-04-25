let isOpen = $state<boolean>(false);

export const palette = {
	get open(): boolean {
		return isOpen;
	}
};

export function openPalette(): void {
	isOpen = true;
}

export function closePalette(): void {
	isOpen = false;
}

export function togglePalette(): void {
	isOpen = !isOpen;
}
