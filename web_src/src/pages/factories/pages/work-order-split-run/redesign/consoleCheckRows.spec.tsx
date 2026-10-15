import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { ConsoleCheckRows } from "./consoleCheckRows";

const checks: WorkOrderCheckPresentation[] = [
  { id: "risk", key: "risk-review", name: "Blast radius", score: 1, maxScore: 5, level: "positive" },
  { id: "performance", key: "performance-review", name: "Performance", score: 3, maxScore: 5, level: "caution" },
  {
    id: "security",
    key: "security-review",
    name: "Security",
    score: 2,
    maxScore: 5,
    level: "critical",
    summary: "The retry endpoint does not check workspace membership.",
  },
  { id: "drift", key: "drift-review", name: "Drift", score: 1, maxScore: 5, level: "positive" },
  { id: "reversibility", key: "reversibility-review", name: "Reversibility", score: 5, maxScore: 5, level: "positive" },
];

describe("ConsoleCheckRows", () => {
  it("colors each summary bar independently and expands status icons with evidence", async () => {
    const user = userEvent.setup();
    render(<ConsoleCheckRows checks={checks} />);

    const header = screen.getByRole("button", { name: "Merge confidence" });
    expect(header).toHaveAttribute("aria-expanded", "false");
    const bars = screen.getByTestId("redesign-console-checks-meter").children;
    expect(bars).toHaveLength(5);
    expect(bars[0]).toHaveClass("bg-emerald-500");
    expect(bars[1]).toHaveClass("bg-amber-500");
    expect(bars[2]).toHaveClass("bg-red-500");
    expect(bars[3]).toHaveClass("bg-emerald-500");
    expect(bars[4]).toHaveClass("bg-emerald-500");
    expect(screen.getByText("2 checks need review")).toBeInTheDocument();
    expect(screen.queryByText("Blast radius")).not.toBeInTheDocument();

    await user.click(header);

    expect(header).toHaveAttribute("aria-expanded", "true");
    expect(within(screen.getByTestId("split-run-check-risk")).getByRole("img", { name: "Passed" })).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-check-performance")).getByRole("img", { name: "Needs review" }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-check-security")).getByRole("img", { name: "Needs attention" }),
    ).toBeInTheDocument();
    expect(screen.getByText("The retry endpoint does not check workspace membership.")).toBeInTheDocument();
    expect(screen.getByText("Drift from Specification")).toBeInTheDocument();
    expect(screen.queryByText("Missed")).not.toBeInTheDocument();
    expect(screen.queryByText("2/5")).not.toBeInTheDocument();

    await user.click(header);
    expect(screen.queryByText("Blast radius")).not.toBeInTheDocument();
  });

  it("does not describe a neutral check as passed", () => {
    render(<ConsoleCheckRows checks={[{ ...checks[0], level: "neutral" }]} defaultOpen />);
    expect(screen.getByRole("img", { name: "Not assessed" })).toBeInTheDocument();
    expect(screen.queryByText("All checks passed")).not.toBeInTheDocument();
  });

  it("shows only reported checks and hides an empty section", () => {
    const view = render(<ConsoleCheckRows checks={[checks[0]]} />);
    expect(screen.getByTestId("redesign-console-checks-meter").children).toHaveLength(1);
    expect(screen.getByText("All checks passed")).toBeInTheDocument();
    view.rerender(<ConsoleCheckRows checks={[]} />);
    expect(screen.queryByRole("button", { name: "Merge confidence" })).not.toBeInTheDocument();
  });
});
