import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import type { IntegrationId } from "./onboardingFixtures";
import { useOnboardingSetupState } from "./useOnboardingSetupState";

describe("useOnboardingSetupState", () => {
  beforeEach(() => {
    sessionStorage.removeItem("superplane.onboarding.keyProvider");
  });

  it("uses real connected state and completes discovery without fixture counts", () => {
    const connected = new Set<IntegrationId>(["github", "claude"]);
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected,
        simulateDiscovery: false,
      }),
    );

    act(() => {
      result.current.selectVcsHost("github");
      result.current.selectRepo("acme/payments");
    });
    act(() => result.current.startIssuesDiscovery());

    expect(result.current.vcsReady).toBe(true);
    expect(result.current.repoReady).toBe(true);
    expect(result.current.issuesDiscovered).toBe(true);
    expect(result.current.issuesChoice).toBe("vcs");
    expect(result.current.issueCount).toBeUndefined();
  });

  it("marks the agent step ready when SuperPlane agent has remaining credit", () => {
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected: new Set<IntegrationId>(),
        remainingCreditCents: 5000,
        simulateDiscovery: false,
      }),
    );

    expect(result.current.keyProvider).toBeNull();
    expect(result.current.agentReady).toBe(true);
  });

  it("does not mark the agent step ready from a connected key without BYOK selection", () => {
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected: new Set<IntegrationId>(["openai"]),
        remainingCreditCents: 0,
        simulateDiscovery: false,
      }),
    );

    expect(result.current.agentReady).toBe(false);
  });

  it("marks the agent step ready after the user selects a connected BYOK provider", () => {
    const connected = new Set<IntegrationId>(["github", "openrouter"]);
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected,
        remainingCreditCents: 0,
        simulateDiscovery: false,
      }),
    );

    act(() => {
      result.current.selectVcsHost("github");
      result.current.selectRepo("acme/payments");
      result.current.setKeyProvider("openrouter");
    });

    expect(result.current.agentReady).toBe(true);
    expect(result.current.canFinish).toBe(true);
  });

  it("lets setup finish on SuperPlane agent credit without a connected key", () => {
    const connected = new Set<IntegrationId>(["github"]);
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected,
        remainingCreditCents: 5000,
        simulateDiscovery: false,
      }),
    );

    act(() => {
      result.current.selectVcsHost("github");
      result.current.selectRepo("acme/payments");
    });

    expect(result.current.agentReady).toBe(true);
    expect(result.current.canFinish).toBe(true);
  });
});
