import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
	base: '/ui/',
	envPrefix: 'SMOOTHDB',
	plugins: [svelte({
		onwarn: (warning, handler) => {
			if (warning.code.startsWith('a11y_')) {
				return; 
			} 
			if (handler) {
				handler(warning);
			}
		},
	})],
	build: {
		// sourcemap: 'inline',
		// minify: false
	}
})

