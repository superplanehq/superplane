import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { AddIntakePicker } from "./AddIntakePicker";
import { ADD_INTAKE_COPY, ADD_INTAKE_TEMPLATES } from "./lineIntakeModel";

function renderPicker(
  onSelect = vi.fn(),
  extras: { takenSourceIds?: string[]; templates?: typeof ADD_INTAKE_TEMPLATES } = {},
) {
  render(
    <AddIntakePicker
      open
      onClose={vi.fn()}
      onSelect={onSelect}
      takenSourceIds={extras.takenSourceIds}
      templates={extras.templates}
    />,
  );
  return { onSelect };
}

describe("AddIntakePicker", () => {
  it("offers GitHub, Jira, Sentry, Productive.io, and coming-soon DataDog and Notion sources", () => {
    renderPicker();

    const picker = screen.getByTestId("add-intake-picker");
    expect(within(picker).getByRole("heading", { name: ADD_INTAKE_COPY.pickerTitle })).toBeInTheDocument();
    expect(within(picker).getByText(ADD_INTAKE_COPY.pickerDescription)).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-search")).toBeInTheDocument();
    expect(within(picker).getAllByTestId(/^add-intake-template-/)).toHaveLength(ADD_INTAKE_TEMPLATES.length);
    expect(within(picker).getByTestId("add-intake-template-github-issues")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-jira-issues")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-sentry-exceptions")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-productive-tasks")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-datadog")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(within(picker).getByTestId("add-intake-template-notion")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(within(picker).queryByTestId("add-intake-template-pagerduty-incidents")).not.toBeInTheDocument();
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

    await user.click(screen.getByTestId("add-intake-template-datadog"));

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

  it("restricts the list to the supplied templates", () => {
    const restricted = ADD_INTAKE_TEMPLATES.filter((template) =>
      ["github-issues", "sentry-exceptions"].includes(template.id),
    );
    renderPicker(vi.fn(), { templates: restricted });

    const picker = screen.getByTestId("add-intake-picker");
    expect(within(picker).getAllByTestId(/^add-intake-template-/)).toHaveLength(2);
    expect(within(picker).getByTestId("add-intake-template-github-issues")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-sentry-exceptions")).toBeInTheDocument();
    expect(within(picker).queryByTestId("add-intake-template-jira-issues")).not.toBeInTheDocument();
  });
});
