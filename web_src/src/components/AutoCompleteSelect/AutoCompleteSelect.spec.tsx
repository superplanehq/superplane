import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { AutoCompleteSelect } from "./AutoCompleteSelect";

describe("AutoCompleteSelect", () => {
  it("uses the same trigger chrome as shadcn Select", () => {
    render(
      <AutoCompleteSelect
        options={[{ value: "KAN", label: "setntry-intake-test-project (KAN)" }]}
        value="KAN"
        onChange={vi.fn()}
      />,
    );

    const trigger = screen.getByText("setntry-intake-test-project (KAN)").closest("div[data-size='default']");
    expect(trigger?.className).toContain("h-8");
    expect(trigger?.className).toContain("bg-white");
    expect(trigger?.className).toContain("border-gray-300");
    expect(trigger?.className).toContain("rounded-md");
    expect(trigger?.className).toContain("shadow-xs");
    expect(trigger?.className).toContain("focus-within:border-gray-500");
    expect(trigger?.className).toContain("transition-[color,box-shadow]");
    expect(trigger?.className).toContain("dark:bg-gray-800");
    expect(trigger?.className).toContain("dark:border-gray-600/70");
    expect(trigger?.className).toContain("dark:text-gray-100");
    expect(trigger?.className).not.toContain("text-gray-800");
    expect(trigger?.className).not.toContain("bg-background");
    expect(trigger?.className).not.toContain("h-9");
  });
});
