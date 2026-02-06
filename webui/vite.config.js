import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig(({ command, mode, ssrBuild }) => {
	const ret = {
		plugins: [vue()],
		resolve: {
			alias: {
				'@': fileURLToPath(new URL('./src', import.meta.url))
			}
		},
		server: {
			proxy: {
				'/session': { target: 'http://host.docker.internal:3000', changeOrigin: true },
				'/users': { target: 'http://host.docker.internal:3000', changeOrigin: true },
				'/user': { target: 'http://host.docker.internal:3000', changeOrigin: true }, // Re-added!
				'/conversations': { target: 'http://host.docker.internal:3000', changeOrigin: true },
				'/groups': { target: 'http://host.docker.internal:3000', changeOrigin: true },
				'/images': { target: 'http://host.docker.internal:3000', changeOrigin: true },
				'/messages': { target: 'http://host.docker.internal:3000', changeOrigin: true },
				'/reactions': { target: 'http://host.docker.internal:3000', changeOrigin: true },
				'/starred': { target: 'http://host.docker.internal:3000', changeOrigin: true },
			}
		}
	};
	ret.define = {
		// Do not modify this constant, it is used in the evaluation.
		"__API_URL__": JSON.stringify(""),
		"__VUE_PROD_HYDRATION_MISMATCH_DETAILS__": JSON.stringify(false),
	};
	return ret;
})
