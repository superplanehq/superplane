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

import { LineIntakeDrawer } from "./LineIntakeDrawer";
import { lineIntakeSourceById, type ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { DEFAULT_GITHUB_INTAKE_SETTINGS } from "./intakeSourceSettingsModel";
import type { LineIntakeDrawerProps } from "./lineIntakeDrawerTypes";

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

function configuredIntake(overrides: Partial<ConfiguredLineIntakeSource> = {}): ConfiguredLineIntakeSource {
  const source = lineIntakeSourceById("github-issues")!;
  return {
    intakeId: "intake-github",
    appId: "app-github-issues-intake",
    healthy: true,
    settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS },
    source,
    ...overrides,
  };
}

const GITHUB_INTAKE = configuredIntake();

const SENTRY_INTAKE = configuredIntake({
  intakeId: "intake-sentry",
  appId: "app-sentry-intake",
  source: lineIntakeSourceById("sentry-exceptions")!,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Sentry exceptions" },
});

const PAGERDUTY_INTAKE = configuredIntake({
  intakeId: "intake-pagerduty",
  appId: "app-pagerduty-intake",
  source: lineIntakeSourceById("pagerduty-incidents")!,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "PagerDuty incidents" },
});

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

function drawerElement(props: Partial<LineIntakeDrawerProps>) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <LineIntakeDrawer
              onClose={props.onClose ?? vi.fn()}
              configuredSources={props.configuredSources ?? [GITHUB_INTAKE, SENTRY_INTAKE, PAGERDUTY_INTAKE]}
              {...props}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function renderDrawer(props: Partial<LineIntakeDrawerProps> = {}) {
  return render(drawerElement(props));
}

function intakeRuns(runs: unknown[]) {
  useFactoryIntakeRuns.mockReturnValue({ data: runs, isLoading: false, isError: false, refetch: vi.fn() });
}

describe("LineIntakeDrawer", () => {
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

  it("lists the intakes the workspace declared", () => {
    renderDrawer();

    const drawer = screen.getByTestId("line-intake-drawer");
    expect(drawer).toHaveAccessibleName("Intake");
    expect(screen.getByRole("heading", { name: "Intake" })).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-source-intake-github")).toHaveTextContent("GitHub issues");
    expect(screen.getByTestId("line-intake-source-intake-sentry")).toHaveTextContent("Sentry exceptions");
    expect(screen.getByTestId("line-intake-source-intake-pagerduty")).toHaveTextContent("PagerDuty incidents");
  });

  it("lists two intakes on the same source as separate rows", () => {
    renderDrawer({
      configuredSources: [
        GITHUB_INTAKE,
        configuredIntake({
          intakeId: "intake-github-triage",
          appId: "app-triage",
          source: { ...lineIntakeSourceById("github-issues")!, name: "Triage issues" },
        }),
      ],
    });

    expect(screen.getByTestId("line-intake-source-intake-github")).toHaveTextContent("GitHub issues");
    expect(screen.getByTestId("line-intake-source-intake-github-triage")).toHaveTextContent("Triage issues");
  });

  it("shows only the intakes it is given", () => {
    renderDrawer({ configuredSources: [GITHUB_INTAKE] });

    expect(screen.getByTestId("line-intake-source-intake-github")).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-source-intake-sentry")).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-source-intake-pagerduty")).not.toBeInTheDocument();
  });

  it("marks an intake whose automation can no longer create work orders", () => {
    renderDrawer({ configuredSources: [configuredIntake({ healthy: false })] });

    const intake = screen.getByTestId("line-intake-source-intake-github");
    expect(within(intake).getByTestId("line-intake-source-intake-github-needs-repair")).toHaveTextContent(
      "Needs repair",
    );
    expect(intake).toHaveTextContent("Open it to repair the steps.");
  });

  it("closes the drawer from the header control", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderDrawer({ onClose });

    await user.click(screen.getByTestId("line-intake-close"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("hides the Add intake control by default", () => {
    renderDrawer();

    expect(screen.queryByTestId("line-intake-add")).not.toBeInTheDocument();
  });

  it("opens a searchable picker with six intake templates", async () => {
    const user = userEvent.setup();
    renderDrawer({ showAddIntakeControl: true });

    await user.click(screen.getByTestId("line-intake-add"));

    const picker = screen.getByTestId("add-intake-picker");
    expect(within(picker).getByRole("heading", { name: "Add intake" })).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-search")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-improve-ci-runtime")).toHaveTextContent(
      "Improve CI runtime",
    );
    expect(within(picker).getAllByTestId(/^add-intake-template-/)).toHaveLength(6);
  });

  it("filters templates from the picker search", async () => {
    const user = userEvent.setup();
    renderDrawer({ showAddIntakeControl: true });

    await user.click(screen.getByTestId("line-intake-add"));
    await user.type(screen.getByTestId("add-intake-search"), "runtime");

    expect(screen.getByTestId("add-intake-template-improve-ci-runtime")).toBeInTheDocument();
    expect(screen.queryByTestId("add-intake-template-flaky-tests")).not.toBeInTheDocument();
  });

  it("reports the chosen template to the caller and closes the picker", async () => {
    const onSelectIntakeTemplate = vi.fn();
    const user = userEvent.setup();
    renderDrawer({ showAddIntakeControl: true, onSelectIntakeTemplate });

    await user.click(screen.getByTestId("line-intake-add"));
    await user.click(screen.getByTestId("add-intake-template-github-issues"));

    expect(onSelectIntakeTemplate).toHaveBeenCalledWith(expect.objectContaining({ id: "github-issues" }));
    expect(screen.queryByTestId("add-intake-picker")).not.toBeInTheDocument();
  });

  it("lists each intake without an expand control or empty run list", () => {
    renderDrawer({ configuredSources: [GITHUB_INTAKE] });

    const github = screen.getByTestId("line-intake-source-intake-github");
    expect(within(github).getByRole("heading", { name: "GitHub issues" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Expand GitHub issues" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Collapse GitHub issues" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-empty")).not.toBeInTheDocument();
    expect(screen.queryByText("No intake runs in progress.")).not.toBeInTheDocument();
  });

  it("opens intake settings from the gear", async () => {
    const user = userEvent.setup();
    renderDrawer({ configuredSources: [GITHUB_INTAKE], organizationId: "org-1", factoryId: "factory-1" });

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));

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

  it("opens the settings of the intake whose gear was used", async () => {
    const user = userEvent.setup();
    renderDrawer({
      configuredSources: [
        GITHUB_INTAKE,
        configuredIntake({
          intakeId: "intake-github-triage",
          appId: "app-triage",
          source: { ...lineIntakeSourceById("github-issues")!, name: "Triage issues" },
          settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Triage issues", confidencePct: 80 },
        }),
      ],
      organizationId: "org-1",
      factoryId: "factory-1",
    });

    await user.click(screen.getByRole("button", { name: "Open Triage issues settings" }));

    const dialog = screen.getByTestId("intake-source-settings");
    expect(within(dialog).getByLabelText("Name")).toHaveValue("Triage issues");
  });

  it("shows the automation of the intake canvas from the Automation tab", async () => {
    const user = userEvent.setup();
    renderDrawer({
      configuredSources: [GITHUB_INTAKE],
      organizationId: "org-1",
      factoryId: "factory-1",
      editAutomationHref: "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1&from=lines",
    });

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));
    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(useCanvas).toHaveBeenCalledWith("org-1", "app-github-issues-intake", { enabled: true });
    const automation = within(screen.getByTestId("intake-source-settings")).getByTestId("intake-source-automation");
    expect(within(automation).getByTestId("rf__node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByText("Matches filters?")).toBeInTheDocument();
    expect(within(automation).getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1&from=lines",
    );
  });

  it("reports an intake without an automation instead of drawing one", async () => {
    useCanvas.mockReturnValue({ data: undefined, isPending: false });
    const user = userEvent.setup();
    renderDrawer({ configuredSources: [GITHUB_INTAKE], organizationId: "org-1", factoryId: "factory-1" });

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));
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
    const onOpenTicket = vi.fn();
    const user = userEvent.setup();
    renderDrawer({
      initialIntakeId: "intake-github",
      initialSettingsOpen: true,
      initialSettingsTab: "runs",
      configuredSources: [GITHUB_INTAKE],
      organizationId: "org-1",
      factoryId: "factory-1",
      onOpenTicket,
    });

    const settings = screen.getByTestId("intake-source-settings");
    const run = within(settings).getByTestId("intake-source-run-run-1");
    expect(run).toHaveTextContent("94%");
    expect(run).toHaveTextContent("Implement");

    await user.click(screen.getByRole("button", { name: "View run for Handle duplicate refunds on retry" }));

    expect(onOpenTicket).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "run-1",
        appId: "app-github-issues-intake",
        runId: "run-1",
        title: "Handle duplicate refunds on retry",
      }),
    );
    expect(screen.getByTestId("intake-source-settings")).toBeInTheDocument();
  });

  it("saves the name and filters through the intake API", async () => {
    const user = userEvent.setup();
    renderDrawer({ configuredSources: [GITHUB_INTAKE], organizationId: "org-1", factoryId: "factory-1" });

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));
    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Acme issues");
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(screen.queryByTestId("intake-source-settings")).not.toBeInTheDocument();
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
