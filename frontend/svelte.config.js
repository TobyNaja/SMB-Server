import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		// Force runes mode for all project files (can be removed in Svelte 6).
		runes: true
	},
	kit: {
		adapter: adapter({
			fallback: 'index.html' // SPA mode: Go Fiber serves index.html for all non-API paths
		})
	}
};

export default config;
