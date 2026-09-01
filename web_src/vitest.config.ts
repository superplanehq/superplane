import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  server: {
    fs: {
      allow: [import.meta.dirname, path.resolve(import.meta.dirname, "../pkg/grpc/actions/factories/templates")],
    },
  },
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
      "@factory-templates": path.resolve(import.meta.dirname, "../pkg/grpc/actions/factories/templates"),
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
});
