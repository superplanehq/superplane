import { lazy, type ComponentType, type LazyExoticComponent } from "react";

/**
 * Detects errors thrown when the browser fails to fetch a dynamically imported
 * module/chunk. This commonly happens after a new deploy: the user is on a
 * stale `index.html` that references chunks whose hashed filenames no longer
 * exist on the server (or whose import succeeds but the parent module fails to
 * locate the next chunk).
 *
 * The exact error message varies across browsers:
 *   - Chrome:  "Failed to fetch dynamically imported module: <url>"
 *   - Firefox: "error loading dynamically imported module"
 *   - Safari:  "Importing a module script failed."
 *   - Vite preload helper: "Unable to preload CSS for <url>"
 */
export function isChunkLoadError(error: unknown): boolean {
  if (!error) {
    return false;
  }

  const message = error instanceof Error ? error.message : String(error);
  if (!message) {
    return false;
  }

  const normalized = message.toLowerCase();

  return (
    normalized.includes("failed to fetch dynamically imported module") ||
    normalized.includes("error loading dynamically imported module") ||
    normalized.includes("importing a module script failed") ||
    normalized.includes("unable to preload css") ||
    normalized.includes("dynamically imported module")
  );
}

const RELOAD_FLAG = "superplane:lazy-reload-attempted";

function hasRecentlyReloaded(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  try {
    return window.sessionStorage.getItem(RELOAD_FLAG) === "true";
  } catch {
    return false;
  }
}

function markReloadAttempted(): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.sessionStorage.setItem(RELOAD_FLAG, "true");
  } catch {
    // sessionStorage may be unavailable (private mode quota, etc.); reload
    // protection is best-effort.
  }
}

function clearReloadFlag(): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.sessionStorage.removeItem(RELOAD_FLAG);
  } catch {
    // ignore
  }
}

function reloadPage(): void {
  if (typeof window === "undefined" || !window.location) {
    return;
  }

  window.location.reload();
}

/**
 * Wraps a dynamic import factory with resilience against transient or
 * post-deploy chunk load failures. Exported separately from `lazyWithReload`
 * so its behavior can be unit-tested without exercising React's `lazy`.
 *
 * Behavior:
 *   1. Call `factory()`.
 *   2. If it rejects with a chunk-load error, wait `retryDelayMs` and retry once.
 *   3. If the retry also rejects with a chunk-load error and the page has not
 *      already been reloaded in this session, reload the page (so the browser
 *      fetches the latest `index.html` with up-to-date chunk references) and
 *      return a never-resolving promise so any pending React `Suspense`
 *      boundary stays in its fallback state until the navigation occurs.
 *   4. Otherwise, re-throw the underlying error.
 */
export function loadModuleWithReload<T>(
  factory: () => Promise<T>,
  options: { retryDelayMs?: number } = {},
): Promise<T> {
  const retryDelayMs = options.retryDelayMs ?? 500;

  return factory().then(
    (mod) => {
      clearReloadFlag();
      return mod;
    },
    async (firstError: unknown) => {
      if (!isChunkLoadError(firstError)) {
        throw firstError;
      }

      await new Promise((resolve) => setTimeout(resolve, retryDelayMs));

      try {
        const mod = await factory();
        clearReloadFlag();
        return mod;
      } catch (secondError) {
        if (isChunkLoadError(secondError) && !hasRecentlyReloaded()) {
          markReloadAttempted();
          reloadPage();
          return new Promise<T>(() => {});
        }

        throw secondError;
      }
    },
  );
}

/**
 * Wraps `React.lazy` with the resilience strategy in `loadModuleWithReload`.
 * Use this in place of `React.lazy` for any code-split component to gracefully
 * recover from "Failed to fetch dynamically imported module" errors that occur
 * when a deploy invalidates chunk hashes referenced by a stale `index.html`.
 */
// `ComponentType<any>` mirrors React's own constraint on `React.lazy`. Using
// `unknown` here would break variance with most component prop types.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function lazyWithReload<T extends ComponentType<any>>(
  factory: () => Promise<{ default: T }>,
  options: { retryDelayMs?: number } = {},
): LazyExoticComponent<T> {
  return lazy(() => loadModuleWithReload(factory, options));
}
