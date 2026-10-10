import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { statusCheckRows } from "./checksPRFeedbackSetup";
import { StatusCheckPicker } from "./StatusCheckPicker";

describe("statusCheckRows", () => {
  it("keeps catalog order and appends selected-only names", () => {
    expect(statusCheckRows([{ name: "lint" }, { name: "e2e" }, { name: "unit" }], ["unit", "custom"])).toEqual([
      { value: "lint", label: "lint" },
      { value: "e2e", label: "e2e" },
      { value: "unit", label: "unit" },
      { value: "custom", label: "custom" },
    ]);
  });

  it("uses the resource id as the value with the name as its label", () => {
    expect(statusCheckRows([{ id: "build-a", name: "Build A" }], [])).toEqual([{ value: "build-a", label: "Build A" }]);
  });

  it("keeps separate choices when two keys share one name", () => {
    expect(
      statusCheckRows(
        [
          { id: "e2e-1", name: "E2E" },
          { id: "e2e-2", name: "E2E" },
        ],
        [],
      ),
    ).toEqual([
      { value: "e2e-1", label: "E2E" },
      { value: "e2e-2", label: "E2E" },
    ]);
  });

  it("keeps saved selections visible when absent from the catalog", () => {
    expect(statusCheckRows([{ id: "build-a", name: "Build A" }], ["build-a", "missing-key"])).toEqual([
      { value: "build-a", label: "Build A" },
      { value: "missing-key", label: "missing-key" },
    ]);
  });
});

describe("StatusCheckPicker", () => {
  it("shows the partial loading state when selected names already exist", () => {
    render(<StatusCheckPicker names={["lint", "e2e"]} catalog={[]} loading onToggle={vi.fn()} />);

    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent("Loading other status checks");
  });

  it("shows the full loading state when no checks are selected yet", () => {
    render(<StatusCheckPicker names={[]} catalog={[]} loading onToggle={vi.fn()} />);

    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent("Loading status checks");
    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent(
      "Reading recent pull requests and the required status checks for this repository",
    );
  });

  it("toggles a catalog row", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<StatusCheckPicker names={["lint"]} catalog={[{ name: "lint" }, { name: "e2e" }]} onToggle={onToggle} />);

    await user.click(screen.getByTestId("pr-feedback-check-option-e2e"));
    expect(onToggle).toHaveBeenCalledWith("e2e");
  });

  it("keeps selected names visible while the catalog is still loading", () => {
    const onToggle = vi.fn();
    render(<StatusCheckPicker names={["lint"]} catalog={[]} loading onToggle={onToggle} />);

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toBeInTheDocument();
  });

  it("shows the key beside the label when they differ", () => {
    render(<StatusCheckPicker names={[]} catalog={[{ id: "build-a", name: "Build A" }]} onToggle={vi.fn()} />);

    expect(screen.getByTestId("pr-feedback-check-option-build-a")).toHaveTextContent("Build A (build-a)");
  });

  it("keeps separate choices when two keys share one name", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(
      <StatusCheckPicker
        names={[]}
        catalog={[
          { id: "e2e-1", name: "E2E" },
          { id: "e2e-2", name: "E2E" },
        ]}
        onToggle={onToggle}
      />,
    );

    expect(screen.getByTestId("pr-feedback-check-option-e2e-1")).toBeInTheDocument();
    expect(screen.getByTestId("pr-feedback-check-option-e2e-2")).toBeInTheDocument();
    await user.click(screen.getByTestId("pr-feedback-check-option-e2e-2"));
    expect(onToggle).toHaveBeenCalledWith("e2e-2");
  });
});
