import { describe, expect, it } from "vitest";

import { isHostedSentryInstallAction } from "./useSentryIntakeSetup";

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
