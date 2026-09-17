import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import { ConfidenceAnalyzingIndicator, ConfidenceMeter, ScorePairMeter } from "./ConfidenceMeter";

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

  it("uses the label for the meter name and tooltip", async () => {
    const user = userEvent.setup();
    render(<ConfidenceMeter score={2} label="Clarity score" testId="clarity-meter" />);

    expect(screen.getByTestId("clarity-meter")).toHaveAttribute("aria-label", "Clarity score");
    await user.hover(screen.getByTestId("clarity-meter"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Clarity score");
  });
});

describe("ScorePairMeter", () => {
  it("stacks a Clarity row over a Confidence row", () => {
    render(<ScorePairMeter clarity={4} confidence={2} testId="pair" />);

    const group = screen.getByTestId("pair");
    expect(group).toHaveAttribute("role", "group");
    expect(group).toHaveAttribute("aria-label", "Clarity and Confidence");
    const clarity = screen.getByTestId("pair-clarity");
    const confidence = screen.getByTestId("pair-confidence");
    expect(clarity).toHaveAttribute("aria-label", "Clarity score");
    expect(clarity).toHaveAttribute("aria-valuenow", "4");
    expect(clarity.querySelectorAll("[data-filled='true']")).toHaveLength(4);
    expect(confidence).toHaveAttribute("aria-label", "Confidence score");
    expect(confidence).toHaveAttribute("aria-valuenow", "2");
    expect(confidence.querySelectorAll("[data-filled='true']")).toHaveLength(2);
    expect(group.compareDocumentPosition(clarity) & Node.DOCUMENT_POSITION_CONTAINED_BY).toBeTruthy();
    expect(clarity.compareDocumentPosition(confidence) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("lists both scores in one tooltip", async () => {
    const user = userEvent.setup();
    render(<ScorePairMeter clarity={4} confidence={2} testId="pair" />);

    await user.hover(screen.getByTestId("pair"));

    const tip = await screen.findByRole("tooltip");
    expect(tip).toHaveTextContent("Clarity score");
    expect(tip).toHaveTextContent("4/5");
    expect(tip).toHaveTextContent("Confidence score");
    expect(tip).toHaveTextContent("2/5");
  });

  it("leaves a row empty when one score is missing", () => {
    render(<ScorePairMeter clarity={3} testId="pair" />);

    const confidence = screen.getByTestId("pair-confidence");
    expect(confidence).not.toHaveAttribute("aria-valuenow");
    expect(confidence).toHaveAttribute("aria-valuetext", "No score yet");
    expect(confidence.querySelectorAll("[data-filled='true']")).toHaveLength(0);
  });
});

describe("ConfidenceAnalyzingIndicator", () => {
  it("keeps the shared chip as a matrix only", () => {
    render(<ConfidenceAnalyzingIndicator testId="analyzing" />);

    const indicator = screen.getByTestId("analyzing");
    expect(indicator.querySelector(".t-matrix")).not.toBeNull();
    expect(indicator).not.toHaveTextContent("Analyzing");
    expect(indicator).not.toHaveTextContent("Refining");
  });

  it("shows thinking states next to the matrix on the card", () => {
    render(<ConfidenceAnalyzingIndicator testId="analyzing" showThinkingStates />);

    expect(screen.getByTestId("analyzing")).toHaveTextContent("Analyzing");
  });
});
