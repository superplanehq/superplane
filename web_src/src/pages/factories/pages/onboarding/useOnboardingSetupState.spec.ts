import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { IntegrationId } from "./onboardingFixtures";
import { useOnboardingSetupState } from "./useOnboardingSetupState";

describe("useOnboardingSetupState", () => {
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

  it("marks the agent step ready when remaining credit is greater than zero", () => {
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected: new Set<IntegrationId>(),
        remainingCreditCents: 5000,
        simulateDiscovery: false,
      }),
    );

    expect(result.current.agentReady).toBe(true);
  });

  it("marks the agent step ready when OpenAI is connected and credit is empty", () => {
    const { result } = renderHook(() =>
      useOnboardingSetupState("Payments", {
        connected: new Set<IntegrationId>(["openai"]),
        remainingCreditCents: 0,
        simulateDiscovery: false,
      }),
    );

    expect(result.current.agentReady).toBe(true);
  });

  it("lets setup finish when OpenRouter is connected", () => {
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
    });

    expect(result.current.agentReady).toBe(true);
    expect(result.current.canFinish).toBe(true);
  });

  it("lets setup finish when Anthropic is connected or remaining credit is greater than zero", () => {
    const connected = new Set<IntegrationId>(["github", "claude"]);
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
    });

    expect(result.current.agentReady).toBe(true);
    expect(result.current.canFinish).toBe(true);
  });
});
