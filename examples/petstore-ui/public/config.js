// Runtime configuration fallback, served at /config.js.
//
// In `vite dev` this file is served as-is and every value is empty, so the app
// falls back to import.meta.env (your .env / .env.local VITE_* vars).
//
// In the container, nginx renders this from environment variables at startup
// (see deploy/ and docker-entrypoint) so the built image needs no rebuild to
// change issuer / client / API URLs. Do NOT put secrets here — it is public.
window.__PETSTORE_CONFIG__ = {
	aegisIssuer: '',
	oidcClientId: '',
	oidcRedirectUri: '',
	catalogUrl: '',
	adoptionsUrl: '',
};
