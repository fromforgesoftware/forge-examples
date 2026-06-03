<template>
	<div class="flex min-h-[60vh] items-center justify-center">
		<Card class="w-full max-w-sm">
			<CardHeader class="items-center text-center">
				<div class="mb-2 flex size-12 items-center justify-center rounded-full bg-primary/10">
					<PawPrint class="size-6 text-primary" />
				</div>
				<CardTitle>Welcome to Petstore</CardTitle>
				<CardDescription>Sign in with your aegis account to browse and adopt pets.</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<Alert v-if="error" variant="destructive">
					<AlertTitle>Sign-in failed</AlertTitle>
					<AlertDescription>{{ error }}</AlertDescription>
				</Alert>
				<Button class="w-full" :disabled="busy" @click="signIn">
					<Spinner v-if="busy" class="mr-2 size-4" />
					Sign in with aegis
				</Button>
			</CardContent>
		</Card>
	</div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { PawPrint } from '@lucide/vue';
import {
	Alert,
	AlertDescription,
	AlertTitle,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Spinner,
} from '@fromforgesoftware/vue-kit';
import { useAuthStore } from '@/stores/auth';

const auth = useAuthStore();
const route = useRoute();
const busy = ref(false);
const error = ref('');

// Already signed in? skip straight through.
onMounted(() => {
	if (auth.isAuthenticated) window.location.assign('/pets');
});

async function signIn(): Promise<void> {
	busy.value = true;
	error.value = '';
	try {
		const returnTo = (route.query.returnTo as string) || '/pets';
		await auth.login(returnTo);
		// login() redirects the browser; control rarely returns here.
	} catch (e) {
		error.value = e instanceof Error ? e.message : 'Could not start sign-in.';
		busy.value = false;
	}
}
</script>
