import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { useOnboardingAgentPlan } from "./useOnboardingAgentPlan";

vi.mock("@/hooks/useHostedLLMModels", () => ({
  useHostedLLMModels: () => ({ data: { models: [] }, isFetched: true }),
}));

describe("useOnboardingAgentPlan", () => {
  it("uses the hosted agent when credit is empty", () => {
    const { result } = renderHook(() =>
      useOnboardingAgentPlan("org-1", new Set(), 0, {
        provider: "anthropic",
        model: "claude-sonnet-4-6",
      }),
    );

    expect(result.current.remainingCreditCents).toBe(0);
    expect(result.current.plan).toEqual({
      component: "runnerSuperPlane",
      credentialsSource: "hosted",
      harness: "AGENT_HARNESS_SUPERPLANE",
      model: "",
      planningModel: "",
    });
  });

  it("requires a provider connection when no hosted agent is configured", () => {
    const { result } = renderHook(() => useOnboardingAgentPlan("org-1", new Set(), 0));

    expect(result.current.plan).toBeUndefined();
  });
});
