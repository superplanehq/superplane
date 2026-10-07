import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { ConsoleCheckRows } from "./consoleCheckRows";

function filledBars(id: string) {
  return screen.getByTestId(`split-run-check-${id}`).querySelectorAll("[data-bar-filled='true']");
}

function check(
  overrides: Partial<WorkOrderCheckPresentation> & Pick<WorkOrderCheckPresentation, "id" | "name">,
): WorkOrderCheckPresentation {
  return {
    score: 4,
    maxScore: 5,
    level: "positive",
    ...overrides,
  };
}

function mergeConfidenceHeader() {
  return screen.getByRole("button", { name: "Merge confidence" });
}

function openChecks() {
  fireEvent.click(mergeConfidenceHeader());
}

describe("ConsoleCheckRows", () => {
  it("lists each check with its score and status", () => {
    render(
      <ConsoleCheckRows
        checks={[
          check({ id: "confidence", key: "confidence", name: "Confidence score", level: "positive" }),
          check({ id: "risk", key: "risk-review", name: "Risk score", level: "critical" }),
        ]}
      />,
    );

    const header = mergeConfidenceHeader();
    expect(header).toHaveAttribute("aria-expanded", "false");
    expect(header.querySelectorAll("[data-bar-filled='true']")).toHaveLength(1);
    expect(screen.getByText("1 of 2 indicates high caution")).toBeInTheDocument();
    expect(screen.queryByText("A check failed")).not.toBeInTheDocument();
    expect(screen.queryByText("Blast radius")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-check-confidence")).not.toBeInTheDocument();

    openChecks();

    expect(header).toHaveAttribute("aria-expanded", "true");
    expect(filledBars("confidence")).toHaveLength(3);
    expect(filledBars("risk")).toHaveLength(1);
    expect(screen.getByText("Blast radius")).toBeInTheDocument();
    expect(screen.queryByText("High")).not.toBeInTheDocument();
    expect(screen.queryByText("4/5")).not.toBeInTheDocument();

    openChecks();

    expect(header).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Blast radius")).not.toBeInTheDocument();
  });

  it("starts closed again after the section leaves the page", () => {
    const checks = [check({ id: "risk", key: "risk-review", name: "Risk score", level: "critical" })];
    const view = render(<ConsoleCheckRows checks={checks} />);
    openChecks();
    expect(screen.getByText("Blast radius")).toBeInTheDocument();

    view.unmount();
    render(<ConsoleCheckRows checks={checks} />);

    expect(mergeConfidenceHeader()).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Blast radius")).not.toBeInTheDocument();
  });

  it("uses a result word for drift, performance, and security", () => {
    render(
      <ConsoleCheckRows
        checks={[
          check({ id: "drift", key: "drift-review", name: "Drift", score: 2, level: "positive" }),
          check({ id: "performance", key: "performance-review", name: "Performance", level: "positive" }),
          check({ id: "security", key: "security-review", name: "Security", level: "caution" }),
        ]}
      />,
    );

    openChecks();

    expect(filledBars("drift")).toHaveLength(3);
    expect(filledBars("performance")).toHaveLength(3);
    expect(filledBars("security")).toHaveLength(2);
    expect(screen.getByText("1 of 3 indicates higher caution")).toBeInTheDocument();
    expect(screen.queryByText("Close")).not.toBeInTheDocument();
    expect(screen.queryByText("Met")).not.toBeInTheDocument();
    expect(screen.queryByText("Partial")).not.toBeInTheDocument();
  });

  it("shows the result and message when the row is hovered", async () => {
    const user = userEvent.setup();
    render(
      <ConsoleCheckRows
        checks={[
          check({
            id: "risk",
            key: "risk-review",
            name: "Risk score",
            score: 1,
            level: "positive",
            summary: "Lower risk because the pull request is documentation only.",
          }),
          check({
            id: "performance",
            key: "performance-review",
            name: "Performance",
            score: 5,
            level: "positive",
            summary: "The change adds no runtime work.",
          }),
          check({
            id: "security",
            key: "security-review",
            name: "Security",
            score: 1,
            level: "critical",
            summary: "The change writes a secret into the repository.",
          }),
        ]}
      />,
    );

    openChecks();
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
    expect(screen.queryByText("1/5")).not.toBeInTheDocument();

    await user.hover(screen.getByText("Blast radius"));
    const riskTip = await screen.findByRole("tooltip");
    expect(riskTip).toHaveTextContent("Low");
    expect(riskTip).toHaveTextContent("Lower risk because the pull request is documentation only.");
    expect(riskTip).not.toHaveTextContent("1/5");
    expect(within(riskTip).getByText("Low")).toHaveClass("text-emerald-400");

    await user.hover(screen.getByText("Performance"));
    const performanceTip = await screen.findByRole("tooltip");
    expect(performanceTip).toHaveTextContent("Met");
    expect(performanceTip).toHaveTextContent("The change adds no runtime work.");
    expect(performanceTip).not.toHaveTextContent("5/5");

    await user.hover(screen.getByText("Security"));
    const securityTip = await screen.findByRole("tooltip");
    expect(within(securityTip).getByText("Missed")).toHaveClass("text-red-400");

    await user.click(screen.getByText("Blast radius"));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("shows the result when a check row receives keyboard focus", async () => {
    const user = userEvent.setup();
    render(
      <ConsoleCheckRows
        checks={[
          check({
            id: "risk",
            key: "risk-review",
            name: "Risk score",
            score: 1,
            level: "positive",
            summary: "Lower risk because the pull request is documentation only.",
          }),
        ]}
      />,
    );

    await user.click(mergeConfidenceHeader());
    await user.tab();

    const row = screen.getByTestId("split-run-check-risk");
    expect(row).toHaveFocus();
    expect(row).toHaveAccessibleName(
      "Blast radius. Low. Lower risk because the pull request is documentation only.",
    );
    const tip = await screen.findByRole("tooltip");
    expect(tip).toHaveTextContent("Low");
    expect(tip).toHaveTextContent("Lower risk because the pull request is documentation only.");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("describes a calm result as low caution", () => {
    render(
      <ConsoleCheckRows
        checks={[
          check({ id: "drift", key: "drift-review", name: "Drift", level: "positive" }),
          check({ id: "performance", key: "performance-review", name: "Performance", level: "positive" }),
          check({ id: "security", key: "security-review", name: "Security", level: "positive" }),
          check({ id: "reversibility", key: "reversibility-review", name: "Reversibility", level: "positive" }),
        ]}
      />,
    );

    expect(screen.getByText("All checks indicate high confidence")).toBeInTheDocument();
    expect(screen.queryByText(/successful/)).not.toBeInTheDocument();
  });
});
