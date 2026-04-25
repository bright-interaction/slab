import * as builds from '$lib/api/builds';
import type { BuildStatusResponse } from '$lib/api/builds';

export type BuildStatusValue = 'pending' | 'building' | 'success' | 'failed' | 'timeout';

export interface BuildStatusState {
	siteID: string;
	buildID: string;
	status: BuildStatusValue;
	build_log: string;
	pages_built: number;
	duration_ms: number;
	error: string;
	dist_dir: string;
	started_at: number;
	updated_at: number;
}

const FAST_INTERVAL_MS = 2_000;
const SLOW_INTERVAL_MS = 5_000;
const SLOW_AFTER_MS = 30_000;
const TIMEOUT_AFTER_MS = 5 * 60_000;

const states = $state<Record<string, BuildStatusState>>({});
const timers: Record<string, ReturnType<typeof setTimeout>> = {};

function isTerminal(status: BuildStatusValue): boolean {
	return status === 'success' || status === 'failed' || status === 'timeout';
}

function normalizeStatus(raw: string): BuildStatusValue {
	if (raw === 'success' || raw === 'failed' || raw === 'timeout') return raw;
	if (raw === 'pending') return 'pending';
	return 'building';
}

function applyTick(buildID: string, resp: BuildStatusResponse): void {
	const prev = states[buildID];
	if (!prev) return;
	states[buildID] = {
		...prev,
		status: normalizeStatus(resp.status),
		build_log: resp.build_log ?? prev.build_log,
		pages_built: resp.pages_built ?? prev.pages_built,
		duration_ms: resp.duration_ms ?? prev.duration_ms,
		error: resp.error ?? prev.error,
		dist_dir: resp.dist_dir ?? prev.dist_dir,
		updated_at: Date.now()
	};
}

function stopTimer(buildID: string): void {
	const t = timers[buildID];
	if (t !== undefined) {
		clearTimeout(t);
		delete timers[buildID];
	}
}

function scheduleNext(siteID: string, buildID: string): void {
	const cur = states[buildID];
	if (!cur || isTerminal(cur.status)) {
		stopTimer(buildID);
		return;
	}
	const elapsed = Date.now() - cur.started_at;
	if (elapsed >= TIMEOUT_AFTER_MS) {
		states[buildID] = { ...cur, status: 'timeout', updated_at: Date.now() };
		stopTimer(buildID);
		return;
	}
	const delay = elapsed >= SLOW_AFTER_MS ? SLOW_INTERVAL_MS : FAST_INTERVAL_MS;
	timers[buildID] = setTimeout(() => {
		void tick(siteID, buildID);
	}, delay);
}

async function tick(siteID: string, buildID: string): Promise<void> {
	const cur = states[buildID];
	if (!cur || isTerminal(cur.status)) {
		stopTimer(buildID);
		return;
	}
	try {
		const resp = await builds.getBuildStatus(siteID, buildID);
		applyTick(buildID, resp);
	} catch {
		// Swallow transient errors; keep polling until timeout.
	}
	scheduleNext(siteID, buildID);
}

export function pollBuild(siteID: string, buildID: string): () => void {
	const now = Date.now();
	if (!states[buildID]) {
		states[buildID] = {
			siteID,
			buildID,
			status: 'building',
			build_log: '',
			pages_built: 0,
			duration_ms: 0,
			error: '',
			dist_dir: '',
			started_at: now,
			updated_at: now
		};
	}
	stopTimer(buildID);
	void tick(siteID, buildID);
	return () => {
		stopTimer(buildID);
	};
}

export function getState(buildID: string): BuildStatusState | null {
	return states[buildID] ?? null;
}

export function clearBuild(buildID: string): void {
	stopTimer(buildID);
	delete states[buildID];
}

export const buildStatusStore = {
	get all(): Record<string, BuildStatusState> {
		return states;
	}
};
