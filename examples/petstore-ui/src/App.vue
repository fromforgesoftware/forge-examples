<template>
	<div class="min-h-screen bg-background text-foreground">
		<header
			v-if="auth.isAuthenticated"
			class="border-b border-border bg-card"
		>
			<div class="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4">
				<RouterLink to="/pets" class="flex items-center gap-2 font-semibold">
					<PawPrint class="size-5 text-primary" />
					Petstore
				</RouterLink>
				<nav class="flex items-center gap-1 text-sm">
					<RouterLink
						to="/pets"
						class="rounded-md px-3 py-1.5 text-muted-foreground hover:text-foreground"
						active-class="!text-foreground bg-accent"
					>
						Pets
					</RouterLink>
					<RouterLink
						to="/adoptions"
						class="rounded-md px-3 py-1.5 text-muted-foreground hover:text-foreground"
						active-class="!text-foreground bg-accent"
					>
						My adoptions
					</RouterLink>
				</nav>
				<div class="ml-auto flex items-center gap-3">
					<span v-if="auth.profile" class="text-sm text-muted-foreground">
						{{ auth.profile.email || auth.profile.name || auth.profile.subject }}
					</span>
					<Button variant="outline" size="sm" @click="auth.logout()">
						<LogOut class="mr-2 size-4" />
						Log out
					</Button>
				</div>
			</div>
		</header>

		<main class="mx-auto max-w-5xl px-4 py-8">
			<RouterView />
		</main>

		<Toaster />
	</div>
</template>

<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router';
import { LogOut, PawPrint } from '@lucide/vue';
import { Button, Toaster } from '@fromforgesoftware/vue-kit';
import { useAuthStore } from '@/stores/auth';

const auth = useAuthStore();
</script>
