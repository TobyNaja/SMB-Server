import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			fallback: 'index.html' // SPA mode: Go Fiber serves index.html for all non-API paths
		})
	}
};

export default config;
