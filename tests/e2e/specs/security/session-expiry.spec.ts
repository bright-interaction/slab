import { request } from '@playwright/test';
import { createHmac } from 'node:crypto';
import { test, expect, u } from '../../fixtures/auth';

// internal/middleware/auth.go pins the algorithm to HS256, validates exp,
// and rejects empty + tampered tokens. We probe each negative path here.

function b64url(input: Buffer | string): string {
	return Buffer.from(input).toString('base64').replace(/=/g, '').replace(/\+/g, '-').replace(/\//g, '_');
}

async function attemptMe(cookieValue: string): Promise<number> {
	const ctx = await request.newContext({
		extraHTTPHeaders: { Cookie: `atomicsite_token=${cookieValue}` }
	});
	try {
		const res = await ctx.get(u('/api/auth/me'));
		return res.status();
	} finally {
		await ctx.dispose();
	}
}

test.describe('security: malformed/expired/wrong-alg JWTs all return 401', () => {
	test('empty token returns 401', async () => {
		expect(await attemptMe('')).toBe(401);
	});

	test('tampered signature (last char flipped) returns 401', async () => {
		const token = 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0.AAAA';
		expect(await attemptMe(token)).toBe(401);
	});

	test('garbage in the cookie returns 401', async () => {
		expect(await attemptMe('not-a-jwt')).toBe(401);
	});

	test('alg: none is rejected', async () => {
		const header = b64url(JSON.stringify({ alg: 'none', typ: 'JWT' }));
		const payload = b64url(
			JSON.stringify({ sub: 'admin', tv: 0, exp: Math.floor(Date.now() / 1000) + 3600 })
		);
		const token = `${header}.${payload}.`;
		expect(await attemptMe(token)).toBe(401);
	});

	test('expired exp claim is rejected even with valid signature', async () => {
		// Sign with the e2e-jwt-secret so the signature is technically valid;
		// only the expiry claim is bad.
		const secret = 'e2e-jwt-secret-not-for-production';
		const header = b64url(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
		const payload = b64url(
			JSON.stringify({
				sub: 'admin',
				tv: 0,
				exp: Math.floor(Date.now() / 1000) - 60,
				iat: Math.floor(Date.now() / 1000) - 7200
			})
		);
		const data = `${header}.${payload}`;
		const sig = createHmac('sha256', secret).update(data).digest();
		const token = `${data}.${b64url(sig)}`;
		expect(await attemptMe(token)).toBe(401);
	});

	test('valid HS256 signature but unknown sub is rejected', async () => {
		const secret = 'e2e-jwt-secret-not-for-production';
		const header = b64url(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
		const payload = b64url(
			JSON.stringify({
				sub: 'no-such-user-id',
				tv: 0,
				exp: Math.floor(Date.now() / 1000) + 3600,
				iat: Math.floor(Date.now() / 1000)
			})
		);
		const data = `${header}.${payload}`;
		const sig = createHmac('sha256', secret).update(data).digest();
		const token = `${data}.${b64url(sig)}`;
		expect(await attemptMe(token)).toBe(401);
	});
});
