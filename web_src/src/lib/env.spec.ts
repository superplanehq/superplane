import { isUsagePageForced } from "@/lib/env";
import { afterEach, describe, expect, it, vi } from "vitest";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("env", () => {
  it("reads the usage page override flag", () => {
    vi.stubEnv("VITE_FORCE_USAGE_PAGE", "true");

    expect(isUsagePageForced()).toBe(true);
  });
});
