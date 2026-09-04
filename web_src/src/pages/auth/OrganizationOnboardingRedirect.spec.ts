import { describe, expect, it } from "vitest";

import { linkedAccountConnectHref } from "@/lib/accountSettings";

import { getGitHubAccountConnectHref, githubOnboardingAuthErrorMessage } from "./githubAccountLinkHref";

describe("getGitHubAccountConnectHref", () => {
  it("connects GitHub as a linked account before resuming onboarding", () => {
    expect(getGitHubAccountConnectHref()).toBe(linkedAccountConnectHref("github", "/onboarding"));
    expect(getGitHubAccountConnectHref()).toBe("/auth/github?intent=connect&redirect=%2Fonboarding");
  });
});

describe("githubOnboardingAuthErrorMessage", () => {
  it("explains when the GitHub identity is already a sign-in method", () => {
    expect(githubOnboardingAuthErrorMessage("signin_method_in_use")).toBe(
      "This GitHub identity already belongs to another SuperPlane account. Delete that account first.",
    );
  });

  it("explains when another account already linked this GitHub identity", () => {
    expect(githubOnboardingAuthErrorMessage("linked_account_in_use")).toBe(
      "Another SuperPlane account already uses this GitHub account.",
    );
  });

  it("gives a recovery message for an unknown auth error", () => {
    expect(githubOnboardingAuthErrorMessage("unknown")).toBe("We could not connect GitHub. Try again.");
    expect(githubOnboardingAuthErrorMessage(null)).toBeNull();
  });
});
