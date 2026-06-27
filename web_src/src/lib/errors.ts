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
 * Recognizes `ReferenceError` messages whose missing identifier is short
 * enough to be a Vite/esbuild minified symbol (e.g. `oU`, `Gy`, `vl`).
 *
 * Our production bundle ships as a single ES module: every top-level
 * binding is hoisted in the same lexical scope, so a "real" missing
 * reference to one of those minified symbols would crash the first render
 * for every user. In practice these reports come from third-party scripts
 * — Safari/iOS content blockers, accessibility overlays, and similar
 * browser extensions — that inject or rewrite code in the page and end up
 * evaluating expressions outside the bundle's module scope.
 *
 * Real bugs in our source surface with descriptive identifier names
 * (because the unminified symbol survives in the runtime error message),
 * so the heuristic only matches 1–3 character identifiers.
 */
export function looksLikeMinifiedReferenceError(value: string): boolean {
  const normalized = value.trim();

  return MINIFIED_SAFARI_REFERENCE_ERROR.test(normalized) || MINIFIED_V8_REFERENCE_ERROR.test(normalized);
}

// Safari / WebKit: "Can't find variable: oU"
const MINIFIED_SAFARI_REFERENCE_ERROR = /^(?:ReferenceError:\s+)?Can't find variable:\s+[A-Za-z_$][A-Za-z0-9_$]{0,2}$/;

// Chrome / V8 and Firefox: "oU is not defined" (optionally prefixed with "ReferenceError: ")
const MINIFIED_V8_REFERENCE_ERROR = /^(?:ReferenceError:\s+)?[A-Za-z_$][A-Za-z0-9_$]{0,2} is not defined$/;
