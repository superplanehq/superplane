import { ACCOUNT_BLOCKED_MESSAGE } from "@/lib/account-blocked";

const ACCOUNT_SESSION_PATHS = new Set(["/account", "/organizations", "/apps/install/preview", "/apps/install"]);

let interceptorSetup = false;

export const setupApiInterceptor = (): void => {
  if (interceptorSetup) return;

  const originalFetch = globalThis.fetch;

  globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    const response = await originalFetch(input, init);

    if (!isAuthenticatedRequest(input)) {
      return response;
    }

    if (await isBlockedAccountResponse(response)) {
      redirectBlockedAccount();
      throw new Error(ACCOUNT_BLOCKED_MESSAGE);
    }

    if (response.status === 401) {
      if (isAccountSessionProbe(input)) {
        return response;
      }

      redirectUnauthorized();
      throw new Error("Unauthorized");
    }

    return response;
  };

  interceptorSetup = true;
};

function requestPath(input: RequestInfo | URL): string {
  const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
  return new URL(url, "http://localhost").pathname;
}

function isAuthenticatedRequest(input: RequestInfo | URL): boolean {
  const path = requestPath(input);
  return path.includes("/api/") || path.startsWith("/account/") || ACCOUNT_SESSION_PATHS.has(path);
}

function isAccountSessionProbe(input: RequestInfo | URL): boolean {
  return requestPath(input) === "/account";
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
