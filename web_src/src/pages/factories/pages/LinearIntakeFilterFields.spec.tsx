import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "bun:test";

import { LinearIntakeFilterFields } from "./LinearIntakeFilterFields";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { linearFilterChoicesFromSettings } from "./linearIntakeScope";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

const resourceCalls = vi.hoisted(() => [] as Array<{ type: string; parameters?: Record<string, string> }>);

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: (
    _organizationId: string,
    _integrationId: string,
    resourceType: string,
    parameters?: Record<string, string>,
  ) => {
    resourceCalls.push({ type: resourceType, parameters });
    const resources: Record<string, Array<{ id: string; name: string }>> = {
      team: [
        { id: "team-1", name: "Platform (PLT)" },
        { id: "team-2", name: "Growth (GRO)" },
      ],
      label: [
        { id: "label-1", name: "bug" },
        { id: "label-2", name: "feature" },
      ],
      project: [
        { id: "project-1", name: "Checkout" },
        { id: "project-2", name: "Billing" },
      ],
    };
    return { data: resources[resourceType] ?? [], isLoading: false, isError: false, refetch: vi.fn() };
  },
}));

function Fields({
  sourceId = "linear-issues" as const,
  initial = {},
  onChange,
}: {
  sourceId?: "linear-issues" | "github-issues";
  initial?: Partial<IntakeSourceSettings>;
  onChange?: (settings: IntakeSourceSettings) => void;
}) {
  const [settings, setSettings] = useState<IntakeSourceSettings>({
    ...DEFAULT_GITHUB_INTAKE_SETTINGS,
    linearTeamId: "team-1",
    linearProjectIds: ["project-1"],
    linearLabels: [],
    ...initial,
  });
  const [filterChoices, setFilterChoices] = useState(() => linearFilterChoicesFromSettings(settings));
  onChange?.(settings);
  return (
    <LinearIntakeFilterFields
      sourceId={sourceId}
      settings={settings}
      onSettingsChange={setSettings}
      organizationId="org-1"
      integrationId="integration-1"
      filterChoices={filterChoices}
      onFilterChoicesChange={setFilterChoices}
    />
  );
}

describe("LinearIntakeFilterFields", () => {
  it("renders nothing for another source", () => {
    const { container } = render(<Fields sourceId="github-issues" />);
    expect(container).toBeEmptyDOMElement();
  });

  it("selects projects and labels loaded from Linear", async () => {
    const user = userEvent.setup();
    resourceCalls.length = 0;
    render(<Fields />);

    expect(screen.getByTestId("linear-team-team-1")).toHaveAttribute("aria-selected", "true");
    expect(resourceCalls).toContainEqual({ type: "project", parameters: { team: "team-1" } });
    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    expect(resourceCalls).toContainEqual({ type: "label", parameters: { team: "team-1" } });

    expect(screen.getByTestId("linear-project-project-1")).toHaveAttribute("aria-selected", "true");
    await user.click(screen.getByTestId("linear-project-project-2"));
    expect(screen.getByTestId("linear-project-project-2")).toHaveAttribute("aria-selected", "true");

    await user.click(screen.getByRole("checkbox", { name: "bug" }));
    expect(screen.getByRole("checkbox", { name: "bug" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "feature" })).not.toBeChecked();
    await user.click(screen.getByRole("checkbox", { name: "bug" }));
    expect(screen.getByRole("checkbox", { name: "bug" })).not.toBeChecked();
    await user.click(screen.getByRole("button", { name: LINEAR_INTAKE_SETUP_COPY.labelNew }));
    await user.type(screen.getByTestId("linear-intake-label-input"), "bug");
    await user.click(screen.getByRole("button", { name: LINEAR_INTAKE_SETUP_COPY.labelAdd }));
    expect(screen.getByText("bug")).toBeInTheDocument();
  });

  it("uses the issue settings label choice without widening the intake accidentally", async () => {
    const user = userEvent.setup();
    let latest: IntakeSourceSettings | undefined;
    render(<Fields initial={{ linearLabels: ["bug"] }} onChange={(next) => (latest = next)} />);

    expect(screen.getByRole("radio", { name: "Only issues with selected labels" })).toBeChecked();
    await user.click(screen.getByRole("radio", { name: "All issues" }));
    expect(latest?.linearLabels).toEqual([]);
    await user.click(screen.getByRole("radio", { name: "Only issues with selected labels" }));
    await user.click(screen.getByRole("checkbox", { name: "feature" }));
    expect(latest?.linearLabels).toEqual(["feature"]);
  });

  it("clears the projects and labels when the team changes", async () => {
    const user = userEvent.setup();
    let latest: IntakeSourceSettings | undefined;
    render(
      <Fields
        initial={{ linearProjectIds: ["project-1"], linearLabels: ["bug"] }}
        onChange={(next) => (latest = next)}
      />,
    );

    await user.click(screen.getByTestId("linear-team-team-2"));

    expect(latest).toMatchObject({ linearTeamId: "team-2", linearProjectIds: [], linearLabels: [] });
  });

  it("offers the no-project choice only while projects are selected", async () => {
    const user = userEvent.setup();
    let latest: IntakeSourceSettings | undefined;
    render(<Fields initial={{ linearProjectIds: [] }} onChange={(next) => (latest = next)} />);

    expect(screen.queryByTestId("linear-include-no-project")).not.toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "All projects" })).toBeChecked();
    await user.click(screen.getByRole("radio", { name: "Only issues in selected projects" }));
    await user.click(screen.getByTestId("linear-project-project-1"));
    const include = screen.getByRole("checkbox", { name: LINEAR_INTAKE_SETUP_COPY.includeNoProjectLabel });
    expect(include).toBeChecked();
    await user.click(include);

    expect(latest).toMatchObject({ linearProjectIds: ["project-1"], linearIncludeIssuesWithoutProject: false });
  });

  it("clears selected projects when all projects is chosen", async () => {
    const user = userEvent.setup();
    let latest: IntakeSourceSettings | undefined;
    render(<Fields initial={{ linearProjectIds: ["project-1"] }} onChange={(next) => (latest = next)} />);

    expect(screen.getByRole("radio", { name: "Only issues in selected projects" })).toBeChecked();
    await user.click(screen.getByRole("radio", { name: "All projects" }));

    expect(latest?.linearProjectIds).toEqual([]);
    expect(screen.queryByTestId("linear-project-picker")).not.toBeInTheDocument();
  });

  it("keeps the projects of an older intake that has no team", () => {
    resourceCalls.length = 0;
    render(<Fields initial={{ linearTeamId: "", linearProjectIds: ["project-1"] }} />);

    expect(screen.getByTestId("linear-project-project-1")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("linear-team-team-1")).toHaveAttribute("aria-selected", "false");
    expect(resourceCalls).toContainEqual({ type: "project", parameters: undefined });
  });

  it("hides the no-project choice while no team is selected", () => {
    render(<Fields initial={{ linearTeamId: "", linearProjectIds: ["project-1"] }} />);

    expect(screen.queryByTestId("linear-include-no-project")).not.toBeInTheDocument();
  });

  it("shows selected projects that Linear no longer returns so the user can clear them", async () => {
    const user = userEvent.setup();
    let latest: IntakeSourceSettings | undefined;
    render(
      <Fields initial={{ linearProjectIds: ["project-1", "project-gone"] }} onChange={(next) => (latest = next)} />,
    );

    const missing = screen.getByTestId("linear-missing-project-project-gone");
    expect(missing).toHaveAttribute("aria-selected", "true");
    expect(missing).toHaveTextContent(LINEAR_INTAKE_SETUP_COPY.projectMissingLabel);
    expect(screen.getByTestId("linear-project-project-1")).toHaveAttribute("aria-selected", "true");

    await user.click(missing);

    expect(latest).toMatchObject({ linearProjectIds: ["project-1"] });
    expect(screen.queryByTestId("linear-missing-project-project-gone")).not.toBeInTheDocument();
  });

  it("shows no projects or labels until a team is chosen", () => {
    render(<Fields initial={{ linearTeamId: "", linearProjectIds: [] }} />);

    expect(screen.getByTestId("linear-team-team-1")).toBeInTheDocument();
    expect(screen.queryByText(LINEAR_INTAKE_SETUP_COPY.projectsLabel)).not.toBeInTheDocument();
    expect(screen.queryByText(LINEAR_INTAKE_SETUP_COPY.labelsLabel)).not.toBeInTheDocument();
  });
});
