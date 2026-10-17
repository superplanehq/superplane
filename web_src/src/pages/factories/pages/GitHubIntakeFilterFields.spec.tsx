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

    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Issue has one of these labels" })).toBeChecked();
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

    const options = within(screen.getByTestId("intake-label-options"));
    await user.click(options.getByRole("checkbox", { name: "docs" }));
    await user.click(options.getByRole("checkbox", { name: "superplane" }));
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

    await user.click(screen.getByRole("checkbox", { name: "Issue has one of these labels" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.filterByLabel).toBe(false);
    expect(saved?.labels).toEqual([]);
    expect(saved?.superplaneLabelAdded).toBe(false);
  });

  it("lists superplane first and leaves out labels that are already selected", async () => {
    const user = userEvent.setup();
    render(<FilterHarness initial={excludeSettings()} onSave={() => undefined} />);

    const options = within(screen.getByTestId("intake-label-options"));
    const list = () => options.getAllByRole("checkbox").map((checkbox) => checkbox.getAttribute("aria-label"));
    expect(list()).toEqual(["superplane", "docs"]);

    await user.click(options.getByRole("checkbox", { name: "docs" }));
    expect(list()).toEqual(["superplane"]);
    expect(screen.getByRole("button", { name: "Remove docs" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove docs" }));
    expect(list()).toEqual(["superplane", "docs"]);
  });

  it("narrows the list when the user searches", async () => {
    const user = userEvent.setup();
    render(<FilterHarness initial={excludeSettings()} onSave={() => undefined} />);

    await user.type(screen.getByRole("textbox", { name: "Search or add a label" }), "doc");

    const options = within(screen.getByTestId("intake-label-options"));
    expect(options.getByRole("checkbox", { name: "docs" })).toBeInTheDocument();
    expect(options.queryByRole("checkbox", { name: "superplane" })).not.toBeInTheDocument();
  });

  it("creates a label that does not exist in the repository", async () => {
    const user = userEvent.setup();
    render(<FilterHarness initial={excludeSettings()} onSave={() => undefined} />);

    await user.type(screen.getByRole("textbox", { name: "Search or add a label" }), "needs-triage");
    await user.click(screen.getByRole("button", { name: 'Add label "needs-triage"' }));

    expect(screen.getByRole("button", { name: "Remove needs-triage" })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Search or add a label" })).toHaveValue("");
  });

  it("replaces the search with a manual add when every label is selected", async () => {
    const user = userEvent.setup();
    render(<FilterHarness initial={excludeSettings()} onSave={() => undefined} />);

    const options = within(screen.getByTestId("intake-label-options"));
    await user.click(options.getByRole("checkbox", { name: "superplane" }));
    await user.click(options.getByRole("checkbox", { name: "docs" }));

    expect(screen.queryByRole("textbox", { name: "Search or add a label" })).not.toBeInTheDocument();
    expect(screen.getByText("All labels selected")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Add label" }));
    await user.type(screen.getByRole("textbox", { name: "Search or add a label" }), "needs-triage{Enter}");

    expect(screen.getByRole("button", { name: "Remove needs-triage" })).toBeInTheDocument();
    expect(screen.getByText("All labels selected")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Search or add a label" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove docs" }));
    expect(screen.getByRole("textbox", { name: "Search or add a label" })).toBeInTheDocument();
    expect(screen.queryByText("All labels selected")).not.toBeInTheDocument();
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

    await user.click(screen.getByRole("checkbox", { name: "A closed issue is re-opened" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(saved?.reopenedIssues).toBe(false);
    expect(saved?.labels).toEqual(["bug"]);
  });
});
