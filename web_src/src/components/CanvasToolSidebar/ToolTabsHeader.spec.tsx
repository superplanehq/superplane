import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ToolTabsHeader } from "./ToolTabsHeader";

describe("ToolTabsHeader", () => {
  it("selects the first available tab when the active tab is hidden", () => {
    render(
      <ToolTabsHeader
        tabs={[
          { value: "runs", label: "Runs" },
          { value: "versions", label: "Versions" },
        ]}
        activeTab="agent"
        onSelectTab={vi.fn()}
      />,
    );

    expect(screen.getByRole("tab", { name: "Runs" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "Versions" })).toHaveAttribute("aria-selected", "false");
  });
});
