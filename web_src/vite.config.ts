import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from "@tailwindcss/vite"
import * as path from 'path';



// https://vite.dev/config/
export default defineConfig(({ command }: { command: string} ) => {
  const isDev = command !== "build";

  return {
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
        secure: false,
        ws: true,
      },
    },
  },
  resolve: {
    alias: {
      '@/canvas': path.resolve('./src/canvas'),
      "@": path.resolve('./src'),
    },
  },
  build: {
    commonjsOptions: { transformMixedEsModules: true },
    target: "es2020",
    outDir: "../pkg/web/assets/dist", // emit assets to /dist
    emptyOutDir: true,
    sourcemap: isDev, // enable source map in dev build
    manifest: false, // do not generate manifest.json
    rollupOptions: {
      input: {
        app: path.resolve('./src/main.tsx'),
      },
      // output: {
      //   // remove hashes to match phoenix way of handling asssets
      //   entryFileNames: "[name].js",
      //   chunkFileNames: "[name].js",
      //   assetFileNames: "[name][extname]",
      // },
    },
  }
};
})
