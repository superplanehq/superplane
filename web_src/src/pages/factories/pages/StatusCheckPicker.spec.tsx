import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { StatusCheckPicker, statusCheckRows } from "./StatusCheckPicker";

describe("statusCheckRows", () => {
  it("puts selected checks first and then the rest of the catalog", () => {
    expect(
      statusCheckRows(
        [
          { name: "lint", required: true },
          { name: "e2e", required: false },
          { name: "unit", required: false },
        ],
        ["e2e"],
      ),
    ).toEqual([
      { name: "e2e", required: false },
      { name: "lint", required: true },
      { name: "unit", required: false },
    ]);
  });
});

describe("StatusCheckPicker", () => {
  it("keeps selected checks visible while the catalog loads", () => {
    render(
      <StatusCheckPicker names={["lint", "e2e"]} catalog={[]} loading onToggle={vi.fn()} />,
    );

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-option-e2e")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent("Loading other status checks");
    expect(screen.queryByText("Reading recent pull requests")).not.toBeInTheDocument();
  });

  it("shows the full loading state when no checks are selected yet", () => {
    render(<StatusCheckPicker names={[]} catalog={[]} loading onToggle={vi.fn()} />);

    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent("Loading status checks");
    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent(
      "Reading recent pull requests and the required status checks for this repository",
    );
    expect(screen.queryByTestId("pr-feedback-check-names-list")).not.toBeInTheDocument();
  });

  it("keeps the catalog list while a refetch is in progress", () => {
    render(
      <StatusCheckPicker
        names={["lint"]}
        catalog={[
          { name: "lint", required: true },
          { name: "e2e", required: false },
        ]}
        loading
        onToggle={vi.fn()}
      />,
    );

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("pr-feedback-check-option-e2e")).toHaveAttribute("aria-selected", "false");
    expect(screen.queryByTestId("pr-feedback-check-names-loading")).not.toBeInTheDocument();
  });

  it("lets the user deselect a check that is already configured", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<StatusCheckPicker names={["lint"]} catalog={[]} loading onToggle={onToggle} />);

    await user.click(screen.getByTestId("pr-feedback-check-option-lint"));
    expect(onToggle).toHaveBeenCalledWith("lint");
  });
});
