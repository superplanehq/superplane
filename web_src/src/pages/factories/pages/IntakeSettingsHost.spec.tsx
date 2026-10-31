import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  connectedJiraIntake,
  GITHUB_INTAKE,
  permissions,
  DATADOG_INTAKE,
  DEPENDABOT_INTAKE,
  DEPENDABOT_INTAKE_CANVAS,
  deleteIntake,
  PRODUCTIVE_INTAKE,
  renderHost,
  resetHostMocks,
  updateIntake,
  useCanvas,
} from "./IntakeSettingsHost.spec.fixtures";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, INTAKE_SETTINGS_COPY, intakeSettingsToApi } from "./intakeSourceSettingsModel";
import "./intakeSettingsHostSentryCases";

describe("IntakeSettingsHost", () => {
  beforeEach(() => {
    resetHostMocks();
  });

  it("opens the settings of the intake it is given", () => {
    renderHost();

    const dialog = screen.getByTestId("intake-source-settings");
    expect(within(dialog).getByRole("heading", { name: "Create tasks from GitHub issues" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("intake-connection")).not.toBeInTheDocument();
    expect(within(dialog).queryByLabelText("Name")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("checkbox", { name: "A new issue is opened" })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("checkbox", { name: "Only authors with repository access" })).toBeChecked();
    expect(within(dialog).getByRole("checkbox", { name: "Include reopened issues" })).toBeChecked();
    expect(
      within(dialog).queryByRole("checkbox", { name: 'The "superplane" label is added to the issue' }),
    ).not.toBeInTheDocument();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["General", "Runs"]);
    expect(within(dialog).getByRole("tab", { name: "Runs" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("intake-source-settings-automatic-intake")).toBeChecked();
    expect(within(dialog).getByTestId("intake-source-settings-delete")).toHaveTextContent(INTAKE_SETTINGS_COPY.delete);
    expect(screen.queryByTestId("merge-confidence-config")).not.toBeInTheDocument();
  });

  it("shows saved settings without edit actions when the user cannot update the workspace", async () => {
    permissions.canUpdate = false;
    updateIntake.mockClear();
    const user = userEvent.setup();
    renderHost({
      intake: {
        ...GITHUB_INTAKE,
        settings: { ...GITHUB_INTAKE.settings, filterByLabel: true, labels: ["superplane"] },
      },
      repository: "operately/operately",
      vcsIntegrationId: "github-1",
    });

    expect(
      screen.getByText("You do not have permission to change intake settings. Ask an organization admin for access."),
    ).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Include reopened issues" })).toBeDisabled();
    expect(screen.getByTestId("intake-source-settings-automatic-intake")).toBeDisabled();
    const labelPicker = screen.getByRole("combobox", { name: "Search or add a label" });
    await user.click(labelPicker.parentElement!);
    expect(screen.queryByRole("option", { name: "bug" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove superplane" })).toBeDisabled();
    expect(screen.queryByTestId("intake-source-settings-save")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings-delete")).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    expect(screen.getByRole("tab", { name: "Runs" })).toHaveAttribute("data-state", "active");
    expect(updateIntake).not.toHaveBeenCalled();
  });

  it("explains a 404 if a save loses access while the settings are open", async () => {
    updateIntake.mockRejectedValueOnce(Object.assign(new Error("Not found"), { status: 404 }));
    renderHost();

    await userEvent.click(screen.getByTestId("intake-source-settings-save"));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This intake could not be found or you do not have access. Refresh the page or ask an organization admin.",
    );
    expect(screen.queryByText("Intake settings saved.")).not.toBeInTheDocument();
  });

  it("confirms a successful save after closing the settings", async () => {
    const onClose = vi.fn();
    renderHost({ onClose });

    await userEvent.click(screen.getByTestId("intake-source-settings-save"));

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(await screen.findByText("Intake settings saved.")).toBeInTheDocument();
  });

  it("pauses and resumes GitHub intake without saving or losing filter edits", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderHost({ onClose });
    await user.click(screen.getByRole("checkbox", { name: "Include reopened issues" }));
    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "bug" }));
    await user.click(screen.getByTestId("intake-source-settings-automatic-intake"));
    await waitFor(() => expect(updateIntake).toHaveBeenLastCalledWith({ intakeId: "intake-github", paused: true }));
    await waitFor(() => expect(screen.getByTestId("intake-source-settings-automatic-intake")).not.toBeChecked());
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Remove bug" })).toBeInTheDocument();
    await user.click(screen.getByTestId("intake-source-settings-automatic-intake"));
    await waitFor(() => expect(updateIntake).toHaveBeenLastCalledWith({ intakeId: "intake-github", paused: false }));
    await waitFor(() => expect(screen.getByTestId("intake-source-settings-automatic-intake")).toBeChecked());
    expect(onClose).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("intake-source-settings-save"));
    await waitFor(() =>
      expect(updateIntake).toHaveBeenLastCalledWith({
        intakeId: "intake-github",
        settings: expect.objectContaining({ labels: ["bug"], reopenedIssues: false }),
      }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("keeps a paused GitHub intake paused after a failed resume and supports retry", async () => {
    updateIntake.mockClear();
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderHost({ intake: { ...GITHUB_INTAKE, paused: true }, onClose });
    expect(screen.getByTestId("intake-source-settings-automatic-intake")).not.toBeChecked();
    updateIntake.mockRejectedValueOnce(new Error("Network error"));
    await user.click(screen.getByTestId("intake-source-settings-automatic-intake"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Network error");
    expect(screen.getByTestId("intake-source-settings-automatic-intake")).not.toBeChecked();
    expect(onClose).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("intake-source-settings-automatic-intake"));
    await waitFor(() => expect(screen.getByTestId("intake-source-settings-automatic-intake")).toBeChecked());
    expect(updateIntake).toHaveBeenCalledTimes(2);
    expect(updateIntake).toHaveBeenLastCalledWith({ intakeId: "intake-github", paused: false });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("shows empty history for an intake without an automation", async () => {
    useCanvas.mockReturnValue({ data: undefined, isPending: false });
    renderHost({ intake: { ...GITHUB_INTAKE, appId: "" } });
    await userEvent.click(screen.getByRole("tab", { name: "Runs" }));
    expect(screen.getByText("No runs yet")).toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
  });

  it("saves the settings without renaming the intake", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-github",
      settings: intakeSettingsToApi(DEFAULT_GITHUB_INTAKE_SETTINGS, "github-issues"),
    });
  });

  it("saves GitHub labels the user selected", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "superplane" }));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-github",
      settings: expect.objectContaining({
        labels: ["superplane"],
      }),
    });
  });

  it("saves GitHub labels while the automation canvas is still loading", async () => {
    useCanvas.mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    const user = userEvent.setup();
    renderHost();

    const save = screen.getByTestId("intake-source-settings-save");
    expect(save).toBeEnabled();

    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "superplane" }));
    await user.click(save);

    expect(updateIntake).toHaveBeenCalledWith(
      expect.objectContaining({
        settings: expect.objectContaining({ labels: ["superplane"] }),
      }),
    );
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
    expect(screen.queryByTestId("intake-source-settings-automatic-intake")).not.toBeInTheDocument();
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

  it("opens Dependabot filters alongside shared run history", async () => {
    const user = userEvent.setup();
    useCanvas.mockReturnValue({ data: DEPENDABOT_INTAKE_CANVAS, isPending: false, isError: false, refetch: vi.fn() });
    renderHost({ intake: DEPENDABOT_INTAKE, repository: "superplanehq/superplane" });
    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Intake Dependabot alerts" })).toBeInTheDocument();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["General", "Runs"]);
    const form = dialog;
    expect(within(form).getByText("Repository")).toBeInTheDocument();
    expect(within(form).getByText("superplanehq/superplane")).toBeInTheDocument();
    expect(within(form).getByText("Create a task for these severities:")).toBeInTheDocument();
    expect(within(form).getByRole("button", { name: "Default task owner" })).toHaveTextContent("No owner");
    expect(within(form).getByRole("checkbox", { name: "Critical" })).toBeChecked();
    expect(within(form).getByRole("checkbox", { name: "Low" })).toBeChecked();
    await user.click(within(dialog).getByRole("tab", { name: "Runs" }));

    expect(within(dialog).getByRole("button", { name: "Open run run-intake-1" })).toBeInTheDocument();
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

    const dialog = screen.getByTestId("intake-source-settings");
    const form = dialog;
    expect(within(form).getByRole("checkbox", { name: "Critical" })).toBeChecked();
    expect(within(form).queryByTestId("merge-confidence-config-form-collapse")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Runs" })).toBeInTheDocument();

    await user.click(within(form).getByRole("checkbox", { name: "Low" }));
    await user.click(within(form).getByTestId("intake-source-settings-save"));

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

    const dialog = screen.getByTestId("intake-source-settings");
    const form = dialog;
    expect(within(form).getByRole("checkbox", { name: "High" })).toBeChecked();
    expect(within(dialog).getByRole("tab", { name: "Runs" })).toBeInTheDocument();

    await user.click(within(form).getByTestId("intake-source-settings-save"));

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
    await user.click(screen.getByTestId("intake-source-settings-save"));

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

    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(await screen.findByRole("alert")).toHaveTextContent("save failed");
  });

  it("deletes a Dependabot intake from settings", async () => {
    useCanvas.mockReturnValue({
      data: DEPENDABOT_INTAKE_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({ intake: DEPENDABOT_INTAKE, onClose });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteDescription);
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

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    await user.click(screen.getByRole("button", { name: INTAKE_SETTINGS_COPY.deleteConfirm }));

    expect(screen.getByTestId("intake-delete-error")).toHaveTextContent("delete failed");
    expect(onClose).not.toHaveBeenCalled();
  });
});
