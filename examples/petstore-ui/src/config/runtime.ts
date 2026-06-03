// Resolved runtime configuration for the SPA.
//
// Precedence (highest first):
//   1. window.__PETSTORE_CONFIG__ — injected by /config.js, which nginx renders
//      from env at container start. This keeps the built image portable: the
//      same image is deployed to dev/stage/prod with different URLs.
//   2. import.meta.env.VITE_* — baked at build time (handy for `vite dev`).
//
// Nothing here is secret: a PUBLIC OIDC client has no secret, and the issuer /
// API URLs are public. So shipping them to the browser is expected.

export interface AppConfig {
	/** Aegis realm issuer, e.g. https://aegis.example.com/realms/petstore */
	aegisIssuer: string;
	/** Public OIDC client id registered in the realm. */
	oidcClientId: string;
	/** Redirect URI registered for this client, e.g. https://app/callback */
	oidcRedirectUri: string;
	/** catalog service REST base, e.g. https://catalog.example.com */
	catalogUrl: string;
	/** adoptions service REST base, e.g. https://adoptions.example.com */
	adoptionsUrl: string;
}

function pick(runtimeKey: keyof NonNullable<Window['__PETSTORE_CONFIG__']>, envValue: string | undefined): string {
	const runtime = window.__PETSTORE_CONFIG__?.[runtimeKey];
	// Treat the unsubstituted "${VAR}" placeholder (config.js default) as unset.
	if (runtime && !runtime.startsWith('${')) return runtime;
	return envValue ?? '';
}

let cached: AppConfig | null = null;

export function appConfig(): AppConfig {
	if (cached) return cached;
	cached = {
		aegisIssuer: pick('aegisIssuer', import.meta.env.VITE_AEGIS_ISSUER).replace(/\/+$/, ''),
		oidcClientId: pick('oidcClientId', import.meta.env.VITE_OIDC_CLIENT_ID),
		oidcRedirectUri: pick('oidcRedirectUri', import.meta.env.VITE_OIDC_REDIRECT_URI),
		catalogUrl: pick('catalogUrl', import.meta.env.VITE_CATALOG_URL).replace(/\/+$/, ''),
		adoptionsUrl: pick('adoptionsUrl', import.meta.env.VITE_ADOPTIONS_URL).replace(/\/+$/, ''),
	};
	return cached;
}
