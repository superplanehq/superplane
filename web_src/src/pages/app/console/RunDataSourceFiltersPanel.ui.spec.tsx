import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { RunDataSourceFiltersPanel } from "./RunDataSourceFiltersPanel";

const callbacks = {
  onStatusesChange: vi.fn(),
  onTriggersChange: vi.fn(),
};

describe("RunDataSourceFiltersPanel", () => {
  it("opens when filters become active after mounting", () => {
    const { rerender } = render(<RunDataSourceFiltersPanel statuses={undefined} triggers={undefined} {...callbacks} />);

    expect(screen.queryByTestId("run-datasource-filters-content")).not.toBeInTheDocument();

    rerender(<RunDataSourceFiltersPanel statuses={["passed"]} triggers={undefined} {...callbacks} />);

    expect(screen.getByTestId("run-datasource-filters-content")).toBeInTheDocument();
  });

  it("reopens a collapsed panel when its active filter configuration changes", () => {
    const { rerender } = render(
      <RunDataSourceFiltersPanel statuses={["passed"]} triggers={undefined} {...callbacks} />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Toggle run filters" }));
    expect(screen.queryByTestId("run-datasource-filters-content")).not.toBeInTheDocument();

    rerender(<RunDataSourceFiltersPanel statuses={["failed"]} triggers={undefined} {...callbacks} />);

    expect(screen.getByTestId("run-datasource-filters-content")).toBeInTheDocument();
  });
});
