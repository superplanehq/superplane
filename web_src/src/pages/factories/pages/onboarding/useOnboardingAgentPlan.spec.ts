import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { anthropicKeyModelIds, useOnboardingAgentPlan } from "./useOnboardingAgentPlan";

vi.mock("@/hooks/useHostedLLMModels", () => ({
  useHostedLLMModels: () => ({ data: { models: [] }, isFetched: true }),
}));

vi.mock("@/hooks/useLLMModelAllowlists", () => ({
  useBYOKLLMModels: () => ({
    data: { candidates: [{ id: "claude-sonnet-4-6" }, { id: "claude-opus-5-5" }] },
    isFetched: true,
  }),
}));

describe("anthropicKeyModelIds", () => {
  it("uses the selected models when the organization has a selection", () => {
    expect(
      anthropicKeyModelIds({ selected: [{ id: "claude-sonnet-4-6" }], candidates: [{ id: "claude-opus-5-5" }] }),
    ).toEqual(["claude-sonnet-4-6"]);
  });

  it("uses every model on the key when nothing is selected", () => {
    expect(anthropicKeyModelIds({ selected: [], candidates: [{ id: "claude-opus-5-5" }] })).toEqual([
      "claude-opus-5-5",
    ]);
    expect(anthropicKeyModelIds(undefined)).toEqual([]);
  });
});

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

  it("uses a connected provider key when the organization brings its own key", () => {
    const { result } = renderHook(() =>
      useOnboardingAgentPlan("org-1", new Set(["claude"]), 0, {
        provider: "anthropic",
        model: "claude-sonnet-4-6",
        preferOwnKey: true,
      }),
    );

    expect(result.current.plan).toMatchObject({
      providerId: "claude",
      credentialsSource: "integration",
      model: "claude-sonnet-4-6",
      planningModel: "claude-opus-5-5",
    });
  });

  it("requires a provider connection when no hosted agent is configured", () => {
    const { result } = renderHook(() => useOnboardingAgentPlan("org-1", new Set(), 0));

    expect(result.current.plan).toBeUndefined();
  });
});
