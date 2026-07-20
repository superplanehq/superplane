import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ButtonsWidget } from "./ButtonsWidget";

describe("ButtonsWidget", () => {
  it("uses dark-mode classes for the widget card and header", () => {
    render(<ButtonsWidget prompt="What do you want to build?" items={["A scheduled health check"]} />);

    const card = screen.getByText("What do you want to build?").closest(".my-4");
    expect(card?.className).toContain("dark:bg-gray-800");
    expect(card?.className).toContain("dark:border-gray-700");

    const header = screen.getByText("What do you want to build?").parentElement;
    expect(header?.className).toContain("dark:bg-gray-900/60");
  });
});
