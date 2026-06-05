<template>
	<div class="flex min-h-[60vh] items-center justify-center">
		<Card class="w-full max-w-sm">
			<!-- Error path: show what went wrong + a manual retry. -->
			<template v-if="error">
				<CardHeader class="items-center text-center">
					<div class="mb-2 flex size-12 items-center justify-center rounded-full bg-primary/10">
						<PawPrint class="size-6 text-primary" />
					</div>
					<CardTitle>Sign in to Petstore</CardTitle>
					<CardDescription>You'll be redirected to aegis to sign in.</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					<Alert variant="destructive">
						<AlertTitle>Sign-in failed</AlertTitle>
						<AlertDescription>{{ error }}</AlertDescription>
					</Alert>
					<Button class="w-full" :disabled="busy" @click="signIn">
						<Spinner v-if="busy" class="mr-2 size-4" />
						Try again
					</Button>
				</CardContent>
			</template>

			<!-- Default path: we're already redirecting to aegis; just a spinner. -->
			<CardContent v-else class="flex flex-col items-center gap-3 py-10 text-center">
				<Spinner class="size-6" />
				<p class="text-sm text-muted-foreground">Redirecting to sign-in…</p>
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
// Seeded from the guard's fallback (?error=…). When set, we DON'T auto-retry
// (that could loop on a persistent failure) — the user retries manually.
const error = ref((route.query.error as string) || '');

// aegis is the only way in: if we land here without an error, start the OIDC
// redirect immediately instead of showing a sign-in page. (The guard already
// does this for protected routes; this covers a direct visit to /login.)
onMounted(() => {
	if (auth.isAuthenticated) {
		window.location.assign('/pets');
		return;
	}
	if (!error.value) void signIn();
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
