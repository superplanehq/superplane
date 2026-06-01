import type { GooglerpcStatus } from "@/api-client/types.gen";

const TRANSIENT_HTTP_STATUSES = new Set([0, 401, 403, 408, 429, 502, 503, 504]);

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

function extractHttpStatus(error: unknown): number | undefined {
  if (!error || typeof error !== "object") {
    return undefined;
  }

  const candidate = error as { status?: unknown; response?: { status?: unknown } };
  if (typeof candidate.status === "number") {
    return candidate.status;
  }

  if (candidate.response && typeof candidate.response === "object" && typeof candidate.response.status === "number") {
    return candidate.response.status;
  }

  return undefined;
}

function looksLikeNetworkFailureError(error: unknown): boolean {
  if (error instanceof TypeError) {
    const message = error.message ?? "";
    if (!message) {
      return true;
    }
    return looksLikeBrowserNetworkError(message);
  }

  if (error && typeof error === "object") {
    const name = (error as { name?: unknown }).name;
    if (name === "AbortError" || name === "NetworkError") {
      return true;
    }
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string" && looksLikeBrowserNetworkError(message)) {
      return true;
    }
  }

  return false;
}

/**
 * Returns true when the error looks like a transient infrastructure or
 * connectivity failure rather than a real application bug. These should not
 * be reported as actionable errors for background/silent operations.
 *
 * Treated as transient:
 * - HTML payloads thrown by `@hey-api/client-fetch` when the response body is
 *   an HTML error page from a proxy / load balancer / auth redirect.
 * - `TypeError("Failed to fetch")` and other generic browser network errors.
 * - HTTP statuses {0, 401, 403, 408, 429, 502, 503, 504}.
 */
export function isTransientHttpError(error: unknown): boolean {
  if (typeof error === "string" && looksLikeHtmlDocument(error)) {
    return true;
  }

  if (looksLikeNetworkFailureError(error)) {
    return true;
  }

  const status = extractHttpStatus(error);
  if (status !== undefined && TRANSIENT_HTTP_STATUSES.has(status)) {
    return true;
  }

  return false;
}

/**
 * Produce a short, telemetry-safe summary of an arbitrary error value. The
 * goal is to avoid emitting full HTML response bodies (or other huge payloads)
 * as log/error titles, which causes noisy and ungrouped Sentry issues.
 */
export function summarizeError(error: unknown, maxLength = 200): string {
  if (error instanceof Error) {
    const status = extractHttpStatus(error);
    const base = error.message?.trim() || error.name || "Error";
    if (looksLikeHtmlDocument(base)) {
      return status !== undefined ? `HTTP ${status} (HTML response)` : "HTML response body";
    }
    const prefix = status !== undefined ? `HTTP ${status}: ` : "";
    return truncate(`${prefix}${base}`, maxLength);
  }

  if (typeof error === "string") {
    if (looksLikeHtmlDocument(error)) {
      return "HTML response body";
    }
    return truncate(error.trim(), maxLength);
  }

  const status = extractHttpStatus(error);
  if (status !== undefined) {
    return `HTTP ${status}`;
  }

  if (error && typeof error === "object") {
    try {
      return truncate(JSON.stringify(error), maxLength);
    } catch {
      return "Unknown error";
    }
  }

  return "Unknown error";
}

function truncate(value: string, maxLength: number): string {
  if (value.length <= maxLength) {
    return value;
  }
  return `${value.slice(0, Math.max(0, maxLength - 1))}…`;
}
