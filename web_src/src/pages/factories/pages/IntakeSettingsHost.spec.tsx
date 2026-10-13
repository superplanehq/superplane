import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as ComponentDataModule from "@/hooks/useComponentData";
import type * as FactoryIntakeDataModule from "@/hooks/useFactoryIntakeData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { unmockedSrc } from "@/test/unmockedModule";

import {
  connectedJiraIntake,
  DATADOG_INTAKE,
  DEPENDABOT_INTAKE,
  DEPENDABOT_INTAKE_CANVAS,
  PRODUCTIVE_INTAKE,
  renderHost,
  resetIntakeSettingsHostMocks,
  SENTRY_INTAKE,
} from "./IntakeSettingsHost.spec.fixtures";
import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomationsCopy";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, INTAKE_SETTINGS_COPY, intakeSettingsToApi } from "./intakeSourceSettingsModel";

const {
  useCanvas,
  useTriggers,
  useComponents,
  useAvailableIntegrations,
  useIntegrationResources,
  updateIntake,
  deleteIntake,
  useInfiniteCanvasRuns,
  useDescribeRun,
  useEventExecutions,
} = vi.hoisted(() => ({
  useCanvas: vi.fn(),
  useTriggers: vi.fn(),
  useComponents: vi.fn(),
  useAvailableIntegrations: vi.fn(),
  useIntegrationResources: vi.fn(),
  updateIntake: vi.fn(),
  deleteIntake: vi.fn(),
  useInfiniteCanvasRuns: vi.fn(),
  useDescribeRun: vi.fn(),
  useEventExecutions: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => {
  function MockMonacoEditor({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) {
    return <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />;
  }
  return { default: MockMonacoEditor, Editor: MockMonacoEditor };
});

vi.mock("@/hooks/useCanvasData", () => ({
  ...unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData"),
  useCanvas,
  useTriggers,
  useInfiniteCanvasRuns,
  useDescribeRun,
  useEventExecutions,
}));

vi.mock("@/hooks/useComponentData", () => ({
  ...unmockedSrc<typeof ComponentDataModule>("hooks/useComponentData"),
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  ...unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations"),
  useAvailableIntegrations,
  useIntegrationResources,
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  ...unmockedSrc<typeof FactoryIntakeDataModule>("hooks/useFactoryIntakeData"),
  useUpdateFactoryIntake: () => ({ mutateAsync: updateIntake, isPending: false, error: null }),
  useDeleteFactoryIntake: () => ({ mutateAsync: deleteIntake, isPending: false, error: null }),
}));

describe("IntakeSettingsHost", () => {
  beforeEach(() => {
    resetIntakeSettingsHostMocks({
      useCanvas,
      useTriggers,
      useComponents,
      useAvailableIntegrations,
      useIntegrationResources,
      useDescribeRun,
      useEventExecutions,
      useInfiniteCanvasRuns,
      updateIntake,
      deleteIntake,
    });
  });

  it("opens the settings of the intake it is given", () => {
    renderHost();

    const dialog = screen.getByTestId("intake-source-settings");
    expect(within(dialog).getByRole("heading", { name: "Intake GitHub issues" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("intake-connection")).not.toBeInTheDocument();
    expect(within(dialog).queryByLabelText("Name")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("checkbox", { name: "A new issue is opened" })).toBeChecked();
    expect(within(dialog).getByRole("checkbox", { name: "A closed issue is re-opened" })).toBeChecked();
    expect(
      within(dialog).getByRole("checkbox", { name: 'The "superplane" label is added to the issue' }),
    ).toBeChecked();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["General", "Automation"]);
    expect(within(dialog).queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("intake-source-settings-pause")).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("intake-source-settings-delete")).toHaveTextContent(INTAKE_SETTINGS_COPY.delete);
    expect(screen.queryByTestId("merge-confidence-config")).not.toBeInTheDocument();
  });

  it("shows the automation of the intake canvas from the Automation tab", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(useCanvas).toHaveBeenCalledWith("org-1", "app-github-issues-intake", { enabled: true });
    const automation = within(screen.getByTestId("intake-source-settings")).getByTestId("intake-source-automation");
    expect(within(automation).getByTestId("rf__node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByText("Matches filters?")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Automation menu" })).not.toBeInTheDocument();
    const edit = within(automation).getByRole("link", { name: "Edit automation" });
    expect(edit).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/automations/app-github-issues-intake?configure=1&agent=1&from=lines&lineId=line-plan",
    );
    expect(useInfiniteCanvasRuns).toHaveBeenCalledWith("app-github-issues-intake", {}, true);
    const sidebar = within(automation).getByTestId("factory-automation-runs-sidebar");
    expect(within(sidebar).getByText("On mention on Issue")).toBeInTheDocument();
    expect(within(sidebar).getByRole("link", { name: "On mention on Issue" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/automations/app-github-issues-intake?run=run-intake-1&from=lines&lineId=line-plan",
    );
  });

  it("loads the selected run on the automation canvas", async () => {
    const user = userEvent.setup();
    renderHost({ initialTab: "automation" });

    const automation = within(screen.getByTestId("intake-source-settings")).getByTestId("intake-source-automation");
    expect(automation).not.toHaveAttribute("data-selected-run-id");
    expect(automation.querySelector(".sp-canvas-live")).toBeNull();

    await user.click(within(automation).getByRole("link", { name: "On mention on Issue" }));

    expect(automation).toHaveAttribute("data-selected-run-id", "run-intake-1");
    expect(automation.querySelector(".sp-canvas-live")).not.toBeNull();
    expect(within(automation).getByTestId("rf__node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByTestId("rf__node-github-issues-filter")).toBeInTheDocument();
    expect(within(automation).getByTestId("rf__node-github-issues-create")).toBeInTheDocument();
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

  it("saves the settings without renaming the intake", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-github",
      settings: intakeSettingsToApi(DEFAULT_GITHUB_INTAKE_SETTINGS),
    });
  });

  it("sends the Datadog service with the settings when the service changes", async () => {
    const user = userEvent.setup();
    renderHost({
      intake: DATADOG_INTAKE,
    });

    expect(screen.getByTestId("datadog-service-checkout")).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByTestId("datadog-service-name")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("datadog-service-billing"));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-datadog",
      settings: expect.objectContaining({
        datadogTriggeredAlerts: true,
        datadogRetriggeredAlerts: false,
      }),
      integrationId: "datadog-1",
      resourceId: "billing",
    });
  });

  it("saves the Jira completion column from General settings", async () => {
    const user = userEvent.setup();
    renderHost({
      intake: connectedJiraIntake({
        appId: "app-jira-issues-intake",
        integrationId: "jira-1",
        resourceId: "ENG",
      }),
    });

    expect(screen.queryByTestId("intake-connection")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-pause")).not.toBeInTheDocument();
    expect(screen.getByTestId("jira-completion-column")).toBeInTheDocument();
    expect(screen.getByTestId("jira-move-on-complete")).toBeChecked();
    await user.click(screen.getByTestId("jira-completion-column-select"));
    await user.click(screen.getByRole("option", { name: "QA" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-jira",
      settings: expect.objectContaining({
        jiraMoveOnComplete: true,
        jiraCompletionColumn: "QA",
      }),
    });
  });

  it("deletes a GitHub intake from settings", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({ onClose });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("intake-delete-dialog")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteDescription);
    await user.click(screen.getByTestId("intake-delete-confirm"));
    expect(deleteIntake).toHaveBeenCalledWith("intake-github");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("deletes a Sentry intake from settings", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({
      intake: SENTRY_INTAKE,
      onClose,
    });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
    await user.click(screen.getByTestId("intake-delete-confirm"));
    expect(deleteIntake).toHaveBeenCalledWith("intake-sentry");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("deletes a Jira intake from settings", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({
      intake: connectedJiraIntake(),
      onClose,
    });

    expect(screen.queryByTestId("intake-connection")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("intake-delete-dialog")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteDescription);
    await user.click(screen.getByTestId("intake-delete-confirm"));
    expect(deleteIntake).toHaveBeenCalledWith("intake-jira");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("deletes a Productive.io intake from settings", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({
      intake: PRODUCTIVE_INTAKE,
      onClose,
    });

    expect(screen.queryByTestId("intake-connection")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("intake-delete-dialog")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteDescription);
    await user.click(screen.getByTestId("intake-delete-confirm"));
    expect(deleteIntake).toHaveBeenCalledWith("intake-productive");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("deletes a Datadog intake from settings", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({
      intake: DATADOG_INTAKE,
      onClose,
    });

    expect(screen.queryByTestId("intake-connection")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("intake-delete-dialog")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteDescription);
    await user.click(screen.getByTestId("intake-delete-confirm"));
    expect(deleteIntake).toHaveBeenCalledWith("intake-datadog");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows a delete error in the confirmation dialog when delete fails", async () => {
    deleteIntake.mockRejectedValue(new Error("delete failed"));
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({
      intake: SENTRY_INTAKE,
      onClose,
    });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    await user.click(screen.getByTestId("intake-delete-confirm"));

    const dialog = screen.getByTestId("intake-delete-dialog");
    expect(within(dialog).getByTestId("intake-delete-error")).toHaveTextContent("delete failed");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("opens Dependabot intake in the split config panel", async () => {
    useCanvas.mockReturnValue({
      data: DEPENDABOT_INTAKE_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    const user = userEvent.setup();
    renderHost({ intake: DEPENDABOT_INTAKE, repository: "superplanehq/superplane" });

    const dialog = screen.getByTestId("merge-confidence-config");
    expect(screen.queryByTestId("intake-source-settings")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Dependabot alerts" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("merge-confidence-config-body")).toHaveAttribute("data-split", "true");
    expect(within(dialog).getByTestId("merge-confidence-config-canvas")).toBeInTheDocument();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["Automation", "Runs"]);
    const form = within(dialog).getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("heading", { name: "On Dependabot Alert" })).toBeInTheDocument();
    expect(within(form).getByText("Repository")).toBeInTheDocument();
    expect(within(form).getByText("superplanehq/superplane")).toBeInTheDocument();
    expect(within(form).getByText("Create a task for these severities:")).toBeInTheDocument();
    expect(within(form).getByRole("checkbox", { name: "Critical" })).toBeChecked();
    expect(within(form).getByRole("checkbox", { name: "Low" })).toBeChecked();
    expect(within(dialog).queryByRole("tab", { name: "General" })).not.toBeInTheDocument();

    await user.click(within(dialog).getByTestId("merge-confidence-config-tab-runs"));

    expect(within(dialog).getByTestId("factory-automation-runs-sidebar")).toBeInTheDocument();
  });

  it("keeps Dependabot severity settings when the intake has no canvas", async () => {
    useCanvas.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    const user = userEvent.setup();
    renderHost({
      intake: { ...DEPENDABOT_INTAKE, appId: "" },
      repository: "superplanehq/superplane",
    });

    const dialog = screen.getByTestId("merge-confidence-config");
    const form = within(dialog).getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("heading", { name: "On Dependabot Alert" })).toBeInTheDocument();
    expect(within(form).getByRole("checkbox", { name: "Critical" })).toBeChecked();
    expect(within(form).queryByTestId("merge-confidence-config-form-collapse")).not.toBeInTheDocument();
    expect(within(dialog).getByText(COLUMN_AUTOMATIONS_COPY.viewEmpty)).toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();

    await user.click(within(form).getByRole("checkbox", { name: "Low" }));
    await user.click(within(form).getByTestId("dependabot-intake-config-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-dependabot",
      settings: expect.objectContaining({
        dependabotSeverities: ["critical", "high", "medium"],
      }),
    });
  });

  it("keeps Dependabot severity settings when the canvas fails to load", async () => {
    const refetch = vi.fn().mockResolvedValue(undefined);
    useCanvas.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
      refetch,
    });
    const user = userEvent.setup();
    renderHost({ intake: DEPENDABOT_INTAKE, repository: "superplanehq/superplane" });

    const dialog = screen.getByTestId("merge-confidence-config");
    const form = within(dialog).getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("checkbox", { name: "High" })).toBeChecked();
    expect(within(dialog).getByText(COLUMN_AUTOMATIONS_COPY.viewError)).toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Runs" })).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: COLUMN_AUTOMATIONS_COPY.viewRetry }));
    await user.click(within(form).getByTestId("dependabot-intake-config-save"));

    expect(refetch).toHaveBeenCalled();
    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-dependabot",
      settings: expect.objectContaining({
        dependabotSeverities: [],
      }),
    });
  });

  it("saves Dependabot severities through intake settings", async () => {
    useCanvas.mockReturnValue({
      data: DEPENDABOT_INTAKE_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    const user = userEvent.setup();
    const view = renderHost({ intake: DEPENDABOT_INTAKE, repository: "superplanehq/superplane" });

    await user.click(screen.getByRole("checkbox", { name: "Low" }));
    await user.click(screen.getByTestId("dependabot-intake-config-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-dependabot",
      settings: expect.objectContaining({
        dependabotSeverities: ["critical", "high", "medium"],
      }),
    });

    view.unmount();
    renderHost({
      intake: {
        ...DEPENDABOT_INTAKE,
        settings: {
          ...DEPENDABOT_INTAKE.settings,
          dependabotSeverities: ["critical", "high", "medium"],
        },
      },
      repository: "superplanehq/superplane",
    });

    expect(screen.getByRole("checkbox", { name: "Low" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Critical" })).toBeChecked();
  });

  it("shows a Dependabot save error in the settings panel", async () => {
    useCanvas.mockReturnValue({
      data: DEPENDABOT_INTAKE_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    updateIntake.mockRejectedValue(new Error("save failed"));
    const user = userEvent.setup();
    renderHost({ intake: DEPENDABOT_INTAKE });

    await user.click(screen.getByTestId("dependabot-intake-config-save"));

    expect(screen.getByTestId("dependabot-intake-config-save-error")).toHaveTextContent("save failed");
  });

  it("deletes a Dependabot intake from the actions menu", async () => {
    useCanvas.mockReturnValue({
      data: DEPENDABOT_INTAKE_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({ intake: DEPENDABOT_INTAKE, onClose });

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(screen.getByRole("menuitem", { name: INTAKE_SETTINGS_COPY.delete }));
    expect(screen.getByRole("alertdialog", { name: INTAKE_SETTINGS_COPY.deleteTitle })).toHaveTextContent(
      INTAKE_SETTINGS_COPY.deleteDescription,
    );
    await user.click(screen.getByRole("button", { name: INTAKE_SETTINGS_COPY.deleteConfirm }));

    expect(deleteIntake).toHaveBeenCalledWith("intake-dependabot");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows a Dependabot delete error in the confirmation dialog", async () => {
    useCanvas.mockReturnValue({
      data: DEPENDABOT_INTAKE_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    deleteIntake.mockRejectedValue(new Error("delete failed"));
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({ intake: DEPENDABOT_INTAKE, onClose });

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(screen.getByRole("menuitem", { name: INTAKE_SETTINGS_COPY.delete }));
    await user.click(screen.getByRole("button", { name: INTAKE_SETTINGS_COPY.deleteConfirm }));

    expect(screen.getByTestId("merge-confidence-config-delete-error")).toHaveTextContent("delete failed");
    expect(onClose).not.toHaveBeenCalled();
  });
});
