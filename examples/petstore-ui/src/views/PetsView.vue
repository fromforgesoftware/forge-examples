<template>
	<div class="space-y-6">
		<div class="flex items-end justify-between gap-4">
			<div class="space-y-1">
				<h1 class="text-2xl font-semibold">Pets</h1>
				<p class="text-sm text-muted-foreground">Browse the catalog and adopt an available pet.</p>
			</div>
			<Button @click="createOpen = true">
				<Plus class="mr-2 size-4" />
				New pet
			</Button>
		</div>

		<Alert v-if="error" variant="destructive">
			<AlertTitle>Could not load pets</AlertTitle>
			<AlertDescription>{{ error }}</AlertDescription>
		</Alert>

		<div class="flex items-center gap-2">
			<span class="text-sm text-muted-foreground">Status:</span>
			<Button
				v-for="opt in statusOptions"
				:key="opt.value ?? 'all'"
				:variant="filter === opt.value ? 'default' : 'outline'"
				size="sm"
				@click="setFilter(opt.value)"
			>
				{{ opt.label }}
			</Button>
		</div>

		<div v-if="loading" class="flex justify-center py-16">
			<Spinner class="size-6" />
		</div>

		<EmptyState
			v-else-if="pets.length === 0"
			:icon="PawPrint"
			title="No pets yet"
			description="Add a pet to the catalog to get started."
		>
			<Button @click="createOpen = true">
				<Plus class="mr-2 size-4" />
				New pet
			</Button>
		</EmptyState>

		<DataTable
			v-else
			:columns="columns"
			:data-source="{ data: pets, totalCount: pets.length }"
			:get-row-id="(r) => r.id"
		/>

		<NewPetDialog v-model:open="createOpen" @created="load" />
	</div>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Plus, PawPrint } from '@lucide/vue';
import {
	Alert,
	AlertDescription,
	AlertTitle,
	Button,
	DataTable,
	EmptyState,
	Spinner,
	type ColumnDef,
} from '@fromforgesoftware/vue-kit';
import { listPets, type Pet, type PetStatus } from '@/api/pets';
import StatusBadge from '@/components/StatusBadge.vue';
import NewPetDialog from '@/components/NewPetDialog.vue';

const router = useRouter();
const pets = ref<Pet[]>([]);
const loading = ref(true);
const error = ref('');
const filter = ref<PetStatus | undefined>(undefined);
const createOpen = ref(false);

const statusOptions: { label: string; value: PetStatus | undefined }[] = [
	{ label: 'All', value: undefined },
	{ label: 'Available', value: 'AVAILABLE' },
	{ label: 'Pending', value: 'PENDING' },
	{ label: 'Adopted', value: 'ADOPTED' },
];

const columns: ColumnDef<Pet, unknown>[] = [
	{ accessorKey: 'name', header: 'Name', cell: ({ row }) => h('span', { class: 'font-medium' }, row.original.name) },
	{ accessorKey: 'species', header: 'Species', cell: ({ row }) => h('span', row.original.species) },
	{
		accessorKey: 'status',
		header: 'Status',
		cell: ({ row }) => h(StatusBadge, { status: row.original.status }),
	},
	{
		id: 'actions',
		header: '',
		cell: ({ row }) =>
			h(
				'div',
				{ class: 'flex justify-end' },
				h(
					Button,
					{ variant: 'outline', size: 'sm', onClick: () => router.push(`/pets/${row.original.id}`) },
					() => 'View',
				),
			),
	},
];

function setFilter(value: PetStatus | undefined): void {
	filter.value = value;
	load();
}

async function load(): Promise<void> {
	loading.value = true;
	error.value = '';
	try {
		pets.value = await listPets(filter.value);
	} catch (e) {
		error.value = e instanceof Error ? e.message : 'Request failed.';
	} finally {
		loading.value = false;
	}
}

onMounted(load);
</script>
