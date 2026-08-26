export const SIGNUP_REQUIRED_AUTH_ERROR = "signup_required";

export function isKnownAuthProvider(provider: string | null): provider is "google" | "github" {
  return provider === "google" || provider === "github";
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

  return "This account does not have a SuperPlane account.";
}

export function getSignupRequiredCreateHref(provider: string | null, redirectQuery: string): string {
  if (!isKnownAuthProvider(provider)) {
    return `/signup${redirectQuery}`;
  }

  const params = new URLSearchParams(redirectQuery.startsWith("?") ? redirectQuery.slice(1) : redirectQuery);
  params.set("signup", "true");
  return `/auth/${provider}?${params.toString()}`;
}
