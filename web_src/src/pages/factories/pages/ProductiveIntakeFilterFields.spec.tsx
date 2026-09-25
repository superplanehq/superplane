import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  DEFAULT_PRODUCTIVE_INTAKE_SETTINGS,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";
import { ProductiveIntakeFilterFields } from "./ProductiveIntakeFilterFields";
import { PRODUCTIVE_INTAKE_SETTINGS_COPY } from "./productiveIntakeSettingsCopy";

const { useIntegrationResources } = vi.hoisted(() => ({
  useIntegrationResources: vi.fn(),
}));

vi.mock("@/hooks/useIntegrations", () => ({ useIntegrationResources }));

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
      <ProductiveIntakeFilterFields
        sourceId={sourceId}
        settings={settings}
        onSettingsChange={setSettings}
        organizationId="org-1"
        integrationId="productive-1"
        projectId="project-1"
      />
      <button type="button" onClick={() => onSave?.(normalizeIntakeSourceSettings(settings))}>
        Save
      </button>
    </div>
  );
}

describe("ProductiveIntakeFilterFields", () => {
  beforeEach(() => {
    useIntegrationResources.mockReset();
    useIntegrationResources.mockReturnValue({
      data: [
        { id: "list-backlog", name: "Backlog" },
        { id: "list-bugs", name: "Bugs" },
      ],
      isLoading: false,
      isError: false,
    });
  });

  it("hides the key-task filter for a GitHub intake", () => {
    render(<FilterHarness sourceId="github-issues" initial={DEFAULT_GITHUB_INTAKE_SETTINGS} />);

    expect(screen.queryByRole("checkbox", { name: "Ignore key tasks" })).not.toBeInTheDocument();
  });

  it("shows the key-task filter on for a Productive.io intake", () => {
    render(<FilterHarness sourceId="productive-tasks" initial={DEFAULT_PRODUCTIVE_INTAKE_SETTINGS} />);

    expect(screen.getByRole("group", { name: "Filters" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Ignore key tasks" })).toBeChecked();
    expect(screen.getByText("SuperPlane skips Productive key tasks (milestones).")).toBeInTheDocument();
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

  it("loads the project task lists and saves the selected ones", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    render(<FilterHarness sourceId="productive-tasks" initial={DEFAULT_PRODUCTIVE_INTAKE_SETTINGS} onSave={onSave} />);

    expect(screen.getByText(PRODUCTIVE_INTAKE_SETTINGS_COPY.filterByTaskListHelper)).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Bugs" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("checkbox", { name: PRODUCTIVE_INTAKE_SETTINGS_COPY.filterByTaskList }));

    expect(useIntegrationResources).toHaveBeenLastCalledWith(
      "org-1",
      "productive-1",
      "task_list",
      { project: "project-1" },
      { enabled: true },
    );
    expect(screen.getByText(PRODUCTIVE_INTAKE_SETTINGS_COPY.taskListsNoneSelected)).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Bugs" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ taskListIds: ["list-bugs"] }));
    expect(intakeSettingsToApi(onSave.mock.calls[0][0])).toMatchObject({ taskListIds: ["list-bugs"] });
  });

  it("clears the selected task lists when the task list filter is turned off", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();
    render(
      <FilterHarness
        sourceId="productive-tasks"
        initial={{ ...DEFAULT_PRODUCTIVE_INTAKE_SETTINGS, taskListIds: ["list-backlog"] }}
        onSave={onSave}
      />,
    );

    expect(screen.getByRole("checkbox", { name: "Backlog" })).toBeChecked();
    await user.click(screen.getByRole("checkbox", { name: PRODUCTIVE_INTAKE_SETTINGS_COPY.filterByTaskList }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ taskListIds: [] }));
  });

  it("keeps a selected task list that the project no longer returns", () => {
    render(
      <FilterHarness
        sourceId="productive-tasks"
        initial={{ ...DEFAULT_PRODUCTIVE_INTAKE_SETTINGS, taskListIds: ["list-archived"] }}
      />,
    );

    expect(screen.getByRole("checkbox", { name: "list-archived" })).toBeChecked();
  });
});
