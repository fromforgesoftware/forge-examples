<template>
	<div class="space-y-6">
		<Button variant="ghost" size="sm" @click="router.back()">
			<ChevronLeft class="mr-1 size-4" />
			Back
		</Button>

		<div v-if="loading" class="flex justify-center py-16">
			<Spinner class="size-6" />
		</div>

		<Alert v-else-if="error" variant="destructive">
			<AlertTitle>Could not load pet</AlertTitle>
			<AlertDescription>{{ error }}</AlertDescription>
		</Alert>

		<Card v-else-if="pet">
			<CardHeader>
				<div class="flex items-start justify-between gap-4">
					<div>
						<CardTitle class="text-2xl">{{ pet.name }}</CardTitle>
						<CardDescription class="capitalize">{{ pet.species }}</CardDescription>
					</div>
					<StatusBadge :status="pet.status" />
				</div>
			</CardHeader>
			<CardContent class="space-y-2 text-sm text-muted-foreground">
				<div><span class="text-foreground">ID:</span> {{ pet.id }}</div>
				<div v-if="pet.createdAt"><span class="text-foreground">Added:</span> {{ formatDate(pet.createdAt) }}</div>
			</CardContent>
			<CardFooter class="flex-col items-stretch gap-3">
				<Alert v-if="adoptError" variant="destructive">
					<AlertDescription>{{ adoptError }}</AlertDescription>
				</Alert>
				<Button
					v-if="pet.status === 'AVAILABLE'"
					:disabled="adopting"
					@click="confirmOpen = true"
				>
					<Spinner v-if="adopting" class="mr-2 size-4" />
					Adopt {{ pet.name }}
				</Button>
				<p v-else class="text-sm text-muted-foreground">
					This pet is {{ pet.status.toLowerCase() }} and cannot be adopted.
				</p>
			</CardFooter>
		</Card>

		<Dialog :open="confirmOpen" @update:open="confirmOpen = $event">
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Adopt {{ pet?.name }}?</DialogTitle>
					<DialogDescription>
						This places an adoption order: the catalog confirms availability, a payment token is
						vended, and the pet is marked ADOPTED.
					</DialogDescription>
				</DialogHeader>
				<DialogFooter>
					<Button variant="outline" :disabled="adopting" @click="confirmOpen = false">Cancel</Button>
					<Button :disabled="adopting" @click="adopt">
						<Spinner v-if="adopting" class="mr-2 size-4" />
						Confirm adoption
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	</div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { ChevronLeft } from '@lucide/vue';
import {
	Alert,
	AlertDescription,
	AlertTitle,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Spinner,
	toast,
} from '@fromforgesoftware/vue-kit';
import { getPet, type Pet } from '@/api/pets';
import { placeAdoption } from '@/api/adoptions';
import StatusBadge from '@/components/StatusBadge.vue';

const props = defineProps<{ id: string }>();
const router = useRouter();

const pet = ref<Pet | null>(null);
const loading = ref(true);
const error = ref('');
const confirmOpen = ref(false);
const adopting = ref(false);
const adoptError = ref('');

function formatDate(iso: string): string {
	return new Date(iso).toLocaleString();
}

async function load(): Promise<void> {
	loading.value = true;
	error.value = '';
	try {
		pet.value = await getPet(props.id);
	} catch (e) {
		error.value = e instanceof Error ? e.message : 'Request failed.';
	} finally {
		loading.value = false;
	}
}

async function adopt(): Promise<void> {
	if (!pet.value) return;
	adopting.value = true;
	adoptError.value = '';
	try {
		const order = await placeAdoption(pet.value.id);
		confirmOpen.value = false;
		toast.success(`Adoption ${order.status.toLowerCase()} for ${pet.value.name}`);
		await load();
	} catch (e) {
		adoptError.value = e instanceof Error ? e.message : 'Adoption failed.';
	} finally {
		adopting.value = false;
	}
}

onMounted(load);
</script>
