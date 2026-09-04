import { afterEach, describe, expect, it, vi } from "vitest";

import {
  consumeIntegrationSetupReturn,
  consumeIntegrationSetupReturnIfArrived,
  hasIntegrationSetupStay,
  peekIntegrationSetupReturn,
  rememberIntegrationSetupReturn,
} from "./integrationSetupReturn";

describe("integration setup return", () => {
  afterEach(() => {
    window.localStorage.clear();
    vi.useRealTimers();
  });

  it("stores and consumes a return path for an organization", () => {
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup");

    expect(peekIntegrationSetupReturn("org-1")).toBe("/org-1/workspaces/APP/setup");

    consumeIntegrationSetupReturn("org-1");
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();
  });

  it("consumes the marker only after the browser arrives on the stored page", () => {
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup?step=vcs&pick=newest");

    consumeIntegrationSetupReturnIfArrived("org-1", "/org-1/settings/integrations/abc");
    expect(peekIntegrationSetupReturn("org-1")).toBe("/org-1/workspaces/APP/setup?step=vcs&pick=newest");

    consumeIntegrationSetupReturnIfArrived("org-1", "/org-1/workspaces/APP/setup");
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();
  });

  it("detects the setupStay query", () => {
    expect(hasIntegrationSetupStay("setupStay=1")).toBe(true);
    expect(hasIntegrationSetupStay("?setupStay=1")).toBe(true);
    expect(hasIntegrationSetupStay("")).toBe(false);
  });

  it("returns the path regardless of the integration the provider redirects to", () => {
    // The legacy connect creates a new integration id during the round trip, so
    // the marker must not depend on any specific integration id.
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup");

    expect(peekIntegrationSetupReturn("org-1")).toBe("/org-1/workspaces/APP/setup");
  });

  it("rejects paths outside the organization", () => {
    rememberIntegrationSetupReturn("org-1", "/org-2/workspaces/APP/setup");
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();

    rememberIntegrationSetupReturn("org-1", "https://example.com");
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();
  });

  it("accepts the account onboarding route as an integration return", () => {
    rememberIntegrationSetupReturn("org-1", "/onboarding?attempt=attempt-1&step=vcs&pick=newest");

    expect(peekIntegrationSetupReturn("org-1")).toBe("/onboarding?attempt=attempt-1&step=vcs&pick=newest");
  });

  it("expires a return path after fifteen minutes", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-19T12:00:00Z"));
    rememberIntegrationSetupReturn("org-1", "/org-1/workspaces/APP/setup");

    vi.setSystemTime(new Date("2026-08-19T12:15:01Z"));
    expect(peekIntegrationSetupReturn("org-1")).toBeNull();
  });
});
