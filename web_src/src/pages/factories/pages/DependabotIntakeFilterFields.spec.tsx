import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "bun:test";

import { DependabotIntakeFilterFields } from "./DependabotIntakeFilterFields";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

function FilterHarness({
  sourceId,
  initial,
  onSave,
}: {
  sourceId: LineIntakeSourceId;
  initial: IntakeSourceSettings;
  onSave: (next: IntakeSourceSettings) => void;
}) {
  const [settings, setSettings] = useState(initial);

  return (
    <div>
      <DependabotIntakeFilterFields sourceId={sourceId} settings={settings} onSettingsChange={setSettings} />
      <button type="button" onClick={() => onSave(normalizeIntakeSourceSettings(settings, sourceId))}>
        Save
      </button>
    </div>
  );
}

describe("DependabotIntakeFilterFields", () => {
  it("hides severity choices for a GitHub issue intake", () => {
    render(
      <FilterHarness sourceId="github-issues" initial={DEFAULT_GITHUB_INTAKE_SETTINGS} onSave={() => undefined} />,
    );

    expect(screen.queryByTestId("dependabot-severity-options")).not.toBeInTheDocument();
  });

  it("stores a subset of severities and treats every severity as all", async () => {
    const user = userEvent.setup();
    let saved: IntakeSourceSettings | undefined;
    render(
      <FilterHarness
        sourceId="dependabot-alerts"
        initial={{ ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Dependabot alerts" }}
        onSave={(next) => {
          saved = next;
        }}
      />,
    );

    expect(screen.getByRole("checkbox", { name: "Critical" })).toBeChecked();
    expect(screen.queryByText(/Turn on Dependabot alerts/)).not.toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Low" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.dependabotSeverities).toEqual(["critical", "high", "medium"]);
    expect(intakeSettingsToApi(saved!).dependabotSeverities).toEqual(["critical", "high", "medium"]);
  });
});
