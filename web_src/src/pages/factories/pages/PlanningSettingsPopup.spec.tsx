import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { TooltipProvider } from "@/ui/tooltip";

import { HEADER_MCP_RESOURCE } from "../__fixtures__/agentResourceFixtures";
import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { PlanningSettingsPopup } from "./PlanningSettingsPopup";
import { DEFAULT_PLANNING_SETTINGS, type PlanningDraftSettings } from "./planningSettingsModel";

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: vi.fn(() => ({
    has: () => false,
    enabledExperimentalFeatures: [],
    isLoading: false,
    organizationReady: true,
  })),
}));

vi.mock("@/hooks/useFactoryAgentResources", () => ({
  useFactoryAgentResources: vi.fn(() => ({ data: [], isLoading: false, isError: false })),
  useFactoryAgentResourceTools: vi.fn(() => ({ data: [], isLoading: false, isError: false })),
}));

function defaultAgentSlot(overrides: Partial<PlanningReviewAgentSlot> = {}): PlanningReviewAgentSlot {
  return {
    draft: PLANNING_REVIEW_DRAFT,
    organizationId: "org-1",
    onSave: vi.fn(),
    showVisualEvidenceSetting: false,
    ...overrides,
  };
}

function renderPopup(
  onSave = vi.fn(),
  settings: PlanningDraftSettings = DEFAULT_PLANNING_SETTINGS,
  options: { agent?: boolean | Partial<PlanningReviewAgentSlot> } = {},
) {
  const agentSlot = options.agent ? defaultAgentSlot(options.agent === true ? {} : options.agent) : undefined;
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PlanningSettingsPopup
              settings={settings}
              onSave={onSave}
              onClose={vi.fn()}
              fixed={false}
              agent={agentSlot}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onSave };
}

describe("PlanningSettingsPopup", () => {
  beforeEach(() => {
    vi.mocked(useExperimentalFeature).mockReturnValue({
      has: () => false,
      enabledExperimentalFeatures: [],
      isLoading: false,
      organizationReady: true,
    });
    vi.mocked(useFactoryAgentResources).mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useFactoryAgentResources>);
  });

  it("saves Planning and score toggles", async () => {
    const user = userEvent.setup();
    const { onSave } = renderPopup();

    expect(screen.getByTestId("planning-settings-health")).toHaveTextContent("Ready");
    expect(within(screen.getByTestId("planning-settings-clarity")).getByRole("switch")).not.toBeChecked();
    await user.click(within(screen.getByTestId("planning-settings-clarity")).getByRole("switch"));
    await user.click(screen.getByTestId("planning-settings-save"));

    expect(onSave).toHaveBeenCalledWith({
      enabled: true,
      clarity: true,
      confidence: true,
    });
  });

  it("disables score toggles when Planning is off and keeps stored flags", async () => {
    const user = userEvent.setup();
    const { onSave } = renderPopup(vi.fn(), {
      enabled: true,
      clarity: true,
      confidence: false,
    });

    await user.click(within(screen.getByTestId("planning-settings-enabled")).getByRole("switch"));
    expect(screen.getByTestId("planning-settings-health")).toHaveTextContent("Off");
    expect(within(screen.getByTestId("planning-settings-clarity")).getByRole("switch")).toBeDisabled();
    expect(within(screen.getByTestId("planning-settings-confidence")).getByRole("switch")).toBeDisabled();
    await user.click(screen.getByTestId("planning-settings-save"));

    expect(onSave).toHaveBeenCalledWith({
      enabled: false,
      clarity: true,
      confidence: false,
    });
  });

  it("keeps an unsaved draft when the host passes an equal settings object", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const popup = (settings: PlanningDraftSettings) => (
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <PlanningSettingsPopup settings={settings} onSave={onSave} onClose={vi.fn()} fixed={false} />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );
    const { rerender } = render(popup({ ...DEFAULT_PLANNING_SETTINGS }));

    await user.click(within(screen.getByTestId("planning-settings-clarity")).getByRole("switch"));
    rerender(popup({ ...DEFAULT_PLANNING_SETTINGS }));

    expect(within(screen.getByTestId("planning-settings-clarity")).getByRole("switch")).toBeChecked();
  });

  it("shows General, Agent, and Automation tabs when an agent exists", () => {
    renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { agent: true });

    expect(screen.getByTestId("planning-settings-tab-general")).toBeInTheDocument();
    expect(screen.getByTestId("planning-settings-tab-agent")).toBeInTheDocument();
    expect(screen.getByTestId("planning-settings-tab-automation")).toBeInTheDocument();
  });

  it("shows workspace resources on the Agent tab when the workspace ids are set", async () => {
    const user = userEvent.setup();
    vi.mocked(useExperimentalFeature).mockReturnValue({
      has: (feature: string) => feature === FEATURE_WORKSPACE_MCP || feature === FEATURE_WORKSPACE_SKILLS,
      enabledExperimentalFeatures: [FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS],
      isLoading: false,
      organizationReady: true,
    });
    vi.mocked(useFactoryAgentResources).mockImplementation((_org, _factory, kind) => {
      if (kind === "KIND_SKILL") {
        return { data: [], isLoading: false, isError: false } as unknown as ReturnType<typeof useFactoryAgentResources>;
      }
      return { data: [HEADER_MCP_RESOURCE], isLoading: false, isError: false } as unknown as ReturnType<
        typeof useFactoryAgentResources
      >;
    });

    renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, {
      agent: { factoryId: PRIMARY_FACTORY_ID, factoryKey: PRIMARY_FACTORY_KEY },
    });
    await user.click(screen.getByTestId("planning-settings-tab-agent"));

    expect(screen.getByTestId("planning-review-resources")).toBeInTheDocument();
    expect(screen.getByTestId(`planning-review-resource-${HEADER_MCP_RESOURCE.id}`)).toBeInTheDocument();
  });
});
