import { ApiClient } from '@fromforgesoftware/ts-kit/jsonapi-client';
import { appConfig } from '@/config/runtime';
import { useAuthStore } from '@/stores/auth';

// Two ts-kit JSON:API clients, one per petstore service. The kit handles the
// vnd.api+json media type, JSON:API error parsing (→ ApiError), and the path
// joining; we just point each at its base + /v1 basePath.
//
// The Authorization header is injected via the kit's async-unaware header
// factory. ApiClient's factory is synchronous, so callers must ensure a fresh
// token first (await auth.validToken()) — the helpers in pets.ts / adoptions.ts
// do exactly that before each call, which also drives refresh-on-expiry.

function bearerHeaders(): Record<string, string> {
	const auth = useAuthStore();
	return auth.accessToken ? { Authorization: `Bearer ${auth.accessToken}` } : {};
}

let catalogClient: ApiClient | null = null;
let adoptionsClient: ApiClient | null = null;

export function catalog(): ApiClient {
	if (!catalogClient) {
		catalogClient = ApiClient.create({
			baseUrl: appConfig().catalogUrl,
			basePath: '/v1',
			headers: bearerHeaders,
		});
	}
	return catalogClient;
}

export function adoptions(): ApiClient {
	if (!adoptionsClient) {
		adoptionsClient = ApiClient.create({
			baseUrl: appConfig().adoptionsUrl,
			basePath: '/v1',
			headers: bearerHeaders,
		});
	}
	return adoptionsClient;
}

// JSON:API resource shape as returned by the petstore services. We use the
// ApiClient `request` escape hatch (same pattern as the forge console) rather
// than decorated model classes — the attributes are a small, stable map.
export interface JsonApiResource<A = Record<string, unknown>> {
	id: string;
	type: string;
	attributes: A;
}

interface CollectionDoc<A> {
	data?: JsonApiResource<A>[];
}
interface SingleDoc<A> {
	data: JsonApiResource<A>;
}

export function collection<A>(body: unknown): JsonApiResource<A>[] {
	return (body as CollectionDoc<A>).data ?? [];
}
export function single<A>(body: unknown): JsonApiResource<A> {
	return (body as SingleDoc<A>).data;
}

export function envelope(type: string, attributes: Record<string, unknown>) {
	return { data: { type, attributes } };
}
