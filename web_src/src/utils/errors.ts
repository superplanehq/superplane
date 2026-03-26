import type { GooglerpcStatus } from "@/api-client/types.gen";

/**
 * Extract error message from API error response
 * Handles the structure returned by @hey-api/client-fetch
 */
export function getApiErrorMessage(error: unknown, fallback = "An error occurred"): string {
  return (
    getNonEmptyString(error) ??
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
    return getApiErrorMessage(parsedBody, trimmedBody);
  } catch {
    return trimmedBody;
  }
}

export function getApiErrorCode(error: unknown): number | null {
  if (!error) {
    return null;
  }

  if (typeof error === "object" && "error" in error) {
    const errorObj = error.error;
    if (errorObj && typeof errorObj === "object" && "code" in errorObj) {
      const code = (errorObj as GooglerpcStatus).code;
      if (typeof code === "number") {
        return code;
      }
    }
  }

  if (typeof error === "object" && "code" in error) {
    const code = (error as GooglerpcStatus).code;
    if (typeof code === "number") {
      return code;
    }
  }

  return null;
}

function getNestedError(error: unknown): unknown {
  if (!error || typeof error !== "object" || !("error" in error)) {
    return null;
  }

  return error.error;
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
  return trimmed ? value : null;
}
