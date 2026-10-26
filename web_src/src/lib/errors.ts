/**
 * Extract error message from API error response
 * Handles the structure returned by the generated OpenAPI client
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

/**
 * Extract the machine-readable reason from an API error.
 * The gateway returns it in a google.rpc.ErrorInfo entry of `details`.
 */
export function getApiErrorReason(error: unknown): string | null {
  const details = getErrorField(error, "details");
  if (!Array.isArray(details)) {
    return null;
  }

  for (const detail of details) {
    const reason = getErrorField(detail, "reason");
    if (getErrorField(detail, "domain") === "superplane.com" && typeof reason === "string" && reason) {
      return reason;
    }
  }

  return null;
}

function getErrorField(value: unknown, field: string): unknown {
  if (!value || typeof value !== "object" || !(field in value)) {
    return null;
  }

  return (value as Record<string, unknown>)[field];
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

  return getNonEmptyString((error as { message?: unknown }).message);
}

function getNonEmptyString(value: unknown): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  if (
    !trimmed ||
    looksLikeHtmlDocument(trimmed) ||
    looksLikeBrowserNetworkError(trimmed) ||
    looksLikeSanitizedInternalError(trimmed)
  ) {
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

function looksLikeSanitizedInternalError(value: string): boolean {
  return value.trim().toLowerCase() === "internal error";
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
