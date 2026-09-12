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
    isolate: false,
    include: [
      "src/lib/**/*.{spec,test}.{ts,tsx}",
      "src/pages/factories/**/*.{spec,test}.{ts,tsx}",
    ],
    setupFiles: ["./src/test/setup.ts", "./src/test/setup.shared.ts"],
  },
  resolve: {
    alias: {
      "@factory-templates": path.resolve(import.meta.dirname, "../pkg/grpc/actions/factories/templates"),
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
});
