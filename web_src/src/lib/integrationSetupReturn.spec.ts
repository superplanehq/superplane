import { afterEach, describe, expect, it, vi } from "vitest";

import {
  consumeIntegrationSetupReturn,
  peekIntegrationSetupReturn,
  rememberIntegrationSetupReturn,
} from "./integrationSetupReturn";

describe("integration setup return", () => {
  afterEach(() => {
    window.localStorage.clear();
    vi.useRealTimers();
  });

  it("stores and consumes a return path for an integration", () => {
    rememberIntegrationSetupReturn("org-1", "integration-1", "/org-1/workspaces/APP/setup");

    expect(peekIntegrationSetupReturn("org-1", "integration-1")).toBe("/org-1/workspaces/APP/setup");

    consumeIntegrationSetupReturn("org-1", "integration-1");
    expect(peekIntegrationSetupReturn("org-1", "integration-1")).toBeNull();
  });

  it("rejects paths outside the organization", () => {
    rememberIntegrationSetupReturn("org-1", "integration-1", "/org-2/workspaces/APP/setup");
    rememberIntegrationSetupReturn("org-1", "integration-2", "https://example.com");

    expect(peekIntegrationSetupReturn("org-1", "integration-1")).toBeNull();
    expect(peekIntegrationSetupReturn("org-1", "integration-2")).toBeNull();
  });

  it("expires a return path after one hour", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-19T12:00:00Z"));
    rememberIntegrationSetupReturn("org-1", "integration-1", "/org-1/workspaces/APP/setup");

    vi.setSystemTime(new Date("2026-08-19T13:00:01Z"));
    expect(peekIntegrationSetupReturn("org-1", "integration-1")).toBeNull();
  });
});
