<template>
	<div class="flex min-h-[60vh] items-center justify-center">
		<Card v-if="error" class="w-full max-w-sm">
			<CardHeader>
				<CardTitle>Sign-in failed</CardTitle>
				<CardDescription>The login could not be completed.</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<Alert variant="destructive">
					<AlertDescription>{{ error }}</AlertDescription>
				</Alert>
				<Button class="w-full" @click="router.replace('/login')">Back to sign in</Button>
			</CardContent>
		</Card>
		<div v-else class="flex flex-col items-center gap-3 text-muted-foreground">
			<Spinner class="size-6" />
			<p>Completing sign-in…</p>
		</div>
	</div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import {
	Alert,
	AlertDescription,
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
const router = useRouter();
const error = ref('');

onMounted(async () => {
	try {
		const params = new URLSearchParams(window.location.search);
		const returnTo = await auth.handleCallback(params);
		// Replace so the code-bearing URL never lands in history.
		await router.replace(returnTo);
	} catch (e) {
		error.value = e instanceof Error ? e.message : 'Unknown error.';
	}
});
</script>
