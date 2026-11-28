import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { TooltipProvider } from "@/ui/tooltip";

import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { PlanningSettingsPopup } from "./PlanningSettingsPopup";
import {
  DEFAULT_PLANNING_SETTINGS,
  type PlanningAutoStartLine,
  type PlanningDraftSettings,
} from "./planningSettingsModel";

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
    ...overrides,
  };
}

function renderPopup(
  onSave = vi.fn(),
  settings: PlanningDraftSettings = DEFAULT_PLANNING_SETTINGS,
  options: {
    agent?: boolean | Partial<PlanningReviewAgentSlot>;
    lines?: PlanningAutoStartLine[];
    onRetry?: () => void;
    onDelete?: () => Promise<void> | void;
    initialPath?: string;
  } = {},
) {
  const agentSlot = options.agent ? defaultAgentSlot(options.agent === true ? {} : options.agent) : undefined;
  const onClose = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const popup = (storedSettings: PlanningDraftSettings) => (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[options.initialPath ?? "/"]}>
        <ThemeProvider>
          <TooltipProvider>
            <PlanningSettingsPopup
              settings={storedSettings}
              lines={options.lines}
              onSave={onSave}
              onDelete={options.onDelete}
              onClose={onClose}
              onRetryAutomation={options.onRetry}
              fixed={false}
              agent={agentSlot}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(popup(settings));
  return { onSave, onClose, rerenderSettings: (next: PlanningDraftSettings) => view.rerender(popup(next)) };
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

  it("puts Save in an independent footer when deletion is unavailable", () => {
    renderPopup();

    const footers = screen.getByTestId("planning-settings").querySelectorAll("footer");
    expect(footers).toHaveLength(1);
    expect(within(footers[0]).getByTestId("planning-settings-save")).toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-footer")).not.toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-delete")).not.toBeInTheDocument();
  });

  it("puts Save and Delete in one shared footer when deletion is available", () => {
    renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { onDelete: vi.fn() });

    const footer = screen.getByTestId("column-automation-view-footer");
    expect(within(footer).getByTestId("planning-settings-save")).toBeInTheDocument();
    expect(within(footer).getByTestId("column-automation-view-delete")).toBeInTheDocument();
    expect(screen.getByTestId("planning-settings").querySelectorAll("footer")).toHaveLength(1);
  });

  it("hides Save during delete confirmation and restores it when confirmation is cancelled", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { onDelete });

    await user.click(screen.getByTestId("column-automation-view-delete"));
    const footer = screen.getByTestId("column-automation-view-footer");
    expect(within(footer).getByTestId("column-automation-view-delete-confirm")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-settings-save")).not.toBeInTheDocument();

    await user.click(within(footer).getByRole("button", { name: "Keep automation" }));
    expect(within(footer).getByTestId("planning-settings-save")).toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-delete-confirm")).not.toBeInTheDocument();
    expect(onDelete).not.toHaveBeenCalled();
  });

  it("saves Planning and keeps stored score flags", async () => {
    const user = userEvent.setup();
    const { onSave } = renderPopup();

    expect(screen.queryByTestId("planning-settings-clarity")).not.toBeInTheDocument();
    expect(screen.queryByTestId("planning-settings-confidence")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("planning-settings-save"));

    expect(onSave).toHaveBeenCalledWith({
      enabled: true,
      clarity: false,
      confidence: true,
      autoStartLineId: "",
    });
  });

  it("pauses new task planning from the menu and keeps stored score flags", async () => {
    const user = userEvent.setup();
    const { onSave, onClose } = renderPopup(vi.fn(), {
      enabled: true,
      clarity: true,
      confidence: false,
      autoStartLineId: "",
    });
    await user.click(screen.getByTestId("planning-settings-menu"));
    await user.click(screen.getByTestId("planning-settings-pause"));
    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith({
        enabled: false,
        clarity: true,
        confidence: false,
        autoStartLineId: "",
      }),
    );
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByTestId("planning-settings-paused")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-settings-enabled")).not.toBeInTheDocument();
  });

  it("keeps unsaved agent and Auto-start changes separate from pause and resume", async () => {
    const user = userEvent.setup();
    const saveAgent = vi.fn();
    const { onSave, rerenderSettings } = renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, {
      agent: { onSave: saveAgent },
      lines: [{ id: "main", name: "Main" }],
    });
    fireEvent.change(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" }), { target: { value: "8" } });
    await user.click(within(screen.getByTestId("planning-settings-auto-start")).getByRole("switch"));
    await user.click(screen.getByTestId("planning-settings-menu"));
    await user.click(screen.getByTestId("planning-settings-pause"));
    await waitFor(() => expect(onSave).toHaveBeenLastCalledWith({ ...DEFAULT_PLANNING_SETTINGS, enabled: false }));
    rerenderSettings({ ...DEFAULT_PLANNING_SETTINGS, enabled: false });
    expect(saveAgent).not.toHaveBeenCalled();
    expect(within(screen.getByTestId("planning-settings-auto-start")).getByRole("switch")).toBeChecked();
    expect(within(screen.getByTestId("planning-settings-auto-start")).getByRole("switch")).toBeDisabled();
    await user.click(screen.getByTestId("planning-settings-menu"));
    await user.click(screen.getByTestId("planning-settings-resume"));
    await waitFor(() => expect(onSave).toHaveBeenLastCalledWith(DEFAULT_PLANNING_SETTINGS));
    rerenderSettings(DEFAULT_PLANNING_SETTINGS);
    expect(screen.queryByTestId("planning-settings-paused")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("planning-settings-save"));
    await waitFor(() =>
      expect(onSave).toHaveBeenLastCalledWith({ ...DEFAULT_PLANNING_SETTINGS, autoStartLineId: "main" }),
    );
    expect(saveAgent).toHaveBeenCalledWith(
      expect.objectContaining({
        components: [
          expect.objectContaining({ concurrency: { ...PLANNING_REVIEW_DRAFT.components[0].concurrency, max: "8" } }),
        ],
      }),
    );
  });

  it("keeps planning active after a failed pause and supports retry", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockRejectedValueOnce(new Error("Network error")).mockResolvedValue(undefined);
    const { onClose } = renderPopup(onSave);
    await user.click(screen.getByTestId("planning-settings-menu"));
    await user.click(screen.getByTestId("planning-settings-pause"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Network error");
    expect(screen.queryByTestId("planning-settings-paused")).not.toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("planning-settings-menu"));
    await user.click(screen.getByTestId("planning-settings-pause"));
    await waitFor(() => expect(screen.getByTestId("planning-settings-paused")).toBeInTheDocument());
    expect(onSave).toHaveBeenCalledTimes(2);
  });

  it("enables Auto-start on the default line", async () => {
    const user = userEvent.setup();
    const lines = [
      { id: "line-implement", name: "implement" },
      { id: "line-hotfix", name: "hotfix" },
    ];
    const { onSave } = renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { lines });

    expect(screen.getByText(PLANNING_SETTINGS_COPY.autoStartReviewHelper)).toBeInTheDocument();
    await user.click(within(screen.getByTestId("planning-settings-auto-start")).getByRole("switch"));
    expect(screen.getByText(PLANNING_SETTINGS_COPY.autoStartReviewHelper)).toBeInTheDocument();
    await user.click(screen.getByTestId("planning-settings-save"));

    expect(onSave).toHaveBeenCalledWith({
      enabled: true,
      clarity: false,
      confidence: true,
      autoStartLineId: "line-implement",
    });
  });

  it("preserves an existing Auto-start destination without offering a selector", async () => {
    const user = userEvent.setup();
    const settings = { ...DEFAULT_PLANNING_SETTINGS, autoStartLineId: "line-hotfix" };
    const { onSave } = renderPopup(vi.fn(), settings, {
      lines: [
        { id: "line-implement", name: "implement" },
        { id: "line-hotfix", name: "hotfix" },
      ],
    });

    expect(screen.queryByTestId("planning-settings-auto-start-line")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("planning-settings-auto-start")).getByRole("switch")).toBeChecked();
    await user.click(screen.getByTestId("planning-settings-save"));
    expect(onSave).toHaveBeenCalledWith(settings);
  });

  it("disables Auto-start when Planning is off and keeps the line", () => {
    renderPopup(
      vi.fn(),
      { enabled: false, clarity: false, confidence: true, autoStartLineId: "line-implement" },
      { lines: [{ id: "line-implement", name: "implement" }] },
    );

    const toggle = within(screen.getByTestId("planning-settings-auto-start")).getByRole("switch");
    expect(toggle).toBeChecked();
    expect(toggle).toBeDisabled();
    expect(screen.getByText(PLANNING_SETTINGS_COPY.autoStartPlanningOffHelper)).toBeInTheDocument();
  });

  it("keeps Auto-start available when Confidence was stored off", () => {
    renderPopup(
      vi.fn(),
      { enabled: true, clarity: false, confidence: false, autoStartLineId: "" },
      {
        lines: [{ id: "line-implement", name: "implement" }],
      },
    );

    const toggle = within(screen.getByTestId("planning-settings-auto-start")).getByRole("switch");
    expect(toggle).not.toBeDisabled();
    expect(screen.getByText(PLANNING_SETTINGS_COPY.autoStartReviewHelper)).toBeInTheDocument();
  });

  it("combines Planning and agent settings in Configuration", async () => {
    const saveAgent = vi.fn();
    const user = userEvent.setup();
    const { onSave } = renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, {
      agent: { onSave: saveAgent },
      lines: [{ id: "main", name: "Main" }],
    });

    expect(screen.queryByTestId("planning-settings-health")).not.toBeInTheDocument();
    expect(screen.queryByTestId("planning-settings-tab-agent")).not.toBeInTheDocument();
    expect(screen.queryByTestId("planning-review-resources")).not.toBeInTheDocument();
    expect(screen.queryByTestId("planning-review-step-1")).not.toBeInTheDocument();
    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByRole("option", { name: "Claude Opus" }));
    await user.click(within(screen.getByTestId("planning-settings-auto-start")).getByRole("switch"));
    const concurrency = screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" });
    fireEvent.change(concurrency, { target: { value: "3" } });
    expect(saveAgent).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("planning-settings-save"));

    await waitFor(() => expect(onSave).toHaveBeenCalledWith({ ...DEFAULT_PLANNING_SETTINGS, autoStartLineId: "main" }));
    expect(saveAgent).toHaveBeenCalledWith({
      ...PLANNING_REVIEW_DRAFT,
      components: [
        {
          ...PLANNING_REVIEW_DRAFT.components[0],
          configuration: { ...PLANNING_REVIEW_DRAFT.components[0].configuration, model: "opus" },
          concurrency: { ...PLANNING_REVIEW_DRAFT.components[0].concurrency, max: "3" },
        },
      ],
    });
  });

  it("saves additional planning instructions separately from managed steps and supports clearing them", async () => {
    const saveAgent = vi.fn();
    renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { agent: { onSave: saveAgent } });
    const instructions = screen.getByRole("textbox", { name: "Additional instructions" });
    const value = "Include migration and rollout risks. Keep {{ example }} literal.";
    fireEvent.change(instructions, { target: { value } });
    fireEvent.blur(instructions);
    expect(saveAgent).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("planning-settings-save"));
    await waitFor(() =>
      expect(saveAgent).toHaveBeenCalledWith({
        ...PLANNING_REVIEW_DRAFT,
        components: [
          {
            ...PLANNING_REVIEW_DRAFT.components[0],
            configuration: { ...PLANNING_REVIEW_DRAFT.components[0].configuration, additionalInstructions: value },
          },
        ],
      }),
    );
    fireEvent.change(instructions, { target: { value: "" } });
    fireEvent.click(screen.getByTestId("planning-settings-save"));
    await waitFor(() =>
      expect(saveAgent).toHaveBeenLastCalledWith(
        expect.objectContaining({
          components: [
            expect.objectContaining({
              configuration: {
                ...PLANNING_REVIEW_DRAFT.components[0].configuration,
                additionalInstructions: "",
              },
            }),
          ],
        }),
      ),
    );
  });

  it("keeps edits after an agent save fails and retries before saving Planning", async () => {
    const saveAgent = vi.fn().mockRejectedValueOnce(new Error("Network error")).mockResolvedValue(undefined);
    const { onSave } = renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { agent: { onSave: saveAgent } });
    fireEvent.change(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" }), { target: { value: "7" } });
    fireEvent.change(screen.getByRole("textbox", { name: "Additional instructions" }), {
      target: { value: "Include rollout risks." },
    });
    fireEvent.click(screen.getByTestId("planning-settings-save"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Network error");
    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" })).toHaveValue(7);
    expect(screen.getByRole("textbox", { name: "Additional instructions" })).toHaveValue("Include rollout risks.");
    fireEvent.click(screen.getByTestId("planning-settings-save"));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(saveAgent).toHaveBeenCalledTimes(2);
  });

  it.each([true, false])(
    "blocks incomplete configuration and offers retry after loading fails: %s",
    async (loading) => {
      const onRetry = vi.fn();
      const { onSave } = renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, {
        agent: { draft: undefined, isLoading: loading },
        onRetry,
      });
      expect(screen.getByTestId("planning-settings-save")).toBeDisabled();
      if (loading) {
        expect(screen.getByRole("status")).toBeInTheDocument();
      } else {
        expect(screen.getByRole("alert")).toBeInTheDocument();
        await userEvent.click(screen.getByRole("button", { name: "Retry" }));
        expect(onRetry).toHaveBeenCalledTimes(1);
      }
      expect(onSave).not.toHaveBeenCalled();
    },
  );

  it("opens an old Agent link in Configuration", () => {
    vi.mocked(useExperimentalFeature).mockReturnValue({
      has: () => true,
      enabledExperimentalFeatures: [],
      isLoading: false,
      organizationReady: true,
    });
    renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { agent: true, initialPath: "/?planningSettings=agent" });
    expect(screen.getByTestId("planning-settings-tab-general")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" })).toBeInTheDocument();
  });

  it("shows Configuration and Runs tabs by default", () => {
    renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { agent: true });

    expect(screen.getByTestId("planning-settings-tab-general")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-settings-tab-agent")).not.toBeInTheDocument();
    expect(screen.getByTestId("planning-settings-tab-runs")).toBeInTheDocument();
  });
});
