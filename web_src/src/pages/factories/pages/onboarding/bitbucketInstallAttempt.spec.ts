import { beforeEach, describe, expect, it } from "bun:test";

import { bitbucketInstallationAttemptMs } from "./bitbucketOnboardingPoll";
import {
  clearBitbucketInstallAttempt,
  markBitbucketInstallAttempt,
  restoreBitbucketInstallAttempt,
} from "./bitbucketInstallAttempt";

const key = "superplane:bitbucket-install-attempt:org-1";

describe("bitbucketInstallAttempt", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("returns null when no attempt was started", () => {
    expect(restoreBitbucketInstallAttempt("org-1")).toBeNull();
  });

  it("restores a stored attempt with its original start time", () => {
    markBitbucketInstallAttempt("org-1", ["acme-team"]);
    const stored = JSON.parse(localStorage.getItem(key) ?? "{}") as { startedAt: number };

    const attempt = restoreBitbucketInstallAttempt("org-1");

    expect(attempt).toEqual({ startedAt: stored.startedAt, knownSlugs: ["acme-team"] });
  });

  it("removes an expired attempt instead of restoring it", () => {
    localStorage.setItem(
      key,
      JSON.stringify({ startedAt: Date.now() - bitbucketInstallationAttemptMs - 1, knownSlugs: [] }),
    );

    expect(restoreBitbucketInstallAttempt("org-1")).toBeNull();
    expect(localStorage.getItem(key)).toBeNull();
  });

  it("ignores a corrupt marker and keeps onboarding usable", () => {
    localStorage.setItem(key, "{not json");

    expect(restoreBitbucketInstallAttempt("org-1")).toBeNull();
  });

  it("replaces the marker when a new attempt starts", () => {
    markBitbucketInstallAttempt("org-1", ["old"]);
    markBitbucketInstallAttempt("org-1", ["new"]);

    expect(restoreBitbucketInstallAttempt("org-1")?.knownSlugs).toEqual(["new"]);
  });

  it("clears the marker when the attempt ends", () => {
    markBitbucketInstallAttempt("org-1", ["acme-team"]);
    clearBitbucketInstallAttempt("org-1");

    expect(localStorage.getItem(key)).toBeNull();
  });

  it("scopes the marker by organization", () => {
    markBitbucketInstallAttempt("org-1", ["acme-team"]);

    expect(restoreBitbucketInstallAttempt("org-2")).toBeNull();
    expect(restoreBitbucketInstallAttempt("org-1")).not.toBeNull();
  });
});
