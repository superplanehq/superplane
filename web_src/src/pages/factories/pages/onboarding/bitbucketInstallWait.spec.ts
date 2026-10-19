import { beforeEach, describe, expect, it } from "bun:test";

import { clearBitbucketInstallWait, readBitbucketInstallWait, writeBitbucketInstallWait } from "./bitbucketInstallWait";

const wait = { startedAt: 1_700_000_000_000, knownSlugs: ["acme-team"] };

describe("bitbucketInstallWait", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("stores the install wait for one organization", () => {
    writeBitbucketInstallWait("org-1", wait);

    expect(readBitbucketInstallWait("org-1")).toEqual(wait);
    expect(readBitbucketInstallWait("org-2")).toBeNull();
  });

  it("ignores a blank organization id", () => {
    writeBitbucketInstallWait("", wait);
    writeBitbucketInstallWait("   ", wait);

    expect(readBitbucketInstallWait("")).toBeNull();
    expect(readBitbucketInstallWait("   ")).toBeNull();
    expect(sessionStorage.length).toBe(0);
  });

  it("removes the stored wait", () => {
    writeBitbucketInstallWait("org-1", wait);
    clearBitbucketInstallWait("org-1");

    expect(readBitbucketInstallWait("org-1")).toBeNull();
  });

  it("ignores corrupt stored data", () => {
    writeBitbucketInstallWait("org-1", wait);
    const key = sessionStorage.key(0);
    expect(key).not.toBeNull();

    for (const raw of ["{", JSON.stringify({ startedAt: "now" }), JSON.stringify({ startedAt: 1, knownSlugs: [1] })]) {
      sessionStorage.setItem(key!, raw);
      expect(readBitbucketInstallWait("org-1")).toBeNull();
    }
  });

  it("keeps going when session storage rejects the write", () => {
    const storage = sessionStorage;
    const setItem = storage.setItem.bind(storage);
    Object.defineProperty(storage, "setItem", {
      configurable: true,
      value() {
        throw new DOMException("quota", "QuotaExceededError");
      },
    });

    try {
      expect(() => writeBitbucketInstallWait("org-1", wait)).not.toThrow();
      expect(readBitbucketInstallWait("org-1")).toBeNull();
    } finally {
      Object.defineProperty(storage, "setItem", {
        configurable: true,
        value: setItem,
      });
    }
  });

  it("keeps going when session storage rejects the read", () => {
    const storage = sessionStorage;
    const getItem = storage.getItem.bind(storage);
    Object.defineProperty(storage, "getItem", {
      configurable: true,
      value() {
        throw new DOMException("blocked", "SecurityError");
      },
    });

    try {
      expect(() => readBitbucketInstallWait("org-1")).not.toThrow();
      expect(readBitbucketInstallWait("org-1")).toBeNull();
    } finally {
      Object.defineProperty(storage, "getItem", {
        configurable: true,
        value: getItem,
      });
    }
  });
});
