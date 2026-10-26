import { ACCOUNT_BLOCKED_MESSAGE } from "@/lib/account-blocked";
import { isPublicFactoryLinePath } from "@/lib/publicFactoryLinePath";
import { client } from "@/api-client/client.gen";
import {
  isSupportSessionActive,
  isSupportSessionAllowedRequest,
  SUPPORT_SESSION_READ_ONLY_MESSAGE,
} from "@/lib/supportSession";
import { showInfoToast } from "@/lib/toast";

const SUPPORT_SESSION_TOAST_ID = "support-session-read-only";
const ACCOUNT_SESSION_PATHS = new Set(["/account", "/organizations"]);
const PUBLIC_LINE_GUEST_PROBES = new Set(["/organizations", "/account/experimental-features"]);

let interceptorFetch: typeof globalThis.fetch | undefined;
let clientErrorInterceptorInstalled = false;

export const setupApiInterceptor = (): void => {
  if (!clientErrorInterceptorInstalled) {
    client.interceptors.error.use((error, response) => {
      if (error instanceof Error) {
        return error;
      }

      const fields = typeof error === "object" && error !== null ? error : {};
      const message = "message" in fields && typeof fields.message === "string" ? fields.message : String(error);
      return Object.assign(new Error(message), fields, { status: response?.status });
    });
    clientErrorInterceptorInstalled = true;
  }

  if (globalThis.fetch === interceptorFetch) {
    return;
  }

  const originalFetch = globalThis.fetch;

  const nextFetch: typeof globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const supportSessionRejection = rejectSupportSessionWrite(input, init);
    if (supportSessionRejection) {
      return supportSessionRejection;
    }

    const response = await originalFetch(input, init);

    if (requestPath(input).includes("/api/v1/public/")) {
      return response;
    }

    if (!isAuthenticatedRequest(input)) {
      return response;
    }

    if (await isBlockedAccountResponse(response)) {
      redirectBlockedAccount();
      throw new Error(ACCOUNT_BLOCKED_MESSAGE);
    }

    if (response.status === 401) {
      if (skipUnauthorizedRedirect(input)) {
        return response;
      }

      redirectUnauthorized();
      throw new Error("Unauthorized");
    }

    return response;
  };

  globalThis.fetch = nextFetch;
  interceptorFetch = nextFetch;
};

function requestURL(input: RequestInfo | URL): string {
  return typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
}

function requestPath(input: RequestInfo | URL): string {
  return new URL(requestURL(input), "http://localhost").pathname;
}

function requestMethod(input: RequestInfo | URL, init?: RequestInit): string {
  if (init?.method) {
    return init.method;
  }
  return input instanceof Request ? input.method : "GET";
}

function isSameOriginRequest(input: RequestInfo | URL): boolean {
  try {
    const pageOrigin = new URL(window.location.href).origin;
    return new URL(requestURL(input), window.location.href).origin === pageOrigin;
  } catch {
    return false;
  }
}

// The backend rejects these writes too. Stopping them here avoids a
// round trip and tells the admin why the action did nothing.
function rejectSupportSessionWrite(input: RequestInfo | URL, init?: RequestInit): Response | null {
  if (!isSupportSessionActive() || !isSameOriginRequest(input)) {
    return null;
  }

  if (isSupportSessionAllowedRequest(requestMethod(input, init), requestPath(input))) {
    return null;
  }

  showInfoToast(SUPPORT_SESSION_READ_ONLY_MESSAGE, { id: SUPPORT_SESSION_TOAST_ID });
  return new Response(SUPPORT_SESSION_READ_ONLY_MESSAGE, {
    status: 403,
    headers: { "Content-Type": "text/plain" },
  });
}

function isAuthenticatedRequest(input: RequestInfo | URL): boolean {
  const path = requestPath(input);
  return path.includes("/api/") || path.startsWith("/account/") || ACCOUNT_SESSION_PATHS.has(path);
}

function skipUnauthorizedRedirect(input: RequestInfo | URL): boolean {
  const path = requestPath(input);
  if (path === "/account") {
    return true;
  }
  return isPublicFactoryLinePath(window.location.pathname) && PUBLIC_LINE_GUEST_PROBES.has(path);
}

function isAuthRoute(pathname: string): boolean {
  return pathname.startsWith("/login") || pathname.startsWith("/signup") || pathname.startsWith("/setup");
}

async function isBlockedAccountResponse(response: Response): Promise<boolean> {
  if (response.status !== 403) {
    return false;
  }

  try {
    return (await response.clone().text()).trim() === ACCOUNT_BLOCKED_MESSAGE;
  } catch {
    return false;
  }
}

function redirectBlockedAccount(): void {
  if (!isAuthRoute(window.location.pathname)) {
    window.location.href = "/login?auth_error=account_blocked";
  }
}

function redirectUnauthorized(): void {
  if (isAuthRoute(window.location.pathname)) {
    return;
  }

  const redirectTarget = `${window.location.pathname}${window.location.search}`;
  const redirectParam = encodeURIComponent(redirectTarget);
  window.location.href = `/login?redirect=${redirectParam}`;
}
