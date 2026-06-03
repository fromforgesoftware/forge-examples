// PKCE (RFC 7636) + OAuth/OIDC nonce/state helpers, built on the Web Crypto API.
// No external OIDC library: ts-kit ships no OIDC/PKCE helper, and the public
// authorization_code + PKCE flow is small enough to implement correctly and
// auditable in one file. crypto.subtle is available in all evergreen browsers
// over HTTPS (and on http://localhost).

const UNRESERVED = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';

// randomString returns `length` chars drawn from the RFC 7636 unreserved set
// using a CSPRNG. Used for the code_verifier (43–128 chars) and for the
// state / nonce values.
export function randomString(length: number): string {
	const bytes = new Uint32Array(length);
	crypto.getRandomValues(bytes);
	let out = '';
	for (let i = 0; i < length; i++) {
		out += UNRESERVED[bytes[i] % UNRESERVED.length];
	}
	return out;
}

// base64UrlEncode encodes bytes as base64url (RFC 4648 §5), no padding —
// the encoding required for the S256 code_challenge.
function base64UrlEncode(bytes: ArrayBuffer): string {
	const view = new Uint8Array(bytes);
	let binary = '';
	for (let i = 0; i < view.length; i++) binary += String.fromCharCode(view[i]);
	return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// createCodeVerifier returns a high-entropy verifier (96 chars, within the
// 43–128 range mandated by RFC 7636 §4.1).
export function createCodeVerifier(): string {
	return randomString(96);
}

// codeChallengeS256 computes the S256 challenge: base64url(SHA-256(verifier)).
export async function codeChallengeS256(verifier: string): Promise<string> {
	const data = new TextEncoder().encode(verifier);
	const digest = await crypto.subtle.digest('SHA-256', data);
	return base64UrlEncode(digest);
}
