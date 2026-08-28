import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as ComponentDataModule from "@/hooks/useComponentData";
import type * as FactoryIntakeDataModule from "@/hooks/useFactoryIntakeData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSettingsHost } from "./IntakeSettingsHost";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, type IntakeSettingsTab } from "./intakeSourceSettingsModel";
import { lineIntakeSourceById, type ConfiguredLineIntakeSource } from "./lineIntakeModel";

const { useCanvas, useTriggers, useComponents, useAvailableIntegrations, useFactoryIntakeRuns, updateIntake } =
  vi.hoisted(() => ({
    useCanvas: vi.fn(),
    useTriggers: vi.fn(),
    useComponents: vi.fn(),
    useAvailableIntegrations: vi.fn(),
    useFactoryIntakeRuns: vi.fn(),
    updateIntake: vi.fn(),
  }));

vi.mock("@/hooks/useCanvasData", async (importOriginal) => ({
  ...(await importOriginal<typeof CanvasDataModule>()),
  useCanvas,
  useTriggers,
}));

vi.mock("@/hooks/useComponentData", async (importOriginal) => ({
  ...(await importOriginal<typeof ComponentDataModule>()),
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", async (importOriginal) => ({
  ...(await importOriginal<typeof IntegrationsModule>()),
  useAvailableIntegrations,
}));

vi.mock("@/hooks/useFactoryIntakeData", async (importOriginal) => ({
  ...(await importOriginal<typeof FactoryIntakeDataModule>()),
  useFactoryIntakeRuns,
  useUpdateFactoryIntake: () => ({ mutateAsync: updateIntake, isPending: false, error: null }),
}));

const GITHUB_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-github",
  appId: "app-github-issues-intake",
  healthy: true,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS },
  source: lineIntakeSourceById("github-issues")!,
};

const GITHUB_INTAKE_CANVAS = {
  metadata: { id: "app-github-issues-intake", name: "GitHub issues" },
  spec: {
    nodes: [
      { id: "github-issues-trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" },
      {
        id: "github-issues-filter",
        name: "Matches filters?",
        type: "TYPE_ACTION",
        component: "if",
        configuration: { expression: "true" },
      },
      { id: "github-issues-create", name: "Create Work Order", type: "TYPE_ACTION", component: "createWorkOrder" },
    ],
    edges: [
      { channel: "default", sourceId: "github-issues-trigger", targetId: "github-issues-filter" },
      { channel: "true", sourceId: "github-issues-filter", targetId: "github-issues-create" },
    ],
  },
};

function renderHost(
  props: {
    intake?: ConfiguredLineIntakeSource;
    initialTab?: IntakeSettingsTab;
    onOpenRun?: (run: { id: string }) => void;
    onClose?: () => void;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSettingsHost
              organizationId="org-1"
              factoryId="factory-1"
              factoryKey="RF"
              lineId="line-plan"
              intake={props.intake ?? GITHUB_INTAKE}
              initialTab={props.initialTab}
              onOpenRun={props.onOpenRun ?? vi.fn()}
              onClose={props.onClose ?? vi.fn()}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function intakeRuns(runs: unknown[]) {
  useFactoryIntakeRuns.mockReturnValue({ data: runs, isLoading: false, isError: false, refetch: vi.fn() });
}

describe("IntakeSettingsHost", () => {
  beforeEach(() => {
    useCanvas.mockReturnValue({
      data: GITHUB_INTAKE_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    useTriggers.mockReturnValue({ data: [{ name: "github.onIssue", label: "On Issue" }], isLoading: false });
    useComponents.mockReturnValue({
      data: [
        { name: "if", label: "If" },
        { name: "createWorkOrder", label: "Create Work Order" },
      ],
      isLoading: false,
    });
    useAvailableIntegrations.mockReturnValue({ data: [], isLoading: false });
    intakeRuns([]);
    updateIntake.mockResolvedValue({ id: "intake-github" });
  });

  it("opens the settings of the intake it is given", () => {
    renderHost();

    const dialog = screen.getByTestId("intake-source-settings");
    expect(within(dialog).getByRole("heading", { name: "Intake GitHub issues" })).toBeInTheDocument();
    expect(within(dialog).getByLabelText("Name")).toHaveValue("GitHub issues");
    expect(within(dialog).getByRole("radio", { name: /Listen for new issues/ })).toBeChecked();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["General", "Runs", "Automation"]);
  });

  it("shows the automation of the intake canvas from the Automation tab", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(useCanvas).toHaveBeenCalledWith("org-1", "app-github-issues-intake", { enabled: true });
    const automation = within(screen.getByTestId("intake-source-settings")).getByTestId("intake-source-automation");
    expect(within(automation).getByTestId("rf__node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByText("Matches filters?")).toBeInTheDocument();
    expect(within(automation).getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1&from=lines&lineId=line-plan",
    );
  });

  it("reports an intake without an automation instead of drawing one", async () => {
    useCanvas.mockReturnValue({ data: undefined, isPending: false });
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    const automation = screen.getByTestId("intake-source-automation");
    expect(automation).toHaveTextContent("This intake has no automation yet.");
    expect(within(automation).queryByTestId("rf__node-github-issues-trigger")).not.toBeInTheDocument();
  });

  it("shows the placement and score the server reported for each run", async () => {
    intakeRuns([
      {
        id: "run-1",
        title: "Handle duplicate refunds on retry",
        confidencePct: 94,
        placement: "PLACEMENT_PROGRESSED",
        stage: "implement",
      },
    ]);
    const onOpenRun = vi.fn();
    const user = userEvent.setup();
    renderHost({ initialTab: "runs", onOpenRun });

    const settings = screen.getByTestId("intake-source-settings");
    const run = within(settings).getByTestId("intake-source-run-run-1");
    expect(run).toHaveTextContent("94%");
    expect(run).toHaveTextContent("Implement");

    await user.click(screen.getByRole("button", { name: "View run for Handle duplicate refunds on retry" }));

    expect(onOpenRun).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "run-1",
        appId: "app-github-issues-intake",
        runId: "run-1",
        title: "Handle duplicate refunds on retry",
      }),
    );
  });

  it("saves the name and filters through the intake API", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Acme issues");
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-github",
      name: "Acme issues",
      settings: {
        confidencePct: 65,
        labels: [],
        labelFilterMode: "LABEL_FILTER_MODE_INCLUDE",
        assignment: "ASSIGNMENT_ANY",
      },
    });
  });
});
