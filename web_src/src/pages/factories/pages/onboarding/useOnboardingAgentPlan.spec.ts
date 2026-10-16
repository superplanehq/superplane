import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { byokModelIds, useOnboardingAgentPlan } from "./useOnboardingAgentPlan";
import { useOnboardingAgentModel } from "./onboardingAgentModel";
import { agentRewriteFromPlan } from "./useOnboardingAgentPlan";

const keyModels = { ids: ["claude-sonnet-4-6", "claude-opus-5-5"], fetched: true, error: false };

vi.mock("@/hooks/useLLMModelAllowlists", () => ({
  useBYOKLLMModels: () => ({
    data: { candidates: keyModels.ids.map((id) => ({ id })) },
    isFetched: keyModels.fetched,
    isError: keyModels.error,
    refetch: vi.fn(),
  }),
}));

describe("byokModelIds", () => {
  it.each([
    { candidates: ["claude-sonnet-4-6", "claude-opus-5-5"], expected: ["claude-sonnet-4-6"] },
    { candidates: ["claude-opus-5-5"], expected: [] },
  ])("keeps only selected models available on the key (%j)", ({ candidates, expected }) => {
    expect(
      byokModelIds({
        selected: [{ id: "claude-sonnet-4-6" }],
        candidates: candidates.map((id) => ({ id })),
      }),
    ).toEqual(expected);
  });

  it("uses every model on the key when nothing is selected", () => {
    expect(byokModelIds({ selected: [], candidates: [{ id: "claude-opus-5-5" }] })).toEqual(["claude-opus-5-5"]);
    expect(byokModelIds(undefined)).toEqual([]);
  });
});

describe("useOnboardingAgentPlan", () => {
  beforeEach(() => {
    keyModels.ids = ["claude-sonnet-4-6", "claude-opus-5-5"];
    keyModels.fetched = true;
    keyModels.error = false;
    localStorage.clear();
  });
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
      planningModel: "claude-sonnet-4-6",
    });
  });

  it("requires a provider connection when no hosted agent is configured", () => {
    const { result } = renderHook(() => useOnboardingAgentPlan("org-1", new Set(), 0));

    expect(result.current.plan).toBeUndefined();
  });

  it("keeps the selected key model through a reload and applies it to every agent", () => {
    function useSetupModel() {
      const [selectedModel, selectModel] = useOnboardingAgentModel("factory-1");
      return {
        selectModel,
        ...useOnboardingAgentPlan("org-1", new Set(["claude"]), 5000, {
          preferOwnKey: true,
          selectedModel,
          provider: "openrouter",
          model: "hosted-only-model",
        }),
      };
    }
    const first = renderHook(useSetupModel);
    expect(first.result.current.modelPicker?.models).toEqual(keyModels.ids);
    act(() => first.result.current.selectModel("claude-opus-5-5"));
    first.unmount();
    const { result } = renderHook(useSetupModel);

    expect(result.current.modelPicker?.value).toBe("claude-opus-5-5");
    expect(
      agentRewriteFromPlan(result.current.plan!, {
        claude: { id: "key-1", name: "my-key", ready: true },
      }),
    ).toEqual({
      component: "runnerClaudeCode",
      model: "claude-opus-5-5",
      planningModel: "claude-opus-5-5",
      credentials: { source: "integration", name: "my-key" },
    });
  });

  it.each(["loading", "error", "empty", "unavailable"] as const)("blocks setup when the key models are %s", (state) => {
    keyModels.fetched = state !== "loading";
    keyModels.error = state === "error";
    if (state === "empty") keyModels.ids = [];
    const { result } = renderHook(() =>
      useOnboardingAgentPlan("org-1", new Set(["claude"]), 5000, {
        preferOwnKey: true,
        selectedModel: state === "unavailable" ? "not-on-this-key" : undefined,
      }),
    );

    expect(result.current.plan).toBeUndefined();
    expect(result.current.modelPicker?.value).toBe("");
  });
});
