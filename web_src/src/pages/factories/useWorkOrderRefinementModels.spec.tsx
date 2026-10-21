import { cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import type { ConfigurationField } from "@/api-client";
import { useWorkOrderRefinementModels } from "./useWorkOrderRefinementModels";

const state = vi.hoisted(() => ({
  component: "runnerSuperPlane",
  model: "anthropic::claude-sonnet-4-6",
  provider: "all",
  models: [
    {
      source: { id: "hosted", name: "SuperPlane" },
      provider: { id: "anthropic", name: "Anthropic" },
      model: { id: "claude-sonnet-4-6", name: "Claude Sonnet" },
      key: "hosted::anthropic::claude-sonnet-4-6",
      label: "Claude Sonnet",
    },
    {
      source: { id: "hosted", name: "SuperPlane" },
      provider: { id: "openai", name: "OpenAI" },
      model: { id: "gpt-5", name: "GPT-5" },
      key: "hosted::openai::gpt-5",
      label: "GPT-5",
    },
  ],
}));
const modelQuery = vi.hoisted(() => vi.fn());
vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryAutomations: () => ({ data: [{ id: "backlog", name: "Backlog" }], isPending: false }),
}));
vi.mock("./pages/useColumnCanvasAgentEditor", () => ({
  useColumnCanvasAgentEditor: () => ({
    agentNode: { id: "refine-task", component: state.component, configuration: { model: state.model } },
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useComponentData", () => ({
  useComponent: () => ({
    data: {
      configuration: [
        {
          name: "model",
          type: "hosted-model",
          typeOptions: { hostedModel: { provider: state.provider } },
        } satisfies ConfigurationField,
      ],
    },
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useSelectableLLMModels", () => ({ useSelectableLLMModels: modelQuery }));
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    data: { defaultHostedProvider: "openai", defaultHostedModel: "gpt-5" },
    isLoading: false,
  }),
}));

describe("Manual task refinement models", () => {
  beforeEach(() => {
    state.component = "runnerSuperPlane";
    state.model = "anthropic::claude-sonnet-4-6";
    state.provider = "all";
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("uses the Backlog hosted model and the hosted list in this workspace", () => {
    modelQuery.mockReturnValue({ data: state.models, isLoading: false });
    const { result } = renderHook(() => useWorkOrderRefinementModels("org-1", "factory-1"));
    expect(result.current.defaultModel).toBe("hosted::anthropic::claude-sonnet-4-6");
    expect(result.current.options.map((option) => option.value)).toEqual(state.models.map((model) => model.key));
    expect(modelQuery).toHaveBeenCalledWith("org-1", { factoryId: "factory-1", sources: ["hosted"], enabled: true });
  });

  it("uses only the refine node provider for a BYOK runner", () => {
    state.component = "runnerClaudeCode";
    state.model = "claude-sonnet-4-6";
    state.provider = "anthropic";
    modelQuery.mockReturnValue({
      data: state.models.map((model) => ({
        ...model,
        source: { id: "byok", name: "Your keys" },
        key: model.key.replace("hosted::", "byok::"),
      })),
      isLoading: false,
    });
    const { result } = renderHook(() => useWorkOrderRefinementModels("org-1", "factory-1"));
    expect(result.current.defaultModel).toBe("claude-sonnet-4-6");
    expect(result.current.options.map((option) => option.value)).toEqual(["claude-sonnet-4-6"]);
    expect(modelQuery).toHaveBeenCalledWith("org-1", { factoryId: "factory-1", sources: ["byok"], enabled: true });
  });
});
