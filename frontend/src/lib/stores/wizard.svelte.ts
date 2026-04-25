import type {
	SeedSiteBranding,
	SeedSiteRequest,
	SeedSiteResponse,
	SeedSiteSilo,
	SeedSiteStructure,
	SiteType,
	StructureType
} from '$lib/api/sites';

export type WizardStep = 'type' | 'kit' | 'info' | 'structure' | 'branding' | 'confirm';

export const WIZARD_STEPS: WizardStep[] = [
	'type',
	'kit',
	'info',
	'structure',
	'branding',
	'confirm'
];

export interface WizardInfo {
	name: string;
	slug: string;
	domain: string;
	lang: string;
	business_name: string;
	contact_email: string;
	country: string;
}

export interface WizardState {
	step: WizardStep;
	type: SiteType | null;
	starterKit: string | null;
	info: WizardInfo;
	structure: SeedSiteStructure;
	silos: SeedSiteSilo[];
	branding: SeedSiteBranding;
}

const STORAGE_KEY = 'atomicsite_wizard_state';

function defaultState(): WizardState {
	return {
		step: 'type',
		type: null,
		starterKit: null,
		info: {
			name: '',
			slug: '',
			domain: '',
			lang: 'sv',
			business_name: '',
			contact_email: '',
			country: 'SE'
		},
		structure: {
			structure_type: 'soft-silo',
			max_depth: 3
		},
		silos: [],
		branding: {
			primary_color: '#0e7490',
			secondary_color: '#52525b',
			bg_color: '#fafaf9',
			text_color: '#18181b',
			font_heading: 'Geist',
			font_body: 'Inter'
		}
	};
}

function readInitial(): WizardState {
	if (typeof window === 'undefined') return defaultState();
	try {
		const raw = window.sessionStorage.getItem(STORAGE_KEY);
		if (!raw) return defaultState();
		const parsed = JSON.parse(raw) as Partial<WizardState>;
		const base = defaultState();
		return {
			step: (parsed.step as WizardStep) ?? base.step,
			type: (parsed.type as SiteType | null) ?? base.type,
			starterKit:
				typeof parsed.starterKit === 'string' || parsed.starterKit === null
					? (parsed.starterKit as string | null)
					: base.starterKit,
			info: { ...base.info, ...(parsed.info ?? {}) },
			structure: { ...base.structure, ...(parsed.structure ?? {}) },
			silos: Array.isArray(parsed.silos) ? parsed.silos : base.silos,
			branding: { ...base.branding, ...(parsed.branding ?? {}) }
		};
	} catch {
		return defaultState();
	}
}

let current = $state<WizardState>(readInitial());

function persist(): void {
	if (typeof window === 'undefined') return;
	try {
		window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(current));
	} catch {
		// Ignore quota or privacy-mode failures.
	}
}

export const wizard = {
	get value(): WizardState {
		return current;
	}
};

export function setStep(step: WizardStep): void {
	current = { ...current, step };
	persist();
}

export function setType(type: SiteType): void {
	current = { ...current, type };
	persist();
}

export function setStarterKit(kitID: string | null): void {
	current = { ...current, starterKit: kitID };
	persist();
}

export function updateInfo(partial: Partial<WizardInfo>): void {
	current = { ...current, info: { ...current.info, ...partial } };
	persist();
}

export function updateStructure(partial: Partial<SeedSiteStructure>): void {
	current = { ...current, structure: { ...current.structure, ...partial } };
	persist();
}

export function updateSilos(silos: SeedSiteSilo[]): void {
	current = { ...current, silos };
	persist();
}

export function updateBranding(partial: Partial<SeedSiteBranding>): void {
	current = { ...current, branding: { ...current.branding, ...partial } };
	persist();
}

export function reset(): void {
	current = defaultState();
	if (typeof window !== 'undefined') {
		try {
			window.sessionStorage.removeItem(STORAGE_KEY);
		} catch {
			// Ignore.
		}
	}
}

export function deriveSlug(name: string): string {
	return name
		.toLowerCase()
		.replace(/\s+/g, '-')
		.replace(/[^a-z0-9-]/g, '')
		.replace(/-+/g, '-')
		.replace(/^-|-$/g, '')
		.slice(0, 50);
}

export interface SeedApiClient {
	seed: (input: SeedSiteRequest) => Promise<SeedSiteResponse>;
}

export function buildSeedRequest(state: WizardState): SeedSiteRequest {
	const info = state.info;
	const branding = state.branding;
	const structure = state.structure;
	const isOnePager =
		state.type === 'one-pager' || structure.structure_type === 'one-pager';
	const req: SeedSiteRequest = {
		type: (state.type ?? 'b2b') as SiteType,
		info: {
			name: info.name.trim(),
			slug: info.slug.trim(),
			domain: info.domain.trim() || undefined,
			lang: info.lang,
			business_name: info.business_name.trim(),
			contact_email: info.contact_email.trim(),
			country: info.country.trim()
		},
		structure: {
			structure_type: (isOnePager
				? 'one-pager'
				: structure.structure_type) as StructureType,
			max_depth: isOnePager ? 1 : structure.max_depth
		},
		silos: isOnePager ? [] : state.silos,
		branding: { ...branding }
	};
	if (state.starterKit) {
		req.starter_kit = state.starterKit;
	}
	return req;
}

export async function submit(api: SeedApiClient): Promise<SeedSiteResponse> {
	const payload = buildSeedRequest(current);
	return api.seed(payload);
}
