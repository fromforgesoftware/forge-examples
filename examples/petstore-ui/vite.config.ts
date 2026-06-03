import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';

// A plain Vite SPA build. Unlike the forge console (a SystemJS plugin host),
// petstore-ui consumes the PUBLISHED @fromforgesoftware/{ts,vue}-kit packages —
// no source aliases, no Module Federation. It is a standalone reference app.
export default defineConfig({
	plugins: [vue(), tailwindcss()],
	resolve: {
		alias: {
			'@': fileURLToPath(new URL('./src', import.meta.url)),
		},
	},
});
