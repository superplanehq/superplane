export const SSO_AUTH_ERROR_MESSAGES: Record<string, string> = {
  sso_not_available: "Single Sign-On is not available for this organization.",
  sso_failed: "Single Sign-On failed. Try again.",
  sso_access_denied: "You do not have access to this organization.",
  sso_account_conflict: "This email belongs to an existing account. Ask an administrator to verify the email domain.",
  sso_seat_limit: "This organization has no seat for a new member.",
  sso_not_licensed: "Single Sign-On is not licensed for a new member.",
};

export function ssoAuthErrorMessage(code: string | null): string | null {
  if (!code) {
    return null;
  }
  return SSO_AUTH_ERROR_MESSAGES[code] ?? null;
}
