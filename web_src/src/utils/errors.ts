import type { GooglerpcStatus } from "@/api-client/types.gen";

/**
 * Extract error message from API error response
 * Handles the structure returned by @hey-api/client-fetch
 */
export function getApiErrorMessage(error: unknown, fallback = "An error occurred"): string {
  if (!error) {
    return fallback;
  }

  if (typeof error === "string" && error.trim()) {
    return error;
  }

  // Check if error has the structure { error: GooglerpcStatus }
  if (typeof error === "object" && "error" in error) {
    const errorObj = error.error;
    if (errorObj && typeof errorObj === "object" && "message" in errorObj) {
      const message = (errorObj as GooglerpcStatus).message;
      if (typeof message === "string" && message.trim()) {
        return message;
      }
    }
  }

  // Check if error itself is GooglerpcStatus
  if (typeof error === "object" && "message" in error) {
    const message = (error as GooglerpcStatus).message;
    if (typeof message === "string" && message.trim()) {
      return message;
    }
  }

  // Check if error is a standard Error object
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallback;
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
