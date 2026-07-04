type AuthRedirectResponse = {
  redirectUrl?: unknown;
};

export const getAuthRedirectURL = async (response: Response) => {
  const contentType = response.headers.get("Content-Type") ?? "";
  if (!contentType.includes("application/json")) {
    return response.url || "/";
  }

  try {
    const body = (await response.clone().json()) as AuthRedirectResponse;
    if (typeof body.redirectUrl === "string" && body.redirectUrl.trim()) {
      return body.redirectUrl;
    }
  } catch {
    return response.url || "/";
  }

  return response.url || "/";
};

export const getWelcomeRedirectPath = (redirectURL: string, redirectTarget: string) => {
  try {
    const parsedURL = new URL(redirectURL || "/", window.location.origin);
    if (parsedURL.origin !== window.location.origin || parsedURL.pathname !== "/welcome") {
      return null;
    }

    if (!parsedURL.searchParams.has("redirect") && redirectTarget) {
      parsedURL.searchParams.set("redirect", redirectTarget);
    }

    return `${parsedURL.pathname}${parsedURL.search}`;
  } catch {
    return null;
  }
};
