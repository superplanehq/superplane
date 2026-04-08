import { defineConfig } from "vite";
import type { ResolvedConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import * as path from "path";

// Plugin that sets HMR port to be the same as server port
// This is useful when you can't use WebSockets in your proxy
const setHmrPortFromPortPlugin = {
  name: "set-hmr-port-from-port",
  configResolved: (config: ResolvedConfig) => {
    if (!config.server.strictPort) {
      throw new Error("Should be strictPort=true");
    }

    if (config.server.hmr !== false) {
      if (config.server.hmr === true) config.server.hmr = {};
      config.server.hmr ??= {};
      config.server.hmr.clientPort = config.server.port;
      config.server.hmr.overlay = true;
    }
  },
};

// index.html uses Go template syntax for boolean literals; replace it during `vite dev`
// so the inline script stays valid JavaScript (matches AGENT_ENABLED=yes like the app server).
const injectAgentEnabledForViteDev = {
  name: "inject-agent-enabled-for-vite-dev",
  apply: "serve" as const,
  transformIndexHtml(html: string) {
    const enabled = process.env.AGENT_ENABLED === "yes";
    return html.replace(
      /window\.SUPERPLANE_AGENT_ENABLED = \{\{if \.AgentEnabled\}\}true\{\{else\}\}false\{\{end\}\};/,
      `window.SUPERPLANE_AGENT_ENABLED = ${enabled};`,
    );
  },
};

// https://vite.dev/config/
export default defineConfig(({ command }: { command: string }) => {
  const isDev = command !== "build";
  const apiPort = process.env.API_PORT || process.env.PUBLIC_API_PORT || "8000";
  const devPort = Number.parseInt(process.env.VITE_DEV_PORT || "5173", 10);

  return {
    plugins: [react(), tailwindcss(), setHmrPortFromPortPlugin, injectAgentEnabledForViteDev],
    base: "/",
    server: {
      port: devPort,
      strictPort: true,
      host: true,
      watch: {
        usePolling: true,
        interval: 1000,
      },
      proxy: {
        "/api": {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
          secure: false,
        },
        // Account session routes (same origin as production when served from Go; required for pure Vite dev)
        "/account": {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
          secure: false,
        },
        "/organizations": {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
          secure: false,
        },
        "/auth": {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
          secure: false,
        },
      },
    },
    resolve: {
      alias: {
        "@/canvas": path.resolve(__dirname, "src/pages/canvas"),
        "@": path.resolve(__dirname, "src"),
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
    },
  };
});
