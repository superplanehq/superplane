/**
 * Presentation for the single GitHub control on Account → Sign in methods.
 * Backend still stores SSO (`providers`) and PR credit (`linked_accounts`) apart.
 */

export type GithubIdentityState =
  | { kind: "none" }
  | { kind: "linked_only"; username: string }
  | { kind: "sso"; identity: string; creditUsername: string | null; split: boolean };

export function resolveGithubIdentityState(
  ssoIdentity: string | null | undefined,
  linkedUsername: string | null | undefined,
): GithubIdentityState {
  const ssoDisplay = (ssoIdentity ?? "").trim();
  const linkedDisplay = (linkedUsername ?? "").trim();
  const ssoKey = ssoDisplay.toLowerCase();
  const linkedKey = linkedDisplay.toLowerCase();

  if (!ssoKey && !linkedKey) {
    return { kind: "none" };
  }
  if (!ssoKey && linkedKey) {
    return { kind: "linked_only", username: linkedDisplay };
  }
  if (!ssoKey) {
    return { kind: "none" };
  }

  const split = Boolean(linkedKey && linkedKey !== ssoKey);
  return {
    kind: "sso",
    identity: ssoDisplay,
    creditUsername: split ? linkedDisplay : null,
    split,
  };
}
