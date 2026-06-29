export type SignupUnavailableReason = "closed" | "waitlist" | null;

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
