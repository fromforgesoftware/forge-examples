import { catalog, collection, single, envelope, type JsonApiResource } from './client';
import { useAuthStore } from '@/stores/auth';

// Pet mirrors the catalog service's PetDTO (internal/catalog/api/pet.go).
export type PetStatus = 'AVAILABLE' | 'PENDING' | 'ADOPTED';

export interface PetAttributes {
	name: string;
	species: string;
	status: PetStatus;
	createdAt?: string;
	updatedAt?: string;
}

export interface Pet extends PetAttributes {
	id: string;
}

function toPet(r: JsonApiResource<PetAttributes>): Pet {
	return { id: r.id, ...r.attributes };
}

// ensureToken refreshes the access token if needed before each call (also the
// router guard runs, but a long-lived list view can outlive token expiry).
async function ensureToken(): Promise<void> {
	const ok = await useAuthStore().validToken();
	if (!ok) throw new Error('not authenticated');
}

// listPets → GET /v1/pets. Optional status filter maps to filter[status].
export async function listPets(status?: PetStatus): Promise<Pet[]> {
	await ensureToken();
	const path = status ? `/pets?filter[status]=${encodeURIComponent(status)}` : '/pets';
	const res = await catalog().request({ method: 'GET', path });
	return collection<PetAttributes>(res.body).map(toPet);
}

// getPet → GET /v1/pets/{id}.
export async function getPet(id: string): Promise<Pet> {
	await ensureToken();
	const res = await catalog().request({ method: 'GET', path: `/pets/${encodeURIComponent(id)}` });
	return toPet(single<PetAttributes>(res.body));
}

// createPet → POST /v1/pets.
export async function createPet(input: { name: string; species: string }): Promise<Pet> {
	await ensureToken();
	const res = await catalog().request({
		method: 'POST',
		path: '/pets',
		body: envelope('pets', input),
	});
	return toPet(single<PetAttributes>(res.body));
}
