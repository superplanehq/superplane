import { defineConfig } from "vite";
import type { ResolvedConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import * as path from "path";
import { keepWebAppManifestLinkOnPageOrigin } from "./src/lib/webAppManifestLink.ts";

// Vite applies `base` to every root-absolute link, including the web app
// manifest. Install checks then resolve start_url and icons on the asset host.
// Put only that link back on the page origin after the base rewrite.
const keepWebAppManifestOnPageOriginPlugin = {
  name: "keep-web-app-manifest-on-page-origin",
  transformIndexHtml: {
    order: "post" as const,
    handler(html: string) {
      return keepWebAppManifestLinkOnPageOrigin(html);
    },
  },
};

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

// https://vite.dev/config/
export default defineConfig(() => {
  const isProduction = process.env.APP_ENV === "production";
  const apiPort = process.env.API_PORT || process.env.PUBLIC_API_PORT || "8000";
  const devPort = Number.parseInt(process.env.VITE_DEV_PORT || "5173", 10);
  const assetBaseUrl = process.env.VITE_ASSET_BASE_URL?.trim();

  return {
    plugins: [react(), tailwindcss(), setHmrPortFromPortPlugin, keepWebAppManifestOnPageOriginPlugin],
    // Empty env vars are common in Docker ARG defaults; ?? alone would yield base: "".
    base: assetBaseUrl ? assetBaseUrl : "/",
    server: {
      port: devPort,
      strictPort: true,
      host: true,
      fs: {
        allow: [import.meta.dirname, path.resolve(import.meta.dirname, "../pkg/grpc/actions/factories/templates")],
      },
      headers: !isProduction ? { "X-Robots-Tag": "noindex" } : undefined,
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
        // Admin JSON API (keep `/admin` itself on Vite for the React admin UI)
        "/admin/api": {
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
        "@/canvas": path.resolve(import.meta.dirname, "src/pages/canvas"),
        "@factory-templates": path.resolve(import.meta.dirname, "../pkg/grpc/actions/factories/templates"),
        "@runner/hosted_video_hosts.json": path.resolve(
          import.meta.dirname,
          "../pkg/components/runner/hosted_video_hosts.json",
        ),
        "@": path.resolve(import.meta.dirname, "src"),
      },
    },
    optimizeDeps: {
      include: ["@pierre/diffs/react"],
    },
    build: {
      target: "es2020",
      outDir: "../pkg/web/assets/dist", // emit assets to pkg/web/assets/dist
      emptyOutDir: true,
      sourcemap: true,
      manifest: false, // do not generate manifest.json
      rolldownOptions: {
        output: {
          codeSplitting: {
            groups: [
              {
                name: "monaco-editor",
                test: /monaco-editor/,
                includeDependenciesRecursively: false,
              },
            ],
          },
        },
      },
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
