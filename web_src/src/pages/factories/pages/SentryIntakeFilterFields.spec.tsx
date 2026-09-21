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
      <button type="button" onClick={() => onSave?.(normalizeIntakeSourceSettings(settings))}>
        Save
      </button>
    </div>
  );
}

describe("SentryIntakeFilterFields", () => {
  it("hides Sentry event and level fields for a GitHub intake", () => {
    render(<FilterHarness sourceId="github-issues" initial={DEFAULT_GITHUB_INTAKE_SETTINGS} />);

    expect(screen.queryByRole("checkbox", { name: "An issue becomes unresolved" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "An issue is assigned" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Fatal" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Error" })).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "A new issue is opened" })).toBeInTheDocument();
  });

  it("shows Sentry event and level fields for a Sentry intake", () => {
    render(<FilterHarness sourceId="sentry-exceptions" initial={DEFAULT_SENTRY_INTAKE_SETTINGS} />);

    expect(screen.getByRole("group", { name: "Create task when:" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "A new issue is created" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "An issue becomes unresolved" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "An issue is assigned" })).not.toBeChecked();
    expect(screen.getByRole("group", { name: "Filters" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Fatal" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Error" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Warning" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Info" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Debug" })).not.toBeChecked();
    expect(screen.queryByRole("checkbox", { name: "A new issue is opened" })).not.toBeInTheDocument();
  });

  it("puts Sentry event and level filters on the save payload", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    render(<FilterHarness sourceId="sentry-exceptions" initial={DEFAULT_SENTRY_INTAKE_SETTINGS} onSave={onSave} />);

    await user.click(screen.getByRole("checkbox", { name: "An issue is assigned" }));
    await user.click(screen.getByRole("checkbox", { name: "Error" }));
    await user.click(screen.getByRole("checkbox", { name: "Fatal" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        sentryNewIssues: true,
        sentryRegressedIssues: true,
        sentryAssignedIssues: true,
        sentryLevels: ["fatal", "error"],
      }),
    );
    expect(intakeSettingsToApi(onSave.mock.calls[0][0])).toMatchObject({
      sentryNewIssues: true,
      sentryRegressedIssues: true,
      sentryAssignedIssues: true,
      sentryLevels: ["fatal", "error"],
    });
  });
});
