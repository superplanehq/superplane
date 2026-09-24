import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "bun:test";

import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  DEFAULT_PRODUCTIVE_INTAKE_SETTINGS,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";
import { ProductiveIntakeFilterFields } from "./ProductiveIntakeFilterFields";

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
      <ProductiveIntakeFilterFields sourceId={sourceId} settings={settings} onSettingsChange={setSettings} />
      <button type="button" onClick={() => onSave?.(normalizeIntakeSourceSettings(settings))}>
        Save
      </button>
    </div>
  );
}

describe("ProductiveIntakeFilterFields", () => {
  it("hides the key-task filter for a GitHub intake", () => {
    render(<FilterHarness sourceId="github-issues" initial={DEFAULT_GITHUB_INTAKE_SETTINGS} />);

    expect(screen.queryByRole("checkbox", { name: "Ignore key tasks" })).not.toBeInTheDocument();
  });

  it("shows the key-task filter on for a Productive.io intake", () => {
    render(<FilterHarness sourceId="productive-tasks" initial={DEFAULT_PRODUCTIVE_INTAKE_SETTINGS} />);

    expect(screen.getByRole("group", { name: "Filters" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Ignore key tasks" })).toBeChecked();
    expect(screen.getByText("SuperPlane skips Productive.io key tasks (milestones).")).toBeInTheDocument();
  });

  it("puts excludeKeyTasks on the save payload", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    render(<FilterHarness sourceId="productive-tasks" initial={DEFAULT_PRODUCTIVE_INTAKE_SETTINGS} onSave={onSave} />);

    await user.click(screen.getByRole("checkbox", { name: "Ignore key tasks" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ excludeKeyTasks: false }));
    expect(intakeSettingsToApi(onSave.mock.calls[0][0])).toMatchObject({ excludeKeyTasks: false });
  });
});
