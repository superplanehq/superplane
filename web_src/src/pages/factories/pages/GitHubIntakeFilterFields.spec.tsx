import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "bun:test";

import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import {
  intakeSettingsFromApi,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";

function FilterHarness({
  initial,
  onSave,
}: {
  initial: IntakeSourceSettings;
  onSave: (next: IntakeSourceSettings) => void;
}) {
  const [settings, setSettings] = useState(initial);

  return (
    <div>
      <GitHubIntakeFilterFields
        sourceId="github-issues"
        settings={settings}
        onSettingsChange={setSettings}
        labelOptions={["bug", "docs"]}
      />
      <button type="button" onClick={() => onSave(normalizeIntakeSourceSettings(settings, "github-issues"))}>
        Save
      </button>
    </div>
  );
}

function excludeSettings(): IntakeSourceSettings {
  return intakeSettingsFromApi("GitHub issues", {
    labels: ["bug"],
    labelFilterMode: "LABEL_FILTER_MODE_EXCLUDE",
  });
}

describe("GitHubIntakeFilterFields label filter mode", () => {
  it("shows a saved exclude filter as excluding its labels", () => {
    render(<FilterHarness initial={excludeSettings()} onSave={() => undefined} />);

    expect(screen.getByRole("radio", { name: /Skip issues with the selected labels/ })).toBeChecked();
    expect(screen.getByRole("radio", { name: /Create a task only for issues/ })).not.toBeChecked();
    expect(within(screen.getByTestId("intake-label-options")).getByRole("checkbox", { name: "bug" })).toBeChecked();
  });

  it("keeps exclude mode when the user edits labels", async () => {
    const user = userEvent.setup();
    let saved: IntakeSourceSettings | undefined;
    render(
      <FilterHarness
        initial={excludeSettings()}
        onSave={(next) => {
          saved = next;
        }}
      />,
    );

    await user.click(within(screen.getByTestId("intake-label-options")).getByRole("checkbox", { name: "docs" }));
    await user.click(screen.getByRole("checkbox", { name: "superplane" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.labelFilterMode).toBe("exclude");
    expect(saved?.labels).toEqual(["bug", "docs", "superplane"]);
    expect(saved?.superplaneLabelAdded).toBe(false);
    expect(intakeSettingsToApi(saved!)).toMatchObject({
      labels: ["bug", "docs", "superplane"],
      labelFilterMode: "LABEL_FILTER_MODE_EXCLUDE",
    });
  });

  it("keeps exclude mode when the user saves unrelated settings", async () => {
    const user = userEvent.setup();
    let saved: IntakeSourceSettings | undefined;
    render(
      <FilterHarness
        initial={excludeSettings()}
        onSave={(next) => {
          saved = next;
        }}
      />,
    );

    await user.click(screen.getByRole("checkbox", { name: "A closed issue is re-opened" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.reopenedIssues).toBe(false);
    expect(saved?.labels).toEqual(["bug"]);
    expect(intakeSettingsToApi(saved!).labelFilterMode).toBe("LABEL_FILTER_MODE_EXCLUDE");
  });

  it("changes to include mode only after an explicit choice", async () => {
    const user = userEvent.setup();
    let saved: IntakeSourceSettings | undefined;
    render(
      <FilterHarness
        initial={excludeSettings()}
        onSave={(next) => {
          saved = next;
        }}
      />,
    );

    await user.click(screen.getByRole("radio", { name: /Create a task only for issues/ }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.labelFilterMode).toBe("include");
    expect(saved?.labels).toEqual(["bug"]);
    expect(saved?.superplaneLabelAdded).toBe(true);
  });
});
