import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";
import type { ConfigurationField } from "@/api-client";
import { useComponent } from "@/hooks/useComponentData";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";

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

function renderForm(draft: PlanningReviewDraft) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <PlanningReviewForm draft={draft} onChange={vi.fn()} organizationId="org-1" />
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
  });

  it("lists hosted SuperPlane models when the automation agent is Run SuperPlane Agent", async () => {
    const user = userEvent.setup();
    vi.mocked(useComponent).mockReturnValue({
      data: { name: "runnerSuperPlane", configuration: [superPlaneModelField] },
    } as ReturnType<typeof useComponent>);

    renderForm(superPlaneDraft());

    expect(screen.getByText("Model used")).toBeInTheDocument();
    await user.click(screen.getByTestId("field-model-hosted-model"));
    expect(screen.getByRole("option", { name: "qwen/qwen3.7-max" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "anthropic/claude-opus-5" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "moonshotai/kimi-k2.6" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Claude Sonnet" })).not.toBeInTheDocument();
  });

  it("lists hosted SuperPlane models from the runner fallback when the catalog is empty", async () => {
    const user = userEvent.setup();

    renderForm(superPlaneDraft());

    await user.click(screen.getByTestId("field-model-hosted-model"));
    expect(screen.getByRole("option", { name: "qwen/qwen3.7-max" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Claude Sonnet" })).not.toBeInTheDocument();
  });

  it("keeps Claude aliases when the agent is not Run SuperPlane Agent", async () => {
    const user = userEvent.setup();
    renderForm(PLANNING_REVIEW_DRAFT);

    await user.click(screen.getByRole("combobox"));
    expect(screen.getByRole("option", { name: "Claude Sonnet" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Claude Opus" })).toBeInTheDocument();
    expect(screen.queryByTestId("field-model-hosted-model")).not.toBeInTheDocument();
  });
});
