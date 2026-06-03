// OIDC discovery + token endpoint calls for a PUBLIC client using
// authorization_code + PKCE. Kept dependency-free (native fetch); ts-kit's
// ApiClient is JSON:API-specific (vnd.api+json media type) and these are plain
// OAuth form/JSON endpoints, so a small fetch wrapper is the right tool here.

export interface OidcDiscovery {
	issuer: string;
	authorization_endpoint: string;
	token_endpoint: string;
	jwks_uri: string;
	end_session_endpoint?: string;
	userinfo_endpoint?: string;
}

export interface TokenResponse {
	access_token: string;
	token_type: string;
	expires_in?: number;
	refresh_token?: string;
	id_token?: string;
	scope?: string;
}

let discoveryCache: { issuer: string; doc: OidcDiscovery } | null = null;

// discover fetches {issuer}/.well-known/openid-configuration. Cached per issuer
// for the page lifetime — the document is stable.
export async function discover(issuer: string): Promise<OidcDiscovery> {
	if (discoveryCache && discoveryCache.issuer === issuer) return discoveryCache.doc;
	const url = `${issuer.replace(/\/+$/, '')}/.well-known/openid-configuration`;
	const res = await fetch(url, { headers: { Accept: 'application/json' } });
	if (!res.ok) {
		throw new Error(`OIDC discovery failed (${res.status}) at ${url}`);
	}
	const doc = (await res.json()) as OidcDiscovery;
	if (!doc.authorization_endpoint || !doc.token_endpoint) {
		throw new Error('OIDC discovery document is missing required endpoints');
	}
	discoveryCache = { issuer, doc };
	return doc;
}

// exchangeCode swaps an authorization code for tokens at the token endpoint.
// PUBLIC client: client_id is sent but NO client_secret; PKCE code_verifier is
// the proof of possession.
export async function exchangeCode(params: {
	tokenEndpoint: string;
	clientId: string;
	code: string;
	redirectUri: string;
	codeVerifier: string;
}): Promise<TokenResponse> {
	const body = new URLSearchParams({
		grant_type: 'authorization_code',
		client_id: params.clientId,
		code: params.code,
		redirect_uri: params.redirectUri,
		code_verifier: params.codeVerifier,
	});
	return postToken(params.tokenEndpoint, body);
}

// refreshTokens uses a refresh_token grant (public client, no secret).
export async function refreshTokens(params: {
	tokenEndpoint: string;
	clientId: string;
	refreshToken: string;
}): Promise<TokenResponse> {
	const body = new URLSearchParams({
		grant_type: 'refresh_token',
		client_id: params.clientId,
		refresh_token: params.refreshToken,
	});
	return postToken(params.tokenEndpoint, body);
}

async function postToken(endpoint: string, body: URLSearchParams): Promise<TokenResponse> {
	const res = await fetch(endpoint, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded',
			Accept: 'application/json',
		},
		body,
	});
	const json = (await res.json().catch(() => ({}))) as Record<string, unknown>;
	if (!res.ok) {
		const desc = (json.error_description as string) || (json.error as string) || `HTTP ${res.status}`;
		throw new Error(`token endpoint error: ${desc}`);
	}
	return json as unknown as TokenResponse;
}

// decodeJwtPayload reads (does not verify) a JWT's payload. The id_token's
// SIGNATURE is verified server-side by the petstore services (which validate
// the realm's RS256 JWKS on every API call); the SPA only needs the claims to
// check the nonce and read expiry/identity. We never trust these claims for
// authorization decisions beyond the nonce check.
export function decodeJwtPayload(jwt: string): Record<string, unknown> {
	const parts = jwt.split('.');
	if (parts.length !== 3) throw new Error('malformed JWT');
	const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/');
	const padded = payload.padEnd(payload.length + ((4 - (payload.length % 4)) % 4), '=');
	return JSON.parse(decodeURIComponent(escape(atob(padded)))) as Record<string, unknown>;
}
