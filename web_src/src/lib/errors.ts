import type { GooglerpcStatus } from "@/api-client/types.gen";

/**
 * Extract error message from API error response
 * Handles the structure returned by @hey-api/client-fetch
 */
export function getApiErrorMessage(error: unknown, fallback = "An error occurred"): string {
  return (
    getNonEmptyString(error) ??
    getStatusMessage(getResponseDataError(error)) ??
    getStatusMessage(getNestedError(error)) ??
    getStatusMessage(error) ??
    getNonEmptyString(error instanceof Error ? error.message : null) ??
    fallback
  );
}

export async function getResponseErrorMessage(response: Response, fallback = "An error occurred"): Promise<string> {
  const rawBody = await response.text();
  const trimmedBody = rawBody.trim();

  if (!trimmedBody) {
    return fallback;
  }

  try {
    const parsedBody = JSON.parse(trimmedBody) as unknown;
    return getApiErrorMessage(parsedBody, fallback);
  } catch {
    return getApiErrorMessage(trimmedBody, fallback);
  }
}

function getNestedError(error: unknown): unknown {
  if (!error || typeof error !== "object" || !("error" in error)) {
    return null;
  }

  return error.error;
}

function getResponseDataError(error: unknown): unknown {
  if (!error || typeof error !== "object" || !("response" in error)) {
    return null;
  }

  const response = error.response;
  if (!response || typeof response !== "object" || !("data" in response)) {
    return null;
  }

  return response.data;
}

function getStatusMessage(error: unknown): string | null {
  if (!error || typeof error !== "object" || !("message" in error)) {
    return null;
  }

  return getNonEmptyString((error as GooglerpcStatus).message);
}

function getNonEmptyString(value: unknown): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  if (!trimmed || looksLikeHtmlDocument(trimmed) || looksLikeBrowserNetworkError(trimmed)) {
    return null;
  }

  return trimmed;
}

function looksLikeHtmlDocument(value: string): boolean {
  const normalized = value.slice(0, 1024).trim().toLowerCase();

  return (
    normalized.startsWith("<!doctype html") ||
    normalized.startsWith("<html") ||
    (normalized.includes("<html") && normalized.includes("</html>"))
  );
}

function looksLikeBrowserNetworkError(value: string): boolean {
  const normalized = value.trim().toLowerCase();

  return (
    normalized === "failed to fetch" ||
    normalized === "load failed" ||
    normalized === "network request failed" ||
    normalized.includes("networkerror when attempting to fetch resource")
  );
}

/**
 * Recognizes `ReferenceError` messages that complain about a short, minified
 * identifier — the kind produced by Vite/esbuild's bundle (e.g. `Gy`, `vl`).
 *
 * Our application bundle is loaded as an ES module wrapped in a single
 * lexical scope, so a "real" reference to a missing minified identifier from
 * inside that bundle is structurally impossible: function declarations are
 * hoisted next to each other and would have crashed during initial render.
 * In practice these reports come from:
 *
 *  - Safari / iOS browser extensions or content-blockers that inject scripts
 *    into the page and corrupt the bundle's lexical scope.
 *  - Page-rewriting tools (Reader mode, accessibility overlays) that re-eval
 *    parts of the bundle in a different scope.
 *
 * Filtering them keeps the Sentry inbox actionable. Real bugs in our source
 * code surface with descriptive identifier names (because the unminified
 * symbol survives in the error message), so this pattern intentionally only
 * matches 1–3 character identifiers.
 */
export function looksLikeMinifiedReferenceError(value: string): boolean {
  const normalized = value.trim();

  // Safari / WebKit: "Can't find variable: Gy"
  // Chrome / V8:     "Gy is not defined"  (often prefixed with "ReferenceError: ")
  // Firefox:         "Gy is not defined"
  return (
    /^(?:ReferenceError: )?Can't find variable: [A-Za-z_$][A-Za-z0-9_$]{0,2}$/.test(normalized) ||
    /^(?:ReferenceError: )?[A-Za-z_$][A-Za-z0-9_$]{0,2} is not defined$/.test(normalized)
  );
}
