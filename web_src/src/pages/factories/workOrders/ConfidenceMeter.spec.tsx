import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { ConfidenceMeter } from "./ConfidenceMeter";

describe("ConfidenceMeter", () => {
  it("fills bars up to the score", () => {
    render(<ConfidenceMeter score={3} testId="confidence-meter" />);

    const meter = screen.getByTestId("confidence-meter");
    expect(meter).toHaveAttribute("role", "meter");
    expect(meter).toHaveAttribute("aria-valuenow", "3");
    expect(meter).toHaveAttribute("aria-valuemax", "5");
    expect(meter.querySelectorAll("[data-filled='true']")).toHaveLength(3);
    expect(meter.querySelectorAll("[data-filled='false']")).toHaveLength(2);
  });

  it("shows the check name and score on hover", async () => {
    const user = userEvent.setup();
    render(<ConfidenceMeter score={3} testId="confidence-meter" />);

    await user.hover(screen.getByTestId("confidence-meter"));

    const tip = await screen.findByRole("tooltip");
    expect(tip).toHaveTextContent("Confidence score");
    expect(tip).toHaveTextContent("3/5");
  });
});
