import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RunStatusBadge } from "./RunStatusBadge";

describe("RunStatusBadge", () => {
  it("renders the shared run status pill styling", () => {
    render(<RunStatusBadge status="running" />);

    const badge = screen.getByLabelText("Running");
    expect(badge).toHaveClass("rounded");
    expect(badge).toHaveClass("pl-1");
    expect(badge).toHaveClass("pr-1.5");
    expect(badge).toHaveClass("text-[12px]");
    expect(badge).toHaveClass("leading-4");
    expect(badge).not.toHaveClass("rounded-full");
    expect(screen.getByText("Running")).toBeInTheDocument();
  });
});
