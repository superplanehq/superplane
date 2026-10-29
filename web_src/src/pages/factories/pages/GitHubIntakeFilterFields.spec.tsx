import { render, screen } from "@testing-library/react";
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

describe("GitHubIntakeFilterFields", () => {
  it("shows a saved exclude filter as its labels and saves it as an include filter", async () => {
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

    expect(screen.getByRole("radio", { name: "Only issues with selected labels" })).toBeChecked();
    expect(screen.getByRole("button", { name: "Remove bug" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.labelFilterMode).toBe("include");
    expect(saved?.labels).toEqual(["bug"]);
    expect(saved?.superplaneLabelAdded).toBe(true);
    expect(intakeSettingsToApi(saved!)).toMatchObject({
      labels: ["bug"],
      labelFilterMode: "LABEL_FILTER_MODE_INCLUDE",
    });
  });

  it("keeps the selected labels when the user edits them", async () => {
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

    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "docs" }));
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "superplane" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.labels).toEqual(["bug", "docs", "superplane"]);
    expect(saved?.superplaneLabelAdded).toBe(true);
  });

  it("clears the labels and label-added listening when the user turns the label filter off", async () => {
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

    await user.click(screen.getByRole("radio", { name: "All issues" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.filterByLabel).toBe(false);
    expect(saved?.labels).toEqual([]);
    expect(saved?.superplaneLabelAdded).toBe(false);
  });

  it("lists superplane first and leaves out labels that are already selected", async () => {
    const user = userEvent.setup();
    render(<FilterHarness initial={excludeSettings()} onSave={() => undefined} />);

    const list = () => screen.getAllByRole("option").map((option) => option.textContent);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    expect(list()).toEqual(["superplane", "docs"]);

    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    await user.click(screen.getByRole("option", { name: "docs" }));
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    expect(list()).toEqual(["superplane"]);
    expect(screen.getByRole("button", { name: "Remove docs" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove docs" }));
    expect(list()).toEqual(["superplane", "docs"]);
  });

  it("narrows the list when the user searches", async () => {
    const user = userEvent.setup();
    render(<FilterHarness initial={excludeSettings()} onSave={() => undefined} />);

    await user.type(screen.getByRole("combobox", { name: "Search or add a label" }), "doc");

    expect(screen.getByRole("option", { name: "docs" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "superplane" })).not.toBeInTheDocument();
  });

  it("creates a label that does not exist in the repository", async () => {
    const user = userEvent.setup();
    render(<FilterHarness initial={excludeSettings()} onSave={() => undefined} />);

    await user.type(screen.getByRole("combobox", { name: "Search or add a label" }), "needs-triage");
    await user.click(screen.getByRole("option", { name: 'Add label "needs-triage"' }));

    expect(screen.getByRole("button", { name: "Remove needs-triage" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Search or add a label" })).toHaveValue("");
    await user.type(
      screen.getByRole("combobox", { name: "Search or add a label" }),
      " NEEDS-TRIAGE {ArrowDown}{Enter}{Escape}",
    );
    expect(screen.getAllByRole("button", { name: /Remove needs-triage/i })).toHaveLength(1);
  });

  it("accepts a custom label after all repository labels have been selected", async () => {
    const user = userEvent.setup();
    render(
      <FilterHarness
        initial={{ ...excludeSettings(), labels: ["superplane", "bug", "docs"] }}
        onSave={() => undefined}
      />,
    );

    await user.type(screen.getByRole("combobox", { name: "Search or add a label" }), "needs-triage{ArrowDown}{Enter}");
    expect(screen.getByRole("button", { name: "Remove needs-triage" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Remove docs" }));
    await user.click(screen.getByRole("combobox", { name: "Search or add a label" }));
    expect(screen.getByRole("option", { name: "docs" })).toBeInTheDocument();
  });

  it("keeps the re-opened toggle apart from the label filter", async () => {
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

    await user.click(screen.getByRole("checkbox", { name: "Include reopened issues" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.reopenedIssues).toBe(false);
    expect(saved?.labels).toEqual(["bug"]);
  });
});
