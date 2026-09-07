import { describe, expect, it } from "vitest";

import { resolveGithubIdentityState } from "./accountGithubIdentity";

describe("resolveGithubIdentityState", () => {
  it("returns none when neither identity exists", () => {
    expect(resolveGithubIdentityState(null, null)).toEqual({ kind: "none" });
  });

  it("returns linked_only when only the credit link exists", () => {
    expect(resolveGithubIdentityState(null, "Ada")).toEqual({ kind: "linked_only", username: "Ada" });
  });

  it("returns sso without split when only SSO exists", () => {
    expect(resolveGithubIdentityState("ada", null)).toEqual({
      kind: "sso",
      identity: "ada",
      creditUsername: null,
      split: false,
    });
  });

  it("returns sso without split when SSO and credit share a login", () => {
    expect(resolveGithubIdentityState("Ada", "ada")).toEqual({
      kind: "sso",
      identity: "Ada",
      creditUsername: null,
      split: false,
    });
  });

  it("returns sso with split when the credit login differs", () => {
    expect(resolveGithubIdentityState("ada", "other")).toEqual({
      kind: "sso",
      identity: "ada",
      creditUsername: "other",
      split: true,
    });
  });
});
