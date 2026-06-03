<template>
	<Dialog :open="open" @update:open="emit('update:open', $event)">
		<DialogContent>
			<DialogHeader>
				<DialogTitle>New pet</DialogTitle>
				<DialogDescription>Add an animal to the catalog. It starts as AVAILABLE.</DialogDescription>
			</DialogHeader>
			<DialogBody class="space-y-4">
				<FormField label="Name" for="pet-name">
					<Input id="pet-name" v-model="name" placeholder="Rex" />
				</FormField>
				<FormField label="Species" for="pet-species">
					<Input id="pet-species" v-model="species" placeholder="dog" />
				</FormField>
				<p v-if="error" class="text-sm text-destructive">{{ error }}</p>
			</DialogBody>
			<DialogFooter>
				<Button variant="outline" :disabled="busy" @click="emit('update:open', false)">Cancel</Button>
				<Button :disabled="busy || !name || !species" @click="submit">
					<Spinner v-if="busy" class="mr-2 size-4" />
					Create
				</Button>
			</DialogFooter>
		</DialogContent>
	</Dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import {
	Button,
	Dialog,
	DialogBody,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	FormField,
	Input,
	Spinner,
	toast,
} from '@fromforgesoftware/vue-kit';
import { createPet } from '@/api/pets';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ 'update:open': [value: boolean]; created: [] }>();

const name = ref('');
const species = ref('');
const busy = ref(false);
const error = ref('');

// Reset the form each time the dialog opens.
watch(
	() => props.open,
	(open) => {
		if (open) {
			name.value = '';
			species.value = '';
			error.value = '';
		}
	},
);

async function submit(): Promise<void> {
	busy.value = true;
	error.value = '';
	try {
		const pet = await createPet({ name: name.value, species: species.value });
		toast.success(`Added ${pet.name}`);
		emit('update:open', false);
		emit('created');
	} catch (e) {
		error.value = e instanceof Error ? e.message : 'Failed to create pet.';
	} finally {
		busy.value = false;
	}
}
</script>
