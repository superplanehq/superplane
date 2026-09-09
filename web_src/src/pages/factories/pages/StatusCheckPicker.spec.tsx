import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { StatusCheckPicker, statusCheckRows } from "./StatusCheckPicker";

describe("statusCheckRows", () => {
  it("keeps catalog order and appends selected-only names", () => {
    expect(
      statusCheckRows(
        [
          { name: "lint" },
          { name: "e2e" },
          { name: "unit" },
        ],
        ["unit", "custom"],
      ),
    ).toEqual([{ name: "lint" }, { name: "e2e" }, { name: "unit" }, { name: "custom" }]);
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
    render(
      <StatusCheckPicker
        names={["lint"]}
        catalog={[
          { name: "lint" },
          { name: "e2e" },
        ]}
        onToggle={onToggle}
      />,
    );

    await user.click(screen.getByTestId("pr-feedback-check-option-e2e"));
    expect(onToggle).toHaveBeenCalledWith("e2e");
  });

  it("keeps selected names visible while the catalog is still loading", () => {
    const onToggle = vi.fn();
    render(<StatusCheckPicker names={["lint"]} catalog={[]} loading onToggle={onToggle} />);

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toBeInTheDocument();
  });
});
