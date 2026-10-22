import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomationsCopy";
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

  it("opens Sentry intake in the split config panel", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE, initialTab: "general" });

    const dialog = screen.getByTestId("merge-confidence-config");
    expect(screen.queryByTestId("intake-source-settings")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Sentry exceptions" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("merge-confidence-config-body")).toHaveAttribute("data-split", "true");
    expect(within(dialog).getByTestId("merge-confidence-config-canvas")).toBeInTheDocument();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["Automation", "Runs"]);
    expect(within(dialog).queryByRole("tab", { name: "General" })).not.toBeInTheDocument();
    const form = within(dialog).getByTestId("merge-confidence-config-form");
    expect(within(form).getByTestId("sentry-project-picker")).toBeInTheDocument();
    expect(within(form).getByRole("button", { name: "Default task owner" })).toBeInTheDocument();
    expect(within(form).getByRole("checkbox", { name: "A new issue is created" })).toBeChecked();
    expect(within(form).getByTestId("sentry-project-payments")).toHaveAttribute("aria-selected", "true");

    await user.click(within(dialog).getByTestId("merge-confidence-config-tab-runs"));

    expect(within(dialog).getByTestId("factory-automation-runs-sidebar")).toBeInTheDocument();
  });

  it("saves Sentry projects and rebinds the intake when projects change", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE });

    await user.click(screen.getByRole("checkbox", { name: "A new issue is created" }));
    await user.click(screen.getByTestId("sentry-project-growth"));
    await user.click(screen.getByTestId("sentry-intake-config-save"));

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
    renderHost({ intake: SENTRY_INTAKE });

    const form = within(screen.getByTestId("merge-confidence-config-form"));
    await user.click(form.getByRole("button", { name: "Default task owner" }));
    await user.click(await screen.findByRole("option", { name: "Alice" }));
    await user.click(form.getByTestId("sentry-intake-config-save"));

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

    expect(screen.getByTestId("sentry-intake-config-save")).toBeDisabled();
    expect(screen.getByTestId("sentry-intake-config-save-error")).toHaveTextContent(
      SENTRY_INTAKE_SETUP_COPY.projectsRequired,
    );
  });

  it("shows a Sentry save error in the settings panel", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    updateIntake.mockRejectedValue(new Error("save failed"));
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE });

    await user.click(screen.getByTestId("sentry-intake-config-save"));

    expect(screen.getByTestId("sentry-intake-config-save-error")).toHaveTextContent("save failed");
  });
});

describe("IntakeSettingsHost Sentry actions", () => {
  beforeEach(() => {
    resetHostMocks();
  });

  it("deletes a Sentry intake from the actions menu", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult());
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({ intake: SENTRY_INTAKE, onClose });

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(screen.getByRole("menuitem", { name: INTAKE_SETTINGS_COPY.delete }));
    expect(screen.getByRole("alertdialog", { name: INTAKE_SETTINGS_COPY.deleteTitle })).toHaveTextContent(
      INTAKE_SETTINGS_COPY.deleteDescription,
    );
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

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(screen.getByRole("menuitem", { name: INTAKE_SETTINGS_COPY.delete }));
    await user.click(screen.getByRole("button", { name: INTAKE_SETTINGS_COPY.deleteConfirm }));

    expect(screen.getByTestId("merge-confidence-config-delete-error")).toHaveTextContent("delete failed");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps Sentry project settings when the intake has no canvas", async () => {
    useCanvas.mockReturnValue(sentryCanvasResult({ data: undefined }));
    const user = userEvent.setup();
    renderHost({
      intake: { ...SENTRY_INTAKE, appId: "" },
    });

    const dialog = screen.getByTestId("merge-confidence-config");
    const form = within(dialog).getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("checkbox", { name: "A new issue is created" })).toBeChecked();
    expect(within(form).queryByTestId("merge-confidence-config-form-collapse")).not.toBeInTheDocument();
    expect(within(dialog).getByText(COLUMN_AUTOMATIONS_COPY.viewEmpty)).toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();

    await user.click(within(form).getByRole("checkbox", { name: "A new issue is created" }));
    await user.click(within(form).getByTestId("sentry-intake-config-save"));

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

    const dialog = screen.getByTestId("merge-confidence-config");
    const form = within(dialog).getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("checkbox", { name: "A new issue is created" })).toBeChecked();
    expect(within(dialog).getByText(COLUMN_AUTOMATIONS_COPY.viewError)).toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Runs" })).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: COLUMN_AUTOMATIONS_COPY.viewRetry }));

    expect(refetch).toHaveBeenCalled();
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
