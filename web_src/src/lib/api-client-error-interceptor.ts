import { client } from "@/api-client/client.gen";

type FailureResponse = {
  status?: number;
  statusText?: string;
};

let interceptorRegistered = false;

export function setupApiClientErrorInterceptor(): void {
  if (interceptorRegistered) {
    return;
  }

  interceptorRegistered = true;
  client.interceptors.error.use((failure, response) => failureToError(failure, response));
}

function failureToError(failure: unknown, response?: FailureResponse): Error {
  if (failure instanceof Error) {
    return failure;
  }

  const error = new Error(failureMessage(failure, response));
  if (typeof response?.status === "number") {
    Object.assign(error, { status: response.status });
  }
  if (failure !== null && typeof failure === "object") {
    Object.assign(error, { error: failure });
  }
  return error;
}

function failureMessage(failure: unknown, response?: FailureResponse): string {
  if (typeof failure === "string") {
    const trimmed = failure.trim();
    if (trimmed) {
      return trimmed;
    }
  }

  if (failure !== null && typeof failure === "object" && "message" in failure) {
    const message = (failure as { message?: unknown }).message;
    if (typeof message === "string" && message.trim()) {
      return message.trim();
    }
  }

  const statusText = response?.statusText?.trim();
  if (statusText) {
    return statusText;
  }

  return "Request failed";
}
