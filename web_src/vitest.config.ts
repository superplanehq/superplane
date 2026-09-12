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
    environment: "happy-dom",
    environmentOptions: {
      happyDOM: {
        settings: {
          disableJavaScriptFileLoading: true,
          disableCSSFileLoading: true,
          disableIframePageLoading: true,
        },
      },
    },
    // happy-dom rejects with an Event whose target is SCRIPT when a
    // <script src> cannot load. Canvas pages inject those tags.
    onUnhandledError(error) {
      return !isHappyDomScriptLoadError(error);
    },
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

function isHappyDomScriptLoadError(error: unknown): boolean {
  if (!error || typeof error !== "object") {
    return false;
  }
  const target = (error as { target?: unknown }).target;
  if (typeof HTMLScriptElement !== "undefined" && target instanceof HTMLScriptElement) {
    return true;
  }
  return target === "SCRIPT";
}
