import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FactoryHomePage } from "./FactoryHomePage";

const openCreateWorkOrder = vi.fn();
const setupAutomation = vi.fn();
let missingIntegrations: Record<string, string | undefined>;
let installedAutomationIds: Set<string>;
let installedAutomationApps: Map<string, { id: string; name: string } | undefined>;
let canvasRuns: Array<{ id: string; state: string }>;

vi.mock("../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
    factory: { name: "Payments", onboarding: {} },
    factories: [],
    openCreateWorkOrder,
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({
    data: { metadata: { liveVersionId: "version-1", createdAt: "2026-08-21T11:50:00Z" } },
    isLoading: false,
  }),
  useDescribeCanvasVersion: () => ({
    data: {
      spec: {
        nodes: [
          {
            component: "schedule",
            configuration: { type: "minutes", minutesInterval: 10 },
            metadata: { nextTrigger: "2099-08-21T12:00:00Z" },
          },
        ],
      },
    },
    isLoading: false,
  }),
  useInfiniteCanvasRuns: () => ({ data: { pages: [{ runs: canvasRuns }] }, isLoading: false }),
}));

vi.mock("./onboarding/useIngestionSetup", () => ({
  useIngestionSetup: () => ({
    dialogs: null,
    installedAutomationApps,
    installedAutomationIds,
    integrationsLoading: false,
    isInstalling: false,
    missingIntegration: (automationId: string) => missingIntegrations[automationId],
    setupAutomation,
  }),
}));

function renderPage() {
  return render(
    <MemoryRouter>
      <FactoryHomePage />
    </MemoryRouter>,
  );
}

describe("FactoryHomePage", () => {
  beforeEach(() => {
    openCreateWorkOrder.mockReset();
    setupAutomation.mockReset().mockResolvedValue(true);
    missingIntegrations = {};
    installedAutomationIds = new Set();
    installedAutomationApps = new Map();
    canvasRuns = [];
  });

  it("offers a task input or optional auto-ingestion", () => {
    renderPage();

    expect(screen.getByRole("heading", { name: "Start a task" })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Task" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create task" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Auto Ingestion" })).toBeInTheDocument();
    expect(screen.getByText("GitHub issue ingestion")).toBeInTheDocument();
    expect(screen.getByText("Sentry issue ingestion")).toBeInTheDocument();
    expect(setupAutomation).not.toHaveBeenCalled();
  });

  it("keeps internal work-order wording off the page", () => {
    renderPage();

    expect(screen.getByTestId("factory-home").textContent).not.toMatch(/work[- ]order/i);
  });

  it("opens the task form with the typed title", async () => {
    renderPage();

    await userEvent.type(screen.getByRole("textbox", { name: "Task" }), "Fix the billing export");
    await userEvent.click(screen.getByRole("button", { name: "Create task" }));

    expect(openCreateWorkOrder).toHaveBeenCalledWith("Fix the billing export");
    expect(screen.getByRole("textbox", { name: "Task" })).toHaveValue("");
  });

  it("opens the task form from the empty input", async () => {
    renderPage();

    await userEvent.click(screen.getByRole("button", { name: "Create task" }));

    expect(openCreateWorkOrder).toHaveBeenCalledWith("");
  });

  it("installs GitHub issue ingestion only after the user selects it", async () => {
    renderPage();

    await userEvent.click(screen.getByRole("button", { name: "Set up GitHub ingestion" }));

    expect(setupAutomation).toHaveBeenCalledWith("issue-intake");
  });

  it("marks Sentry ingestion as coming soon and offers no setup", () => {
    renderPage();

    expect(screen.getByText("Coming soon")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Set up Sentry ingestion" })).not.toBeInTheDocument();
  });

  it("shows status instead of a setup button when ingestion is installed", () => {
    installedAutomationIds = new Set(["issue-intake"]);
    installedAutomationApps = new Map([["issue-intake", { id: "app-1", name: "Issue Ingestion" }]]);
    renderPage();

    expect(screen.queryByRole("button", { name: "Set up GitHub ingestion" })).not.toBeInTheDocument();
    expect(screen.getByText(/Next scan/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "View history" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/PAY/apps/app-1?from=overview",
    );
  });

  it("links the working status to the active run", () => {
    installedAutomationIds = new Set(["issue-intake"]);
    installedAutomationApps = new Map([["issue-intake", { id: "app-1", name: "Issue Ingestion" }]]);
    canvasRuns = [{ id: "run-9", state: "STATE_STARTED" }];
    renderPage();

    expect(screen.getByText("Working now")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "View run" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/PAY/apps/app-1?run=run-9&from=overview",
    );
  });
});
