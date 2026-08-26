import { describe, expect, it } from "vitest";

import {
  getLogoutHref,
  getSignupRequiredAccountBody,
  getSignupRequiredCreateHref,
  isKnownAuthProvider,
  shouldShowSignupRequiredPrompt,
} from "./signupRequiredPrompt";

describe("getSignupRequiredAccountBody", () => {
  it("names the Google account", () => {
    expect(getSignupRequiredAccountBody("google")).toBe("This Google account does not have a SuperPlane account.");
  });

  it("names the GitHub account", () => {
    expect(getSignupRequiredAccountBody("github")).toBe("This GitHub account does not have a SuperPlane account.");
  });

  it("uses generic copy when the provider is missing", () => {
    expect(getSignupRequiredAccountBody(null)).toBe("This account does not have a SuperPlane account.");
  });
});

describe("getSignupRequiredCreateHref", () => {
  it("continues Google OAuth with signup intent", () => {
    expect(getSignupRequiredCreateHref("google", "")).toBe("/auth/google?signup=true");
  });

  it("preserves the redirect query on the Google create path", () => {
    expect(getSignupRequiredCreateHref("google", "?redirect=%2Finvite%2Fabc")).toBe(
      "/auth/google?redirect=%2Finvite%2Fabc&signup=true",
    );
  });

  it("sends unknown providers to signup", () => {
    expect(getSignupRequiredCreateHref(null, "?redirect=%2Finvite%2Fabc")).toBe("/signup?redirect=%2Finvite%2Fabc");
  });
});

describe("shouldShowSignupRequiredPrompt", () => {
  it("shows the create prompt when signups are open", () => {
    expect(shouldShowSignupRequiredPrompt("signup_required", true)).toBe(true);
  });

  it("hides the create prompt when signups are closed or waitlisted", () => {
    expect(shouldShowSignupRequiredPrompt("signup_required", false)).toBe(false);
  });

  it("ignores other auth errors", () => {
    expect(shouldShowSignupRequiredPrompt("account_blocked", true)).toBe(false);
  });
});

describe("getLogoutHref", () => {
  it("keeps logout on login when no redirect is present", () => {
    expect(getLogoutHref("")).toBe("/logout");
  });

  it("preserves the redirect query on logout", () => {
    expect(getLogoutHref("?redirect=%2Finvite%2Fabc")).toBe("/logout?redirect=%2Finvite%2Fabc");
  });
});

describe("isKnownAuthProvider", () => {
  it("accepts google and github only", () => {
    expect(isKnownAuthProvider("google")).toBe(true);
    expect(isKnownAuthProvider("github")).toBe(true);
    expect(isKnownAuthProvider(null)).toBe(false);
    expect(isKnownAuthProvider("okta")).toBe(false);
  });
});
