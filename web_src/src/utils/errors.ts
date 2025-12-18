import type { GooglerpcStatus } from "@/api-client/types.gen";

/**
 * Extract error message from API error response
 * Handles the structure returned by @hey-api/client-fetch
 */
export function getApiErrorMessage(error: unknown, fallback = "An error occurred"): string {
  if (!error) {
    return fallback;
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
