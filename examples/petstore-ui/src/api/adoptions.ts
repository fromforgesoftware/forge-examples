import { adoptions, collection, single, envelope, type JsonApiResource } from './client';
import { useAuthStore } from '@/stores/auth';

// Adoption mirrors the adoptions service's AdoptionDTO
// (internal/adoptions/api/adoption.go). `owner` is taken from the token by the
// service, never the client — we only send `petId`.
export type AdoptionStatus = 'PLACED' | 'COMPLETED' | 'FAILED';

export interface AdoptionAttributes {
	owner: string;
	petId: string;
	status: AdoptionStatus;
	feeCents: number;
	createdAt?: string;
	updatedAt?: string;
}

export interface Adoption extends AdoptionAttributes {
	id: string;
}

function toAdoption(r: JsonApiResource<AdoptionAttributes>): Adoption {
	return { id: r.id, ...r.attributes };
}

async function ensureToken(): Promise<void> {
	const ok = await useAuthStore().validToken();
	if (!ok) throw new Error('not authenticated');
}

// listAdoptions → GET /v1/adoptions (the caller's own orders).
export async function listAdoptions(): Promise<Adoption[]> {
	await ensureToken();
	const res = await adoptions().request({ method: 'GET', path: '/adoptions' });
	return collection<AdoptionAttributes>(res.body).map(toAdoption);
}

// placeAdoption → POST /v1/adoptions. Orchestrates catalog + gleipnir + herald
// server-side; returns the order (status COMPLETED on success).
export async function placeAdoption(petId: string): Promise<Adoption> {
	await ensureToken();
	const res = await adoptions().request({
		method: 'POST',
		path: '/adoptions',
		body: envelope('adoptions', { petId }),
	});
	return toAdoption(single<AdoptionAttributes>(res.body));
}
