import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { seededIntakeCatalog } from "@/test/intakeCatalog";

import { AddIntakePicker } from "./AddIntakePicker";
import { ADD_INTAKE_COPY, addIntakeTemplatesFromCatalog, type AddIntakeTemplate } from "./lineIntakeModel";

const SEEDED_TEMPLATES = addIntakeTemplatesFromCatalog(seededIntakeCatalog());

function renderPicker(
  onSelect = vi.fn(),
  extras: { takenSourceIds?: string[]; templates?: AddIntakeTemplate[]; loading?: boolean } = {},
) {
  render(
    <AddIntakePicker
      open
      onClose={vi.fn()}
      onSelect={onSelect}
      takenSourceIds={extras.takenSourceIds}
      templates={extras.templates ?? SEEDED_TEMPLATES}
      loading={extras.loading}
    />,
  );
  return { onSelect };
}

describe("AddIntakePicker", () => {
  it("offers the catalog sources and marks the ones the company cannot use as coming soon", () => {
    renderPicker();

    const picker = screen.getByTestId("add-intake-picker");
    expect(within(picker).getByRole("heading", { name: ADD_INTAKE_COPY.pickerTitle })).toBeInTheDocument();
    expect(within(picker).getByText(ADD_INTAKE_COPY.pickerDescription)).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-github-issues")).toBeEnabled();
    expect(within(picker).getByTestId("add-intake-template-sentry-exceptions")).toBeEnabled();
    expect(within(picker).getByTestId("add-intake-template-dependabot-alerts")).toBeEnabled();
    for (const id of ["jira-issues", "productive-tasks", "datadog", "linear-issues", "notion"]) {
      expect(within(picker).getByTestId(`add-intake-template-${id}`)).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    }
    expect(within(picker).queryByTestId("add-intake-template-pagerduty-incidents")).not.toBeInTheDocument();
  });

  it("shows a Beta badge on a Beta source that the company can use", () => {
    renderPicker(vi.fn(), { templates: addIntakeTemplatesFromCatalog(seededIntakeCatalog(["datadog"])) });

    const datadog = screen.getByTestId("add-intake-template-datadog");
    expect(datadog).toBeEnabled();
    expect(within(datadog).getByTestId("add-intake-beta-badge")).toHaveTextContent(ADD_INTAKE_COPY.beta);
    expect(
      within(screen.getByTestId("add-intake-template-github-issues")).queryByTestId("add-intake-beta-badge"),
    ).toBeNull();
  });

  it("marks a configured source as already set up", () => {
    renderPicker(vi.fn(), { takenSourceIds: ["github-issues"] });

    const github = screen.getByTestId("add-intake-template-github-issues");
    expect(github).toBeDisabled();
    expect(github).toHaveTextContent(ADD_INTAKE_COPY.sourceTaken);
  });

  it("does not report a coming-soon source to the caller", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderPicker();

    await user.click(screen.getByTestId("add-intake-template-notion"));

    expect(onSelect).not.toHaveBeenCalled();
  });

  it("reports the chosen live source to the caller", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderPicker();

    await user.click(screen.getByTestId("add-intake-template-github-issues"));

    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "github-issues" }));
  });

  it("filters sources from the search field", async () => {
    const user = userEvent.setup();
    renderPicker();

    await user.type(screen.getByTestId("add-intake-search"), "jira");

    expect(screen.getByTestId("add-intake-template-jira-issues")).toBeInTheDocument();
    expect(screen.queryByTestId("add-intake-template-github-issues")).not.toBeInTheDocument();
  });

  it("tells the user that the sources load", () => {
    renderPicker(vi.fn(), { templates: [], loading: true });

    expect(screen.getByText(ADD_INTAKE_COPY.loading)).toBeInTheDocument();
    expect(screen.queryAllByTestId(/^add-intake-template-/)).toHaveLength(0);
  });
});
