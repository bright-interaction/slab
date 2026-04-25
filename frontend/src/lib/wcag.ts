// WCAG contrast helpers. Pure functions, no DOM dependencies.

export interface Rgb {
	r: number;
	g: number;
	b: number;
}

const HEX_RE = /^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/;

export function hexToRgb(hex: string): Rgb | null {
	const m = HEX_RE.exec(hex.trim());
	if (!m || !m[1]) return null;
	let body: string = m[1];
	if (body.length === 3) {
		body = body
			.split('')
			.map((c) => c + c)
			.join('');
	}
	const num = parseInt(body, 16);
	return {
		r: (num >> 16) & 0xff,
		g: (num >> 8) & 0xff,
		b: num & 0xff
	};
}

function clamp(n: number, min: number, max: number): number {
	return Math.max(min, Math.min(max, n));
}

export function rgbToHex(r: number, g: number, b: number): string {
	const toHex = (n: number): string => {
		const v = Math.round(clamp(n, 0, 255));
		return v.toString(16).padStart(2, '0');
	};
	return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
}

function channelLuminance(c: number): number {
	const s = c / 255;
	return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
}

export function relativeLuminance(r: number, g: number, b: number): number {
	const R = channelLuminance(r);
	const G = channelLuminance(g);
	const B = channelLuminance(b);
	return 0.2126 * R + 0.7152 * G + 0.0722 * B;
}

export function contrastRatio(fgHex: string, bgHex: string): number {
	const fg = hexToRgb(fgHex);
	const bg = hexToRgb(bgHex);
	if (!fg || !bg) return 1;
	const Lfg = relativeLuminance(fg.r, fg.g, fg.b);
	const Lbg = relativeLuminance(bg.r, bg.g, bg.b);
	const lighter = Math.max(Lfg, Lbg);
	const darker = Math.min(Lfg, Lbg);
	return (lighter + 0.05) / (darker + 0.05);
}

export function passesAA(ratio: number, largeText = false): boolean {
	return ratio >= (largeText ? 3 : 4.5);
}

export function passesAAA(ratio: number, largeText = false): boolean {
	return ratio >= (largeText ? 4.5 : 7);
}

interface Hsl {
	h: number;
	s: number;
	l: number;
}

function rgbToHsl({ r, g, b }: Rgb): Hsl {
	const rN = r / 255;
	const gN = g / 255;
	const bN = b / 255;
	const max = Math.max(rN, gN, bN);
	const min = Math.min(rN, gN, bN);
	const l = (max + min) / 2;
	let h = 0;
	let s = 0;
	if (max !== min) {
		const d = max - min;
		s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
		switch (max) {
			case rN:
				h = (gN - bN) / d + (gN < bN ? 6 : 0);
				break;
			case gN:
				h = (bN - rN) / d + 2;
				break;
			case bN:
				h = (rN - gN) / d + 4;
				break;
		}
		h /= 6;
	}
	return { h, s, l };
}

function hslToRgb({ h, s, l }: Hsl): Rgb {
	let r: number;
	let g: number;
	let b: number;
	if (s === 0) {
		r = g = b = l;
	} else {
		const hue2rgb = (p: number, q: number, t: number): number => {
			if (t < 0) t += 1;
			if (t > 1) t -= 1;
			if (t < 1 / 6) return p + (q - p) * 6 * t;
			if (t < 1 / 2) return q;
			if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
			return p;
		};
		const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
		const p = 2 * l - q;
		r = hue2rgb(p, q, h + 1 / 3);
		g = hue2rgb(p, q, h);
		b = hue2rgb(p, q, h - 1 / 3);
	}
	return { r: r * 255, g: g * 255, b: b * 255 };
}

function shiftLightness(hex: string, delta: number): string {
	const rgb = hexToRgb(hex);
	if (!rgb) return hex;
	const hsl = rgbToHsl(rgb);
	hsl.l = clamp(hsl.l + delta, 0, 1);
	const shifted = hslToRgb(hsl);
	return rgbToHex(shifted.r, shifted.g, shifted.b);
}

export function darken(hex: string, percent: number): string {
	return shiftLightness(hex, -(percent / 100));
}

export function lighten(hex: string, percent: number): string {
	return shiftLightness(hex, percent / 100);
}

export interface SuggestResult {
	hex: string;
	percent: number;
}

// suggestDarken returns an adjusted version of fgHex that meets the target ratio
// against bgHex. It searches in the direction (darken or lighten) most likely to
// improve contrast given the background luminance.
export function suggestDarken(
	fgHex: string,
	bgHex: string,
	target: number
): SuggestResult {
	const bgRgb = hexToRgb(bgHex);
	if (!bgRgb) return { hex: fgHex, percent: 0 };
	const bgL = relativeLuminance(bgRgb.r, bgRgb.g, bgRgb.b);
	// Light bg -> darken fg. Dark bg -> lighten fg.
	const direction: 'darken' | 'lighten' = bgL > 0.5 ? 'darken' : 'lighten';
	const apply = (pct: number): string =>
		direction === 'darken' ? darken(fgHex, pct) : lighten(fgHex, pct);

	let lo = 0;
	let hi = 100;
	let best = { hex: fgHex, percent: 0 };
	const initialRatio = contrastRatio(fgHex, bgHex);
	if (initialRatio >= target) return best;

	for (let i = 0; i < 24; i++) {
		const mid = (lo + hi) / 2;
		const candidate = apply(mid);
		const ratio = contrastRatio(candidate, bgHex);
		if (ratio >= target) {
			best = { hex: candidate, percent: mid };
			hi = mid;
		} else {
			lo = mid;
		}
	}
	if (best.percent === 0) {
		// Fall back to maximum shift even if target unmet.
		best = { hex: apply(100), percent: 100 };
	}
	return best;
}
