import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { createSafeStorage, JsonStorage } from '@fromforgesoftware/ts-kit/storage';
import { appConfig } from '@/config/runtime';
import { createCodeVerifier, codeChallengeS256, randomString } from '@/auth/pkce';
import { discover, exchangeCode, refreshTokens, decodeJwtPayload, type TokenResponse } from '@/auth/oidc';

// SECURITY MODEL (browser-side public client):
//
// - The ACCESS token lives only in memory (this store). It is never persisted,
//   so an XSS-readable store (localStorage) cannot leak a long-lived bearer.
// - The REFRESH token + the post-login redirect path are kept in sessionStorage
//   (cleared when the tab closes, not shared across tabs). This is the standard
//   trade-off for a public SPA with no httpOnly-cookie backend; the alternative
//   (silent-iframe re-auth) needs an IdP session and is out of scope for this
//   reference. The realm should issue short-lived access tokens + rotating
//   refresh tokens to bound exposure.
// - PKCE transaction state (verifier, OAuth state, OIDC nonce) is held in
//   sessionStorage only between the authorize redirect and the callback, then
//   deleted. PKCE S256 means an intercepted code is useless without the
//   verifier; `state` defeats CSRF; `nonce` binds the id_token to this request.

const SS_REFRESH = 'petstore.oidc.refresh_token';
const SS_TX = 'petstore.oidc.tx'; // in-flight authorize transaction
const SS_RETURN = 'petstore.oidc.return_to';

interface AuthTx {
	codeVerifier: string;
	state: string;
	nonce: string;
}

interface Profile {
	subject: string;
	name?: string;
	email?: string;
}

const session = new JsonStorage(createSafeStorage('session'));

export const useAuthStore = defineStore('auth', () => {
	// Access token: in memory only.
	const accessToken = ref<string | null>(null);
	const expiresAt = ref<number>(0); // epoch ms
	const profile = ref<Profile | null>(null);
	const initialized = ref(false);

	const isAuthenticated = computed(() => accessToken.value !== null && Date.now() < expiresAt.value);

	function setTokens(t: TokenResponse): void {
		accessToken.value = t.access_token;
		// expires_in is seconds; default to 300s. Subtract a 30s skew so we
		// refresh slightly early rather than on a just-expired token.
		const ttlMs = (t.expires_in ?? 300) * 1000;
		expiresAt.value = Date.now() + Math.max(ttlMs - 30_000, 0);
		if (t.refresh_token) session.set(SS_REFRESH, t.refresh_token);
	}

	function clear(): void {
		accessToken.value = null;
		expiresAt.value = 0;
		profile.value = null;
		session.remove(SS_REFRESH);
		session.remove(SS_TX);
	}

	// login starts the authorization_code + PKCE flow: discover, build a
	// verifier/challenge + state + nonce, stash the transaction, and redirect
	// the browser to the realm's authorize endpoint.
	async function login(returnTo?: string): Promise<void> {
		const cfg = appConfig();
		const disco = await discover(cfg.aegisIssuer);

		const codeVerifier = createCodeVerifier();
		const challenge = await codeChallengeS256(codeVerifier);
		const state = randomString(32);
		const nonce = randomString(32);

		session.set<AuthTx>(SS_TX, { codeVerifier, state, nonce });
		if (returnTo) session.set(SS_RETURN, returnTo);

		const url = new URL(disco.authorization_endpoint);
		url.searchParams.set('response_type', 'code');
		url.searchParams.set('client_id', cfg.oidcClientId);
		url.searchParams.set('redirect_uri', cfg.oidcRedirectUri);
		url.searchParams.set('scope', 'openid profile email offline_access');
		url.searchParams.set('state', state);
		url.searchParams.set('nonce', nonce);
		url.searchParams.set('code_challenge', challenge);
		url.searchParams.set('code_challenge_method', 'S256');

		window.location.assign(url.toString());
	}

	// handleCallback validates state, exchanges the code (public client + PKCE
	// verifier), validates the id_token nonce, and seeds tokens + profile.
	// Returns the path to navigate to after a successful login.
	async function handleCallback(query: URLSearchParams): Promise<string> {
		const tx = session.get<AuthTx>(SS_TX);
		session.remove(SS_TX);

		const error = query.get('error');
		if (error) throw new Error(query.get('error_description') || error);

		const code = query.get('code');
		const returnedState = query.get('state');
		if (!code) throw new Error('callback missing authorization code');
		if (!tx) throw new Error('no in-flight login transaction (start login again)');
		if (returnedState !== tx.state) throw new Error('OAuth state mismatch (possible CSRF)');

		const cfg = appConfig();
		const disco = await discover(cfg.aegisIssuer);
		const tokens = await exchangeCode({
			tokenEndpoint: disco.token_endpoint,
			clientId: cfg.oidcClientId,
			code,
			redirectUri: cfg.oidcRedirectUri,
			codeVerifier: tx.codeVerifier,
		});

		if (tokens.id_token) {
			const claims = decodeJwtPayload(tokens.id_token);
			if (claims.nonce !== tx.nonce) throw new Error('id_token nonce mismatch (possible replay)');
			profile.value = {
				subject: String(claims.sub ?? ''),
				name: claims.name as string | undefined,
				email: claims.email as string | undefined,
			};
		}

		setTokens(tokens);

		const returnTo = session.get<string>(SS_RETURN) ?? '/pets';
		session.remove(SS_RETURN);
		return returnTo;
	}

	// validToken returns a usable access token, refreshing it first if expired
	// and a refresh token is available. Returns null if re-login is required.
	async function validToken(): Promise<string | null> {
		if (accessToken.value && Date.now() < expiresAt.value) return accessToken.value;

		const refresh = session.get<string>(SS_REFRESH);
		if (!refresh) {
			clear();
			return null;
		}
		try {
			const cfg = appConfig();
			const disco = await discover(cfg.aegisIssuer);
			const tokens = await refreshTokens({
				tokenEndpoint: disco.token_endpoint,
				clientId: cfg.oidcClientId,
				refreshToken: refresh,
			});
			setTokens(tokens);
			return accessToken.value;
		} catch {
			clear();
			return null;
		}
	}

	// restore attempts a silent session restore on app load using a persisted
	// refresh token (e.g. after a full reload that wiped the in-memory access
	// token). Marks the store initialized either way.
	async function restore(): Promise<void> {
		if (!isAuthenticated.value && session.get<string>(SS_REFRESH)) {
			await validToken();
		}
		initialized.value = true;
	}

	// logout clears local tokens and, if the realm advertises one, follows the
	// RP-initiated end_session endpoint to terminate the IdP session too.
	async function logout(): Promise<void> {
		clear();
		const cfg = appConfig();
		try {
			const disco = await discover(cfg.aegisIssuer);
			if (disco.end_session_endpoint) {
				const url = new URL(disco.end_session_endpoint);
				url.searchParams.set('client_id', cfg.oidcClientId);
				url.searchParams.set('post_logout_redirect_uri', window.location.origin);
				window.location.assign(url.toString());
				return;
			}
		} catch {
			// discovery failed — fall through to a local redirect.
		}
		window.location.assign('/');
	}

	return {
		accessToken,
		profile,
		initialized,
		isAuthenticated,
		login,
		handleCallback,
		validToken,
		restore,
		logout,
	};
});
