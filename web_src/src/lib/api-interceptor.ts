import { ACCOUNT_BLOCKED_MESSAGE } from "@/lib/account-blocked";

const ACCOUNT_SESSION_PATHS = new Set(["/account", "/organizations"]);
/** Guests on a public line receive 401 here. A member API 401 still asks for login. */
const GUEST_SAFE_UNAUTHORIZED_PATHS = new Set(["/account", "/organizations", "/account/experimental-features"]);

let interceptorFetch: typeof globalThis.fetch | undefined;

export const setupApiInterceptor = (): void => {
  if (globalThis.fetch === interceptorFetch) {
    return;
  }

  const originalFetch = globalThis.fetch;

  const nextFetch: typeof globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
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
      if (GUEST_SAFE_UNAUTHORIZED_PATHS.has(requestPath(input))) {
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

function requestPath(input: RequestInfo | URL): string {
  const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
  return new URL(url, "http://localhost").pathname;
}

function isAuthenticatedRequest(input: RequestInfo | URL): boolean {
  const path = requestPath(input);
  return path.includes("/api/") || path.startsWith("/account/") || ACCOUNT_SESSION_PATHS.has(path);
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
