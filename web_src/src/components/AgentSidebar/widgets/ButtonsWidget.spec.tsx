import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ButtonsWidget } from "./ButtonsWidget";

describe("ButtonsWidget", () => {
  it("uses semantic surface classes for the widget card and header", () => {
    render(<ButtonsWidget prompt="What do you want to build?" items={["A scheduled health check"]} />);

    const card = screen.getByText("What do you want to build?").closest(".my-4");
    expect(card?.className).toContain("bg-surface-raised");
    expect(card?.className).toContain("border-edge-default");

    const header = screen.getByText("What do you want to build?").parentElement;
    expect(header?.className).toContain("bg-surface-subtle");
  });
});
