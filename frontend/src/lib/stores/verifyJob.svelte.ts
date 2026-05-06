import * as migrations from '$lib/api/migrations';
import type { VerifyJob, VerifyJobStatus } from '$lib/api/migrations';

// Mirrors buildStatus.svelte.ts (Sprint 4.6, 2026-05-06). Polls the
// async verify-job lifecycle so the migration UI can render a progress
// bar + cancel button while a long crawl runs in the background.
//
// Cadence: 2s for the first 30s, then 5s. Hard timeout at 30 min so a
// crashed worker doesn't leave the UI poll forever (the backend reaper
// also handles this on the server side, but the client timeout is the
// human-visible safety net).
//
// Terminal states close the timer + freeze the row at the last known
// counters so the UI can keep rendering the result summary.

export type VerifyJobView = VerifyJobStatus | 'timeout';

export interface VerifyJobPollState {
	siteID: string;
	migrationID: string;
	jobID: string;
	status: VerifyJobView;
	total_urls: number;
	processed_urls: number;
	ok_count: number;
	fail_count: number;
	deployed_domain: string;
	error: string;
	started_at: number;
	updated_at: number;
}

const FAST_INTERVAL_MS = 2_000;
const SLOW_INTERVAL_MS = 5_000;
const SLOW_AFTER_MS = 30_000;
const TIMEOUT_AFTER_MS = 30 * 60_000;

const states = $state<Record<string, VerifyJobPollState>>({});
const timers: Record<string, ReturnType<typeof setTimeout>> = {};

function isTerminal(status: VerifyJobView): boolean {
	return (
		status === 'done' ||
		status === 'failed' ||
		status === 'cancelled' ||
		status === 'timeout'
	);
}

function applyTick(jobID: string, resp: VerifyJob): void {
	const prev = states[jobID];
	if (!prev) return;
	states[jobID] = {
		...prev,
		status: (resp.status as VerifyJobStatus) ?? prev.status,
		total_urls: resp.total_urls ?? prev.total_urls,
		processed_urls: resp.processed_urls ?? prev.processed_urls,
		ok_count: resp.ok_count ?? prev.ok_count,
		fail_count: resp.fail_count ?? prev.fail_count,
		deployed_domain: resp.deployed_domain ?? prev.deployed_domain,
		error: resp.error ?? prev.error,
		updated_at: Date.now()
	};
}

function stopTimer(jobID: string): void {
	const t = timers[jobID];
	if (t !== undefined) {
		clearTimeout(t);
		delete timers[jobID];
	}
}

function scheduleNext(siteID: string, migrationID: string, jobID: string): void {
	const cur = states[jobID];
	if (!cur || isTerminal(cur.status)) {
		stopTimer(jobID);
		return;
	}
	const elapsed = Date.now() - cur.started_at;
	if (elapsed >= TIMEOUT_AFTER_MS) {
		states[jobID] = { ...cur, status: 'timeout', updated_at: Date.now() };
		stopTimer(jobID);
		return;
	}
	const delay = elapsed >= SLOW_AFTER_MS ? SLOW_INTERVAL_MS : FAST_INTERVAL_MS;
	timers[jobID] = setTimeout(() => {
		void tick(siteID, migrationID, jobID);
	}, delay);
}

async function tick(siteID: string, migrationID: string, jobID: string): Promise<void> {
	const cur = states[jobID];
	if (!cur || isTerminal(cur.status)) {
		stopTimer(jobID);
		return;
	}
	try {
		const resp = await migrations.getVerifyJob(siteID, migrationID, jobID);
		applyTick(jobID, resp);
	} catch {
		// Swallow transient errors; keep polling until terminal or timeout.
		// A 404 here usually means a wrong migration_id was passed; the
		// 30-min timeout will eventually surface that without a noisy
		// loop.
	}
	scheduleNext(siteID, migrationID, jobID);
}

// pollVerifyJob seeds the row, kicks off the first tick, and returns a
// cancel function the component should call on cleanup. Idempotent: a
// second call with the same jobID restarts the timer without losing
// counter state.
export function pollVerifyJob(
	siteID: string,
	migrationID: string,
	jobID: string,
	seed?: { total_urls?: number; deployed_domain?: string }
): () => void {
	const now = Date.now();
	if (!states[jobID]) {
		states[jobID] = {
			siteID,
			migrationID,
			jobID,
			status: 'queued',
			total_urls: seed?.total_urls ?? 0,
			processed_urls: 0,
			ok_count: 0,
			fail_count: 0,
			deployed_domain: seed?.deployed_domain ?? '',
			error: '',
			started_at: now,
			updated_at: now
		};
	}
	stopTimer(jobID);
	void tick(siteID, migrationID, jobID);
	return () => {
		stopTimer(jobID);
	};
}

export function getVerifyJobState(jobID: string): VerifyJobPollState | null {
	return states[jobID] ?? null;
}

export function clearVerifyJob(jobID: string): void {
	stopTimer(jobID);
	delete states[jobID];
}

export const verifyJobStore = {
	get all(): Record<string, VerifyJobPollState> {
		return states;
	}
};
