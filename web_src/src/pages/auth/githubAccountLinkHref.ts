import { linkedAccountConnectHref } from "@/lib/accountSettings";

export function getGitHubAccountConnectHref(): string {
  return linkedAccountConnectHref("github", "/onboarding");
}

export function githubOnboardingAuthErrorMessage(error: string | null | undefined): string | null {
  if (error === "signin_method_in_use") {
    return "This GitHub identity already belongs to another SuperPlane account. Delete that account first.";
  }
  if (error === "linked_account_in_use") {
    return "Another SuperPlane account already uses this GitHub account.";
  }
  if (error) {
    return "We could not connect GitHub. Try again.";
  }
  return null;
}
