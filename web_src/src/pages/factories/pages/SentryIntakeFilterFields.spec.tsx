import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "bun:test";

import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  DEFAULT_SENTRY_INTAKE_SETTINGS,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";
import { SentryIntakeFilterFields } from "./SentryIntakeFilterFields";

function FilterHarness({
  sourceId,
  initial,
  onSave,
}: {
  sourceId: LineIntakeSourceId;
  initial: IntakeSourceSettings;
  onSave?: (next: IntakeSourceSettings) => void;
}) {
  const [settings, setSettings] = useState(initial);

  return (
    <div>
      <GitHubIntakeFilterFields sourceId={sourceId} settings={settings} onSettingsChange={setSettings} />
      <SentryIntakeFilterFields sourceId={sourceId} settings={settings} onSettingsChange={setSettings} />
      <button type="button" onClick={() => onSave?.(normalizeIntakeSourceSettings(settings, sourceId))}>
        Save
      </button>
    </div>
  );
}

describe("SentryIntakeFilterFields", () => {
  it("hides Sentry event fields and level checkboxes for a GitHub intake", () => {
    render(<FilterHarness sourceId="github-issues" initial={DEFAULT_GITHUB_INTAKE_SETTINGS} />);

    expect(screen.queryByRole("checkbox", { name: "A new issue is created" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Fatal" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Error" })).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "A new issue is opened" })).toBeInTheDocument();
  });

  it("shows only the new-issue trigger for a Sentry intake", () => {
    render(<FilterHarness sourceId="sentry-exceptions" initial={DEFAULT_SENTRY_INTAKE_SETTINGS} />);

    expect(screen.getByRole("group", { name: "Create task when:" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "A new issue is created" })).toBeChecked();
    expect(screen.queryByText("An issue becomes unresolved")).not.toBeInTheDocument();
    expect(screen.queryByText("An issue is assigned")).not.toBeInTheDocument();
    expect(screen.queryByRole("group", { name: "Filters" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Fatal" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Error" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Warning" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Info" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Debug" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "A new issue is opened" })).not.toBeInTheDocument();
  });

  it("toggles the new-issue trigger when its label is clicked", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    render(<FilterHarness sourceId="sentry-exceptions" initial={DEFAULT_SENTRY_INTAKE_SETTINGS} onSave={onSave} />);

    await user.click(screen.getByText("A new issue is created"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ sentryNewIssues: false }));
    expect(intakeSettingsToApi(onSave.mock.calls[0][0])).toMatchObject({
      sentryNewIssues: false,
    });
  });

  it("turns off hidden triggers on save so they cannot keep creating tasks", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    const initial: IntakeSourceSettings = {
      ...DEFAULT_SENTRY_INTAKE_SETTINGS,
      sentryRegressedIssues: true,
      sentryAssignedIssues: true,
    };
    render(<FilterHarness sourceId="sentry-exceptions" initial={initial} onSave={onSave} />);

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        sentryNewIssues: true,
        sentryRegressedIssues: false,
        sentryAssignedIssues: false,
      }),
    );
    expect(intakeSettingsToApi(onSave.mock.calls[0][0])).toMatchObject({
      sentryNewIssues: true,
      sentryRegressedIssues: false,
      sentryAssignedIssues: false,
    });
  });

  it("preserves stored sentryLevels on save even though the UI no longer shows them", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    const initial: IntakeSourceSettings = {
      ...DEFAULT_SENTRY_INTAKE_SETTINGS,
      sentryLevels: ["fatal", "error"],
    };
    render(<FilterHarness sourceId="sentry-exceptions" initial={initial} onSave={onSave} />);

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        sentryLevels: ["fatal", "error"],
      }),
    );
    expect(intakeSettingsToApi(onSave.mock.calls[0][0])).toMatchObject({
      sentryLevels: ["fatal", "error"],
    });
  });
});
