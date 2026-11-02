import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  deleteIntake,
  renderHost,
  resetHostMocks,
  SENTRY_INTAKE,
  SENTRY_INTAKE_CANVAS,
  updateIntake,
  useCanvas,
} from "./IntakeSettingsHost.spec.fixtures";
import { INTAKE_SETTINGS_COPY } from "./intakeSourceSettingsModel";
import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";

describe("IntakeSettingsHost Sentry settings", () => {
  beforeEach(() => {
    resetHostMocks();
  });

  it("opens Sentry project settings alongside shared run history", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE, initialTab: "general" });

    const dialog = screen.getByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Intake Sentry exceptions" })).toBeInTheDocument();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["General", "Runs"]);
    expect(within(dialog).getByTestId("sentry-project-picker")).toBeInTheDocument();
    expect(within(dialog).queryByRole("combobox", { name: /Default task owner/ })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("checkbox", { name: "A new issue is created" })).toBeChecked();
    expect(within(dialog).getByTestId("sentry-project-payments")).toHaveAttribute("aria-selected", "true");

    await user.click(within(dialog).getByRole("tab", { name: "Runs" }));

    expect(within(dialog).getByRole("button", { name: "Open run run-intake-1" })).toBeInTheDocument();
  });

  it("saves Sentry projects and rebinds the intake when projects change", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE });

    await user.click(screen.getByRole("checkbox", { name: "A new issue is created" }));
    await user.click(screen.getByTestId("sentry-project-growth"));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-sentry",
      settings: expect.objectContaining({
        sentryProjectIds: ["payments", "growth"],
        sentryNewIssues: false,
      }),
      integrationId: "sentry-1",
      resourceId: "payments,growth",
    });
  });

  it("saves a default task owner with the Sentry intake settings", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE, subscription: true });

    const form = within(screen.getByRole("dialog"));
    await user.click(form.getByRole("combobox", { name: /Default task owner/ }));
    await user.click(await screen.findByRole("option", { name: "Alice" }));
    await user.click(form.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenLastCalledWith({
      intakeId: "intake-sentry",
      settings: expect.objectContaining({
        defaultOwnerId: "alice",
        sentryProjectIds: ["payments"],
      }),
    });
  });

  it("keeps Sentry save disabled until a project is selected", () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    renderHost({
      intake: {
        ...SENTRY_INTAKE,
        resourceId: "",
        settings: { ...SENTRY_INTAKE.settings, sentryProjectIds: [] },
      },
    });

    expect(screen.getByTestId("intake-source-settings-save")).toBeDisabled();
    expect(screen.getByRole("alert")).toHaveTextContent(SENTRY_INTAKE_SETUP_COPY.projectsRequired);
  });

  it("shows a Sentry save error in the settings panel", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    updateIntake.mockRejectedValue(new Error("save failed"));
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE });

    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(await screen.findByRole("alert")).toHaveTextContent("save failed");
  });
});

describe("IntakeSettingsHost Sentry actions", () => {
  beforeEach(() => {
    resetHostMocks();
  });

  it("deletes a Sentry intake from settings", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE, onClose });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteDescription);
    await user.click(screen.getByRole("button", { name: INTAKE_SETTINGS_COPY.deleteConfirm }));

    expect(deleteIntake).toHaveBeenCalledWith("intake-sentry");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows a Sentry delete error in the confirmation dialog", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    deleteIntake.mockRejectedValue(new Error("delete failed"));
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE, onClose });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    await user.click(screen.getByRole("button", { name: INTAKE_SETTINGS_COPY.deleteConfirm }));

    expect(screen.getByTestId("intake-delete-error")).toHaveTextContent("delete failed");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps Sentry project settings when the intake has no canvas", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult({ data: undefined }));
    const user = userEvent.setup();
    renderHost({
      intake: { ...SENTRY_INTAKE, appId: "" },
    });

    const dialog = screen.getByTestId("intake-source-settings");
    const form = dialog;
    expect(within(form).getByRole("checkbox", { name: "A new issue is created" })).toBeChecked();
    expect(within(form).queryByTestId("merge-confidence-config-form-collapse")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Runs" })).toBeInTheDocument();

    await user.click(within(form).getByRole("checkbox", { name: "A new issue is created" }));
    await user.click(within(form).getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-sentry",
      settings: expect.objectContaining({
        sentryNewIssues: false,
        sentryProjectIds: ["payments"],
      }),
    });
  });

  it("keeps Sentry project settings when the canvas fails to load", async () => {
    const refetch = vi.fn().mockResolvedValue(undefined);
    useCanvas.mockReturnValue(sentryCanvasResult({ data: undefined, isError: true, refetch }));
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE });

    const dialog = screen.getByTestId("intake-source-settings");
    const form = dialog;
    expect(within(form).getByRole("checkbox", { name: "A new issue is created" })).toBeChecked();
    expect(within(dialog).getByRole("tab", { name: "Runs" })).toBeInTheDocument();

    await user.click(screen.getByTestId("intake-source-settings-save"));
    await waitFor(() => expect(refetch).toHaveBeenCalled());
  });
});

function sentryCanvasResult(
  overrides: {
    data?: typeof SENTRY_INTAKE_CANVAS | undefined;
    isError?: boolean;
    refetch?: ReturnType<typeof vi.fn>;
  } = {},
) {
  return {
    data: "data" in overrides ? overrides.data : SENTRY_INTAKE_CANVAS,
    isPending: false,
    isError: overrides.isError ?? false,
    refetch: overrides.refetch ?? vi.fn().mockResolvedValue(undefined),
  };
}
