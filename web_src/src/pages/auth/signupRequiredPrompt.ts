export const SIGNUP_REQUIRED_AUTH_ERROR = "signup_required";
const SIGNUP_REQUIRED_LEGACY_MESSAGE = "signup must be started from the signup page";

export function isKnownAuthProvider(provider: string | null): provider is "google" | "github" {
  return provider === "google" || provider === "github";
}

export function isEmailAuthProvider(provider: string | null): boolean {
  return provider === "email";
}

export function shouldShowSignupRequiredPrompt(authError: string | null, canSignup: boolean): boolean {
  return authError === SIGNUP_REQUIRED_AUTH_ERROR && canSignup;
}

export function getSignupRequiredAccountBody(provider: string | null): string {
  if (provider === "google") {
    return "This Google account does not have a SuperPlane account.";
  }

  if (provider === "github") {
    return "This GitHub account does not have a SuperPlane account.";
  }

  if (provider === "email") {
    return "This email does not have a SuperPlane account.";
  }

  return "This account does not have a SuperPlane account.";
}

export function getSignupRequiredCreateHref(provider: string | null, redirectQuery: string): string {
  if (isEmailAuthProvider(provider)) {
    return "";
  }

  if (!isKnownAuthProvider(provider)) {
    return `/signup${redirectQuery}`;
  }

  const params = new URLSearchParams(redirectQuery.startsWith("?") ? redirectQuery.slice(1) : redirectQuery);
  params.set("signup", "true");
  return `/auth/${provider}?${params.toString()}`;
}

export function isSignupRequiredErrorBody(body: string): boolean {
  const text = body.trim();
  if (text === SIGNUP_REQUIRED_LEGACY_MESSAGE) {
    return true;
  }

  try {
    const parsed = JSON.parse(text) as { error?: string };
    return parsed.error === SIGNUP_REQUIRED_AUTH_ERROR;
  } catch {
    return false;
  }
}

export function getLogoutHref(redirectQuery: string): string {
  return redirectQuery ? `/logout${redirectQuery}` : "/logout";
}
