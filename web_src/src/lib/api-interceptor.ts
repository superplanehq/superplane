import { ACCOUNT_BLOCKED_MESSAGE } from "@/lib/account-blocked";

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
      redirectUnauthorized();
      throw new Error("Unauthorized");
    }

    return response;
  };

  interceptorSetup = true;
};

function isAuthenticatedRequest(input: RequestInfo | URL): boolean {
  const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
  const path = url.split("?")[0];

  return path.includes("/api/") || path.endsWith("/account");
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
