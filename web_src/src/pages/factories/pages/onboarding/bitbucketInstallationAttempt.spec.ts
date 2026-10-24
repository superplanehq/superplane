import { beforeEach, describe, expect, it } from "bun:test";

import {
  clearBitbucketInstallationAttempt,
  readBitbucketInstallationAttempt,
  writeBitbucketInstallationAttempt,
} from "./bitbucketInstallationAttempt";

describe("bitbucketInstallationAttempt", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("restores the attempt for one organization only", () => {
    const attempt = { startedAt: Date.now(), knownSlugs: ["acme"] };
    writeBitbucketInstallationAttempt("org-1", attempt);

    expect(readBitbucketInstallationAttempt("org-1")).toEqual(attempt);
    expect(readBitbucketInstallationAttempt("org-2")).toBeNull();
  });

  it("removes the stored attempt", () => {
    writeBitbucketInstallationAttempt("org-1", { startedAt: Date.now(), knownSlugs: [] });
    clearBitbucketInstallationAttempt("org-1");

    expect(readBitbucketInstallationAttempt("org-1")).toBeNull();
  });

  it.each([
    ["invalid JSON", "{not json"],
    ["a non-object", "42"],
    ["a null value", "null"],
    ["a missing start time", JSON.stringify({ knownSlugs: [] })],
    ["a non-numeric start time", JSON.stringify({ startedAt: "now", knownSlugs: [] })],
    ["non-string slugs", JSON.stringify({ startedAt: Date.now(), knownSlugs: [1] })],
    ["a missing slug list", JSON.stringify({ startedAt: Date.now() })],
  ])("returns null for %s", (_name, raw) => {
    sessionStorage.setItem("superplane:onboarding-bitbucket-install-attempt:org-1", raw);

    expect(readBitbucketInstallationAttempt("org-1")).toBeNull();
  });

  it("keeps going when session storage is blocked", () => {
    const storage = sessionStorage;
    const original = {
      getItem: storage.getItem.bind(storage),
      setItem: storage.setItem.bind(storage),
      removeItem: storage.removeItem.bind(storage),
    };
    const blocked = () => {
      throw new DOMException("blocked", "SecurityError");
    };
    for (const method of ["getItem", "setItem", "removeItem"] as const) {
      Object.defineProperty(storage, method, { configurable: true, value: blocked });
    }

    try {
      expect(() => writeBitbucketInstallationAttempt("org-1", { startedAt: 1, knownSlugs: [] })).not.toThrow();
      expect(readBitbucketInstallationAttempt("org-1")).toBeNull();
      expect(() => clearBitbucketInstallationAttempt("org-1")).not.toThrow();
    } finally {
      for (const method of ["getItem", "setItem", "removeItem"] as const) {
        Object.defineProperty(storage, method, { configurable: true, value: original[method] });
      }
    }
  });
});
