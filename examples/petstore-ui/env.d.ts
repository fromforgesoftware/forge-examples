/// <reference types="vite/client" />

declare module '*.vue' {
	import type { DefineComponent } from 'vue';
	const component: DefineComponent<object, object, unknown>;
	export default component;
}

// Build-time configuration. Each var has a runtime override in
// /config.js (see src/config/runtime.ts) so the built image stays portable.
interface ImportMetaEnv {
	readonly VITE_AEGIS_ISSUER?: string;
	readonly VITE_OIDC_CLIENT_ID?: string;
	readonly VITE_OIDC_REDIRECT_URI?: string;
	readonly VITE_CATALOG_URL?: string;
	readonly VITE_ADOPTIONS_URL?: string;
}
interface ImportMeta {
	readonly env: ImportMetaEnv;
}

// Runtime config injected by nginx-served /config.js at deploy time.
interface Window {
	__PETSTORE_CONFIG__?: Partial<{
		aegisIssuer: string;
		oidcClientId: string;
		oidcRedirectUri: string;
		catalogUrl: string;
		adoptionsUrl: string;
	}>;
}
