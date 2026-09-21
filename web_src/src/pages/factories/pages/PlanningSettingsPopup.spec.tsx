import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { PlanningSettingsPopup } from "./PlanningSettingsPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { DEFAULT_PLANNING_SETTINGS, type PlanningDraftSettings } from "./planningSettingsModel";

function renderPopup(
  onSave = vi.fn(),
  settings: PlanningDraftSettings = DEFAULT_PLANNING_SETTINGS,
  options: { agent?: boolean } = {},
) {
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
              agent={
                options.agent
                  ? {
                      draft: PLANNING_REVIEW_DRAFT,
                      organizationId: "org-1",
                      onSave: vi.fn(),
                      showVisualEvidenceSetting: false,
                    }
                  : undefined
              }
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onSave };
}

describe("PlanningSettingsPopup", () => {
  it("saves Planning and score toggles", async () => {
    const user = userEvent.setup();
    const { onSave } = renderPopup();

    expect(screen.getByTestId("planning-settings-health")).toHaveTextContent("Ready");
    await user.click(within(screen.getByTestId("planning-settings-clarity")).getByRole("switch"));
    await user.click(screen.getByTestId("planning-settings-save"));

    expect(onSave).toHaveBeenCalledWith({
      enabled: true,
      clarity: false,
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

  it("shows General, Agent, and Automation tabs when an agent exists", () => {
    renderPopup(vi.fn(), DEFAULT_PLANNING_SETTINGS, { agent: true });

    expect(screen.getByTestId("planning-settings-tab-general")).toBeInTheDocument();
    expect(screen.getByTestId("planning-settings-tab-agent")).toBeInTheDocument();
    expect(screen.getByTestId("planning-settings-tab-automation")).toBeInTheDocument();
  });
});
