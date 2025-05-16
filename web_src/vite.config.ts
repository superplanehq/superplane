import { defineConfig } from 'vite'
import type { ResolvedConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from "@tailwindcss/vite"
import * as path from 'path';

// Plugin that sets HMR port to be the same as server port
// This is useful when you can't use WebSockets in your proxy
const setHmrPortFromPortPlugin = {
  name: 'set-hmr-port-from-port',
  configResolved: (config: ResolvedConfig) => {
    if (!config.server.strictPort) {
      throw new Error('Should be strictPort=true')
    }

    if (config.server.hmr !== false) {
      if (config.server.hmr === true) config.server.hmr = {}
      config.server.hmr ??= {}
      config.server.hmr.clientPort = config.server.port
    }
  }
};

// https://vite.dev/config/
export default defineConfig(({ command }: { command: string} ) => {
  const isDev = command !== "build";

  return {
  plugins: [react(), tailwindcss(), setHmrPortFromPortPlugin],
  base: "/app",
  server: {
    port: 5173,
    strictPort: true,
    host: true,
    proxy: {
      "/api": {
        target: "http://localhost:8000",
        changeOrigin: true,
        secure: false,
      },
    },
  },
  resolve: {
    alias: {
      '@/canvas': path.resolve(__dirname, 'src/canvas'),
      "@": path.resolve(__dirname, 'src'),
    },
  },
  build: {
    commonjsOptions: { transformMixedEsModules: true },
    target: "es2020",
    outDir: "../pkg/web/assets/dist", // emit assets to pkg/web/assets/dist
    emptyOutDir: true,
    sourcemap: isDev, // enable source map in dev build
    manifest: false, // do not generate manifest.json
    // rollupOptions: {
    //   input: {
    //     app: path.resolve('./src/main.tsx'),
    //   },
    //   // output: {
    //   //   // remove hashes to match phoenix way of handling asssets
    //   //   entryFileNames: "[name].js",
    //   //   chunkFileNames: "[name].js",
    //   //   assetFileNames: "[name][extname]",
    //   // },
    // },
  }
};
})
