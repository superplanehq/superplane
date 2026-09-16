import { describe, expect, it } from "vitest";

import { isHostedSentryInstallAction, readySentryConnectionId } from "./useSentryIntakeSetup";

describe("readySentryConnectionId", () => {
  it("reuses the selected ready connection", () => {
    expect(
      readySentryConnectionId([{ metadata: { id: "sentry-a" } }, { metadata: { id: "sentry-b" } }], "sentry-b"),
    ).toBe("sentry-b");
  });

  it("falls back to the first ready connection", () => {
    expect(readySentryConnectionId([{ metadata: { id: "sentry-a" } }], "")).toBe("sentry-a");
  });

  it("is empty when SuperPlane has no Sentry connection", () => {
    expect(readySentryConnectionId([], "")).toBe("");
  });
});

describe("isHostedSentryInstallAction", () => {
  it("accepts the SuperPlane install URL and the Sentry app page", () => {
    expect(isHostedSentryInstallAction("/api/v1/sentry/app/install?state=abc")).toBe(true);
    expect(isHostedSentryInstallAction("https://sentry.io/sentry-apps/superplane/external-install/")).toBe(true);
  });

  it("rejects a private-app or missing URL", () => {
    expect(isHostedSentryInstallAction(undefined)).toBe(false);
    expect(isHostedSentryInstallAction("https://sentry.io/settings/developer-settings/")).toBe(false);
  });
});
