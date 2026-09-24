import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import { CardReadinessMark, CardScoreBadges, ReadinessDot } from "./ReadinessMark";

describe("ReadinessDot", () => {
  it("is decorative and carries the tone colour", () => {
    const { container } = render(<ReadinessDot tone="caution" />);

    const dot = container.firstElementChild;
    expect(dot).toHaveAttribute("aria-hidden", "true");
    expect(dot?.className).toContain("status-waiting-dot");
  });
});

describe("CardReadinessMark", () => {
  it("shows one short verdict word and reads both scores to assistive tech", () => {
    render(<CardReadinessMark clarity={4} confidence={2} testId="mark" />);

    const mark = screen.getByTestId("mark");
    expect(mark).toHaveAttribute("data-tone", "caution");
    expect(mark).toHaveTextContent("Review");
    expect(mark).not.toHaveTextContent("4");
    expect(mark).toHaveAttribute(
      "aria-label",
      "Review before you start. Clarity score 4 of 5. Confidence score 2 of 5",
    );
  });

  it("leads the tooltip with the headline, then both scores", async () => {
    const user = userEvent.setup();
    render(<CardReadinessMark clarity={4} confidence={2} testId="mark" />);

    await user.hover(screen.getByTestId("mark"));

    const tip = await screen.findByRole("tooltip");
    expect(tip.textContent?.startsWith("Review before you start")).toBe(true);
    expect(tip).toHaveTextContent("Clarity score");
    expect(tip).toHaveTextContent("4/5");
    expect(tip).toHaveTextContent("Confidence score");
    expect(tip).toHaveTextContent("2/5");
  });

  it("says no score yet for a missing score", () => {
    render(<CardReadinessMark clarity={3} testId="mark" />);

    const mark = screen.getByTestId("mark");
    expect(mark).toHaveAttribute("data-tone", "caution");
    expect(mark).toHaveAttribute(
      "aria-label",
      "Review the plan before you start. Clarity score 3 of 5. Confidence score no score yet",
    );
  });

  it("calls a low Clarity not ready and a pair of high scores ready", () => {
    const { rerender } = render(<CardReadinessMark clarity={2} confidence={5} testId="mark" />);
    expect(screen.getByTestId("mark")).toHaveAttribute("data-tone", "blocked");
    expect(screen.getByTestId("mark")).toHaveTextContent("Not ready");

    rerender(<CardReadinessMark clarity={5} confidence={4} testId="mark" />);
    expect(screen.getByTestId("mark")).toHaveAttribute("data-tone", "ready");
    expect(screen.getByTestId("mark")).toHaveTextContent("Ready");
  });
});

describe("CardScoreBadges", () => {
  it("shows a name and number badge per score, tinted by band", () => {
    render(<CardScoreBadges clarity={5} confidence={2} testId="badges" />);

    const clarity = screen.getByTestId("badges-clarity");
    const confidence = screen.getByTestId("badges-confidence");
    expect(clarity).toHaveTextContent("Clarity5");
    expect(clarity).toHaveClass("text-emerald-700");
    expect(confidence).toHaveTextContent("Confidence2");
    expect(confidence).toHaveClass("text-red-700");
    expect(screen.getByTestId("badges")).toHaveAttribute("data-tone", "caution");
    expect(screen.getByTestId("badges")).toHaveAttribute(
      "aria-label",
      "Review before you start. Clarity score 5 of 5. Confidence score 2 of 5",
    );
  });

  it("mutes a badge without a score and shows a dash", () => {
    render(<CardScoreBadges confidence={4} testId="badges" />);

    const clarity = screen.getByTestId("badges-clarity");
    expect(clarity).toHaveTextContent("Clarity–");
    expect(clarity).toHaveClass("text-muted-foreground");
    expect(screen.getByTestId("badges-confidence")).toHaveClass("text-emerald-700");
  });

  it("hides a score when that Planning toggle is off", () => {
    render(<CardScoreBadges clarity={5} confidence={2} showClarity={false} testId="badges" />);

    expect(screen.queryByTestId("badges-clarity")).not.toBeInTheDocument();
    expect(screen.getByTestId("badges-confidence")).toHaveTextContent("Confidence2");
  });

  it("keeps the verdict headline in the tooltip", async () => {
    const user = userEvent.setup();
    render(<CardScoreBadges clarity={2} confidence={5} testId="badges" />);

    await user.hover(screen.getByTestId("badges"));

    const tip = await screen.findByRole("tooltip");
    expect(tip).toHaveTextContent("This task is not ready to start");
  });
});
