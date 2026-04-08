import { afterEach, describe, expect, it, vi } from "vitest";
import { isAgentReplEnabled, isUsagePageForced } from "@/lib/env";

afterEach(() => {
  vi.unstubAllEnvs();
  Reflect.deleteProperty(window, "SUPERPLANE_AGENT_ENABLED");
});

describe("env", () => {
  it("is false when the server flag is missing (e.g. tests or prod bundle without Go render)", () => {
    expect(isAgentReplEnabled()).toBe(false);
  });

  it("is true when the server flag is boolean true", () => {
    (window as Window & { SUPERPLANE_AGENT_ENABLED?: boolean }).SUPERPLANE_AGENT_ENABLED = true;

    expect(isAgentReplEnabled()).toBe(true);
  });

  it("is false when the server flag is boolean false", () => {
    (window as Window & { SUPERPLANE_AGENT_ENABLED?: boolean }).SUPERPLANE_AGENT_ENABLED = false;

    expect(isAgentReplEnabled()).toBe(false);
  });

  it("is false when the flag is a string (not a boolean)", () => {
    (window as unknown as { SUPERPLANE_AGENT_ENABLED: string }).SUPERPLANE_AGENT_ENABLED = "true";

    expect(isAgentReplEnabled()).toBe(false);
  });

  it("reads the usage page override flag", () => {
    vi.stubEnv("VITE_FORCE_USAGE_PAGE", "true");

    expect(isUsagePageForced()).toBe(true);
  });
});
