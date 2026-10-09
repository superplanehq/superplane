import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

const { organizationsDescribeIntegration } = vi.hoisted(() => ({
  organizationsDescribeIntegration: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  organizationsDescribeIntegration,
}));

import { useOnboardingIntegrationSelections } from "./useOnboardingIntegrationSelections";

const onboarding = { vcsIntegrationId: "int-1", vcsProvider: "github" as const };

describe("useOnboardingIntegrationSelections", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    organizationsDescribeIntegration.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("marks the saved GitHub selection ready after a failed installation-name lookup recovers", async () => {
    organizationsDescribeIntegration
      .mockRejectedValueOnce(new Error("temporary"))
      .mockResolvedValueOnce({ data: { integration: { metadata: { name: "github-acme" } } } });

    const { result } = renderHook(() => useOnboardingIntegrationSelections("org-1", onboarding));

    await act(async () => {});
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000);
    });

    expect(result.current.selections.github).toEqual({ id: "int-1", name: "github-acme", ready: true });
    expect(result.current.connected.has("github")).toBe(true);
  });

  it("does not mark the saved selection ready while every lookup fails", async () => {
    organizationsDescribeIntegration.mockRejectedValue(new Error("temporary"));

    const { result } = renderHook(() => useOnboardingIntegrationSelections("org-1", onboarding));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000 + 2_000 + 4_000);
    });

    expect(result.current.selections.github).toEqual({ id: "int-1", name: "int-1", ready: false });
    expect(result.current.connected.has("github")).toBe(false);
  });

  it("retries the lookup when the window regains focus", async () => {
    organizationsDescribeIntegration.mockRejectedValue(new Error("temporary"));

    const { result } = renderHook(() => useOnboardingIntegrationSelections("org-1", onboarding));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000 + 2_000 + 4_000);
    });
    expect(result.current.selections.github?.ready).toBe(false);

    organizationsDescribeIntegration.mockResolvedValue({
      data: { integration: { metadata: { name: "github-acme" } } },
    });
    await act(async () => {
      window.dispatchEvent(new Event("focus"));
    });

    expect(result.current.selections.github).toEqual({ id: "int-1", name: "github-acme", ready: true });
    expect(result.current.connected.has("github")).toBe(true);
  });
});
