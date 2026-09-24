import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";
import type { ConfigurationField } from "@/api-client";
import { useComponent } from "@/hooks/useComponentData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { FEATURE_WORKSPACE_AGENT_RESOURCES } from "@/lib/experimentalFeatures";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";

import { HEADER_MCP_RESOURCE } from "../__fixtures__/agentResourceFixtures";
import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { PlanningReviewForm } from "./PlanningReviewForm";
import { PLANNING_REVIEW_DRAFT, type PlanningReviewDraft } from "./planningReviewMockup";

const useCanvasMock = vi.hoisted(() =>
  vi.fn((): { data: { metadata?: { factoryId?: string } } | undefined; isPending: boolean } => ({
    data: undefined,
    isPending: false,
  })),
);

vi.mock("@/hooks/useComponentData", () => ({
  useComponent: vi.fn(),
}));

vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: vi.fn(),
}));

vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: useCanvasMock,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: vi.fn(() => ({
    has: () => false,
    enabledExperimentalFeatures: [],
    isLoading: false,
  })),
}));

vi.mock("@/hooks/useFactoryAgentResources", () => ({
  useFactoryAgentResources: vi.fn(() => ({ data: [], isLoading: false, isError: false })),
}));

const superPlaneModelField: ConfigurationField = {
  name: "model",
  label: "Model",
  type: "hosted-model",
  placeholder: "Instance SuperPlane agent model",
  typeOptions: { hostedModel: { provider: HOSTED_MODEL_ALL_PROVIDERS } },
};

function selectableModel(provider: string, id: string) {
  return {
    source: { id: "hosted", name: "SuperPlane" },
    provider: { id: provider, name: provider },
    model: { id, name: id },
    key: `hosted::${provider}::${id}`,
    label: provider === "openrouter" ? id : `${provider}/${id}`,
  };
}

function claudeCodeDraft(model: string): PlanningReviewDraft {
  return {
    ...PLANNING_REVIEW_DRAFT,
    components: [
      {
        ...PLANNING_REVIEW_DRAFT.components[0],
        component: "runnerClaudeCode",
        configuration: {
          ...PLANNING_REVIEW_DRAFT.components[0].configuration,
          model,
        },
      },
    ],
  };
}

function byokAnthropicModel(id: string) {
  return {
    source: { id: "byok", name: "Your keys" },
    provider: { id: "anthropic", name: "Anthropic" },
    model: { id, name: id },
    key: `byok::anthropic::${id}`,
    label: `anthropic/${id}`,
  };
}

function superPlaneDraft(): PlanningReviewDraft {
  return {
    ...PLANNING_REVIEW_DRAFT,
    components: [
      {
        ...PLANNING_REVIEW_DRAFT.components[0],
        component: "runnerSuperPlane",
        configuration: {
          ...PLANNING_REVIEW_DRAFT.components[0].configuration,
          model: "",
        },
      },
    ],
  };
}

function renderForm(
  draft: PlanningReviewDraft,
  props: {
    onChange?: (draft: PlanningReviewDraft) => void;
    showVisualEvidenceSetting?: boolean;
    factoryId?: string;
    factoryKey?: string;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <PlanningReviewForm
          draft={draft}
          onChange={props.onChange ?? vi.fn()}
          organizationId="org-1"
          factoryId={props.factoryId}
          factoryKey={props.factoryKey}
          showVisualEvidenceSetting={props.showVisualEvidenceSetting}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PlanningReviewForm model options", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  beforeEach(() => {
    useCanvasMock.mockReset();
    useCanvasMock.mockReturnValue({ data: undefined, isPending: false });
    vi.mocked(useComponent).mockReturnValue({ data: undefined } as ReturnType<typeof useComponent>);
    vi.mocked(useSelectableLLMModels).mockReturnValue({
      data: [
        selectableModel("openrouter", "qwen/qwen3.7-max"),
        selectableModel("anthropic", "claude-opus-5"),
        selectableModel("openrouter", "moonshotai/kimi-k2.6"),
      ],
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useSelectableLLMModels>);
    vi.mocked(useOrganizationWorkspaceUsage).mockReturnValue({
      data: { defaultHostedProvider: "openrouter", defaultHostedModel: "qwen/qwen3.7-max" },
      isLoading: false,
    } as unknown as ReturnType<typeof useOrganizationWorkspaceUsage>);
    vi.mocked(useExperimentalFeature).mockReturnValue({
      has: () => false,
      enabledExperimentalFeatures: [],
      isLoading: false,
    });
    vi.mocked(useFactoryAgentResources).mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useFactoryAgentResources>);
  });

  it("lists hosted SuperPlane models when the automation agent is Run SuperPlane Agent", async () => {
    const user = userEvent.setup();
    vi.mocked(useComponent).mockReturnValue({
      data: { name: "runnerSuperPlane", configuration: [superPlaneModelField] },
    } as ReturnType<typeof useComponent>);

    renderForm(superPlaneDraft());

    expect(screen.getByText("Model used")).toBeInTheDocument();
    await user.click(screen.getByTestId("field-model-hosted-model"));
    await user.hover(screen.getByTestId("field-model-hosted-model-list"));
    expect(await screen.findByRole("menuitem", { name: "qwen/qwen3.7-max" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "anthropic/claude-opus-5" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "moonshotai/kimi-k2.6" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Claude Sonnet" })).not.toBeInTheDocument();
  });

  it("lists hosted SuperPlane models from the runner fallback when the catalog is empty", async () => {
    const user = userEvent.setup();

    renderForm(superPlaneDraft());

    await user.click(screen.getByTestId("field-model-hosted-model"));
    await user.hover(screen.getByTestId("field-model-hosted-model-list"));
    expect(await screen.findByRole("menuitem", { name: "qwen/qwen3.7-max" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Claude Sonnet" })).not.toBeInTheDocument();
  });

  it("keeps Claude aliases when the agent is not Run SuperPlane Agent", async () => {
    const user = userEvent.setup();
    renderForm(PLANNING_REVIEW_DRAFT);

    await user.click(screen.getByRole("combobox"));
    expect(screen.getByRole("option", { name: "Claude Sonnet" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Claude Opus" })).toBeInTheDocument();
    expect(screen.queryByTestId("field-model-hosted-model")).not.toBeInTheDocument();
  });

  it("replaces the sonnet alias with a model from the organization key", () => {
    const onChange = vi.fn();
    vi.mocked(useComponent).mockReturnValue({
      data: {
        name: "runnerClaudeCode",
        configuration: [
          {
            name: "model",
            label: "Model",
            type: "hosted-model",
            typeOptions: { hostedModel: { provider: "anthropic" } },
          },
        ],
      },
    } as ReturnType<typeof useComponent>);
    vi.mocked(useSelectableLLMModels).mockReturnValue({
      data: [byokAnthropicModel("claude-opus-4-6"), byokAnthropicModel("claude-sonnet-4-6")],
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useSelectableLLMModels>);

    renderForm(claudeCodeDraft("sonnet"), { onChange });

    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("anthropic/claude-sonnet-4-6");
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({
        components: [
          expect.objectContaining({
            configuration: expect.objectContaining({ model: "claude-sonnet-4-6" }),
          }),
        ],
      }),
    );
  });

  it("keeps an explicit model that is not a Claude alias", () => {
    const onChange = vi.fn();
    vi.mocked(useComponent).mockReturnValue({
      data: {
        name: "runnerClaudeCode",
        configuration: [
          {
            name: "model",
            label: "Model",
            type: "hosted-model",
            typeOptions: { hostedModel: { provider: "anthropic" } },
          },
        ],
      },
    } as ReturnType<typeof useComponent>);
    vi.mocked(useSelectableLLMModels).mockReturnValue({
      data: [byokAnthropicModel("claude-sonnet-4-6")],
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useSelectableLLMModels>);

    renderForm(claudeCodeDraft("claude-opus-4-7"), { onChange });

    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("claude-opus-4-7");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("shows and saves the visual evidence setting when enabled for the automation", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    renderForm(PLANNING_REVIEW_DRAFT, { onChange, showVisualEvidenceSetting: true });

    const settings = screen.getByTestId("planning-review-settings");
    expect(settings.className).toContain("grid-cols-3");
    const toggle = screen.getByRole("switch", { name: "Include visual evidence" });
    expect(toggle).not.toBeChecked();
    await user.click(toggle);
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({
        components: [
          expect.objectContaining({
            configuration: expect.objectContaining({ includeVisualEvidence: true }),
          }),
        ],
      }),
    );
  });

  it("hides the visual evidence setting for other automations", () => {
    renderForm(PLANNING_REVIEW_DRAFT);

    expect(screen.queryByRole("switch", { name: "Include visual evidence" })).not.toBeInTheDocument();
    expect(screen.getByTestId("planning-review-settings").className).toContain("grid-cols-2");
  });

  it("writes disabledAgentResourceIds when a resource is turned off", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    vi.mocked(useExperimentalFeature).mockReturnValue({
      has: (feature: string) => feature === FEATURE_WORKSPACE_AGENT_RESOURCES,
      enabledExperimentalFeatures: [FEATURE_WORKSPACE_AGENT_RESOURCES],
      isLoading: false,
    });
    vi.mocked(useFactoryAgentResources).mockImplementation((_org, _factory, kind) => {
      if (kind === "KIND_SKILL") {
        return { data: [], isLoading: false, isError: false } as unknown as ReturnType<typeof useFactoryAgentResources>;
      }
      return { data: [HEADER_MCP_RESOURCE], isLoading: false, isError: false } as unknown as ReturnType<
        typeof useFactoryAgentResources
      >;
    });

    renderForm(PLANNING_REVIEW_DRAFT, {
      onChange,
      factoryId: PRIMARY_FACTORY_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
    });

    await user.click(screen.getByTestId(`planning-review-resource-${HEADER_MCP_RESOURCE.id}`));
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({
        components: [
          expect.objectContaining({
            configuration: expect.objectContaining({ disabledAgentResourceIds: [HEADER_MCP_RESOURCE.id] }),
          }),
        ],
      }),
    );
  });
});
