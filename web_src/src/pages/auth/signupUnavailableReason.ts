export type SignupUnavailableReason = "closed" | "waitlist" | null;

export const SIGNUP_DISABLED_AUTH_ERROR = "signup_disabled";

export const getSignupUnavailableReason = (
  showSignupUnavailable: boolean,
  signupsBlockedByEnvironment: boolean,
  hasConfiguredSignupWaitlist: boolean,
): SignupUnavailableReason => {
  if (!showSignupUnavailable) {
    return null;
  }

  if (!signupsBlockedByEnvironment && hasConfiguredSignupWaitlist) {
    return "waitlist";
  }

  return "closed";
};

export const isSignupDisabledErrorBody = (body: string): boolean => {
  try {
    const parsed = JSON.parse(body.trim()) as { error?: string };
    return parsed.error === SIGNUP_DISABLED_AUTH_ERROR;
  } catch {
    return false;
  }
};
