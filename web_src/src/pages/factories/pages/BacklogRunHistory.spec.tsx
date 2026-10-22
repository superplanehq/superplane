import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation, useNavigate } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import {
  DEPENDABOT_INTAKE,
  DEPENDABOT_INTAKE_CANVAS,
  GITHUB_INTAKE,
  GITHUB_INTAKE_CANVAS,
  JIRA_INTAKE,
  SENTRY_INTAKE,
  SENTRY_INTAKE_CANVAS,
  renderHost,
  resetHostMocks,
  useCanvas,
} from "./IntakeSettingsHost.spec.fixtures";
import { PlanningSettingsPopup } from "./PlanningSettingsPopup";
import { DEFAULT_PLANNING_SETTINGS } from "./planningSettingsModel";

let historyEnabled = true;
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (feature: string) => historyEnabled && feature === "factory_run_history",
    isLoading: false,
    organizationReady: true,
  }),
}));
vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({ data: [], isLoading: false }),
}));
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: () => {} }));

describe("Backlog intake run history", () => {
  beforeEach(() => {
    resetHostMocks();
    historyEnabled = true;
  });

  it.each([
    [GITHUB_INTAKE, GITHUB_INTAKE_CANVAS],
    [SENTRY_INTAKE, SENTRY_INTAKE_CANVAS],
    [DEPENDABOT_INTAKE, DEPENDABOT_INTAKE_CANVAS],
    [JIRA_INTAKE, GITHUB_INTAKE_CANVAS],
  ] as const)("replaces the canvas with shared history for $0.source.id", async (intake, canvas) => {
    useCanvas.mockReturnValue({ data: canvas, isPending: false, isError: false, refetch: vi.fn() });
    const user = userEvent.setup();
    renderHost({ intake });

    expect(screen.queryByRole("tab", { name: "Automation" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-config-canvas")).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "General" }));
    expect(screen.getByTestId("intake-source-settings-save")).toBeInTheDocument();
  });

  it("keeps the legacy canvas when the flag is off", async () => {
    historyEnabled = false;
    const user = userEvent.setup();
    renderHost();
    await user.click(screen.getByRole("tab", { name: "Automation" }));
    expect(screen.getByTestId("intake-source-automation")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
  });

  it("shows the repository watched by Dependabot instead of the workspace default", () => {
    useCanvas.mockReturnValue({ data: DEPENDABOT_INTAKE_CANVAS, isPending: false, isError: false, refetch: vi.fn() });
    renderHost({ intake: DEPENDABOT_INTAKE, repository: "superplanehq/another-repository" });

    expect(screen.getByText("superplanehq/superplane")).toBeInTheDocument();
    expect(screen.queryByText("superplanehq/another-repository")).not.toBeInTheDocument();
  });

  it("shares Planning history and run selection through browser navigation", async () => {
    const user = userEvent.setup();
    renderPlanning();
    await user.click(screen.getByTestId("planning-settings-tab-runs"));
    expect(screen.getByTestId("location")).toHaveTextContent("planningSettings=runs");
    expect(screen.getByRole("table")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Open run run-intake-1" }));
    expect(screen.getByTestId("location")).toHaveTextContent("automationRun=run-intake-1");
    expect(screen.getByTestId("location")).toHaveTextContent("keep=1");
    await user.click(screen.getByTestId("browser-back"));
    expect(screen.getByRole("table")).toBeInTheDocument();
    await user.click(screen.getByTestId("browser-back"));
    expect(screen.getByTestId("planning-settings-save")).toBeInTheDocument();
    await user.click(screen.getByTestId("browser-forward"));
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-automation")).not.toBeInTheDocument();
  });

  it("opens old Planning automation links as history when enabled", () => {
    renderPlanning("/?planningSettings=automation");
    expect(screen.getByRole("tab", { name: "Runs" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-automation")).not.toBeInTheDocument();
  });

  it("keeps loading feedback while Planning's automation is being resolved", () => {
    renderPlanning("/?planningSettings=runs", { loading: true });
    expect(within(screen.getByRole("region", { name: "Run history" })).getByRole("status")).toHaveTextContent(
      "Loading run history",
    );
    expect(screen.queryByText("No runs yet")).not.toBeInTheDocument();
  });

  it("offers retry when Planning's automation could not be resolved", async () => {
    const retry = vi.fn();
    renderPlanning("/?planningSettings=runs", { error: true, onRetry: retry });
    expect(screen.getByRole("alert")).toHaveTextContent("Could not load run history");
    expect(screen.queryByText("No runs yet")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });
});

function BrowserControls() {
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <>
      <output data-testid="location">{location.search}</output>
      <button data-testid="browser-back" onClick={() => navigate(-1)}>
        Back
      </button>
      <button data-testid="browser-forward" onClick={() => navigate(1)}>
        Forward
      </button>
    </>
  );
}

function renderPlanning(
  path = "/?planning=1&keep=1",
  options?: { canvasId?: string; loading?: boolean; error?: boolean; onRetry?: () => void },
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[path]}>
        <BrowserControls />
        <PlanningSettingsPopup
          organizationId="org-1"
          factoryId="factory-1"
          canvasId={options ? options.canvasId : "planning-canvas"}
          automationLoading={options?.loading}
          automationError={options?.error}
          onRetryAutomation={options?.onRetry}
          settings={DEFAULT_PLANNING_SETTINGS}
          onSave={() => {}}
          onClose={() => {}}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}
