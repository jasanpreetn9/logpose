import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: (import.meta.env?.VITE_API_URL as string | undefined) ?? 'http://localhost:8989',
				changeOrigin: true
			}
		}
	}
});
