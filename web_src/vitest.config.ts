import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";
import type { Plugin } from "vite";

/**
 * Vite plugin that provides empty stub modules for generated API-client files
 * (`sdk.gen`, `types.gen`) when they have not been generated yet (e.g. in the
 * verify / CI environments that do not run `make pb.gen`).  All named imports
 * will resolve to `undefined`; components call them only inside React-Query
 * hooks whose results are mocked at a higher level in the tests.
 */
function generatedApiClientStub(): Plugin {
  const VIRTUAL_PREFIX = "\0virtual:api-client-stub:";

  return {
    name: "generated-api-client-stub",
    enforce: "pre",
    resolveId(id) {
      // Match both alias form (@/api-client/sdk.gen) and relative form
      // (../api-client/sdk.gen, ../../api-client/sdk.gen, etc.)
      if (/api-client\/(sdk|types)\.gen(\.ts)?$/.test(id)) {
        return VIRTUAL_PREFIX + id;
      }
    },
    load(id) {
      if (id.startsWith(VIRTUAL_PREFIX)) {
        // Return an empty ESM module so all destructured named imports are
        // undefined.  Actual function calls happen lazily inside hooks and
        // React-Query queryFns, never at module initialisation time.
        return "export {};";
      }
    },
  };
}

export default defineConfig({
  plugins: [generatedApiClientStub(), react()],
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    coverage: {
      provider: "v8",
      reporter: ["text-summary"],
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
