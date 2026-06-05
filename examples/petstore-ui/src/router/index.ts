import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const routes: RouteRecordRaw[] = [
	{ path: '/', redirect: '/pets' },
	{
		path: '/login',
		name: 'login',
		component: () => import('@/views/LoginView.vue'),
		meta: { public: true },
	},
	{
		path: '/callback',
		name: 'callback',
		component: () => import('@/views/CallbackView.vue'),
		meta: { public: true },
	},
	{
		path: '/pets',
		name: 'pets',
		component: () => import('@/views/PetsView.vue'),
	},
	{
		path: '/pets/:id',
		name: 'pet-detail',
		component: () => import('@/views/PetDetailView.vue'),
		props: true,
	},
	{
		path: '/adoptions',
		name: 'adoptions',
		component: () => import('@/views/AdoptionsView.vue'),
	},
	{ path: '/:pathMatch(.*)*', redirect: '/pets' },
];

const router = createRouter({
	history: createWebHistory(),
	routes,
});

// Auth guard: public routes always pass; everything else needs a valid token.
// `restore()` runs once to silently re-auth from a persisted refresh token
// after a full reload (the in-memory access token is gone). Unauthenticated
// users go STRAIGHT to the aegis OIDC login (no SPA sign-in page): the guard
// starts the authorize redirect itself and aborts the in-app navigation. The
// intended path is remembered (login() stashes it) so the callback returns
// the user there. If starting the redirect fails (e.g. discovery is down) we
// fall back to /login, which surfaces the error with a retry.
router.beforeEach(async (to) => {
	const auth = useAuthStore();
	if (!auth.initialized) await auth.restore();

	if (to.meta.public) return true;
	if (auth.isAuthenticated) return true;

	const token = await auth.validToken();
	if (token) return true;

	try {
		await auth.login(to.fullPath);
		return false; // browser is redirecting to aegis; cancel this navigation
	} catch (e) {
		const error = e instanceof Error ? e.message : 'Could not start sign-in.';
		return { name: 'login', query: { error } };
	}
});

export default router;
