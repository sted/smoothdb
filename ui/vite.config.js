import { defineConfig, loadEnv } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// The dev server proxies the smoothdb routes, so the UI is same-origin as in
// production: no CORS, and the browser can read every response header.
// SMOOTHDB_URL (shell or .env.development) names the smoothdb server.
const proxied = ['/api', '/admin', '/token']

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
	const target = process.env.SMOOTHDB_URL || loadEnv(mode, process.cwd(), 'SMOOTHDB').SMOOTHDB_URL || 'http://localhost:4000'
	return {
		base: '/ui/',
		envPrefix: 'SMOOTHDB',
		server: {
			proxy: Object.fromEntries(proxied.map((p) => [p, { target, changeOrigin: true }])),
		},
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
		},
	}
})
