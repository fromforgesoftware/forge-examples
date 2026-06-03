<template>
	<div class="space-y-6">
		<div class="space-y-1">
			<h1 class="text-2xl font-semibold">My adoptions</h1>
			<p class="text-sm text-muted-foreground">Adoption orders placed under your account.</p>
		</div>

		<Alert v-if="error" variant="destructive">
			<AlertTitle>Could not load adoptions</AlertTitle>
			<AlertDescription>{{ error }}</AlertDescription>
		</Alert>

		<div v-if="loading" class="flex justify-center py-16">
			<Spinner class="size-6" />
		</div>

		<EmptyState
			v-else-if="orders.length === 0"
			:icon="Heart"
			title="No adoptions yet"
			description="Browse the catalog and adopt an available pet."
		>
			<Button @click="router.push('/pets')">Browse pets</Button>
		</EmptyState>

		<DataTable
			v-else
			:columns="columns"
			:data-source="{ data: orders, totalCount: orders.length }"
			:get-row-id="(r) => r.id"
		/>
	</div>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Heart } from '@lucide/vue';
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
import { listAdoptions, type Adoption } from '@/api/adoptions';
import StatusBadge from '@/components/StatusBadge.vue';

const router = useRouter();
const orders = ref<Adoption[]>([]);
const loading = ref(true);
const error = ref('');

const columns: ColumnDef<Adoption, unknown>[] = [
	{ accessorKey: 'petId', header: 'Pet', cell: ({ row }) => h('span', { class: 'font-mono text-xs' }, row.original.petId) },
	{
		accessorKey: 'feeCents',
		header: 'Fee',
		cell: ({ row }) => h('span', `$${(row.original.feeCents / 100).toFixed(2)}`),
	},
	{
		accessorKey: 'status',
		header: 'Status',
		cell: ({ row }) => h(StatusBadge, { status: row.original.status }),
	},
];

async function load(): Promise<void> {
	loading.value = true;
	error.value = '';
	try {
		orders.value = await listAdoptions();
	} catch (e) {
		error.value = e instanceof Error ? e.message : 'Request failed.';
	} finally {
		loading.value = false;
	}
}

onMounted(load);
</script>
