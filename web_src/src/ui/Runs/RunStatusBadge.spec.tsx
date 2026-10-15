import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { RunStatusBadge } from "./RunStatusBadge";

describe("RunStatusBadge", () => {
  it("shows the run status with an accessible label", () => {
    render(<RunStatusBadge status="running" />);

    const badge = screen.getByLabelText("Running");
    expect(badge).toHaveTextContent("Running");
  });
});
