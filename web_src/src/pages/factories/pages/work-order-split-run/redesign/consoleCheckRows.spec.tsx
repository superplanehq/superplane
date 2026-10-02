import { render, screen } from "@testing-library/react";
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

    const header = screen.getByRole("button", { name: "Merge confidence" });
    expect(header).toHaveAttribute("aria-expanded", "true");
    expect(header.querySelectorAll("[data-bar-filled='true']")).toHaveLength(1);
    expect(screen.getByText("1 of 2 indicates high caution")).toBeInTheDocument();
    expect(screen.queryByText("A check failed")).not.toBeInTheDocument();
    expect(filledBars("confidence")).toHaveLength(3);
    expect(filledBars("risk")).toHaveLength(1);
    expect(screen.queryByText("High")).not.toBeInTheDocument();
    expect(screen.queryByText("4/5")).not.toBeInTheDocument();
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

    expect(filledBars("drift")).toHaveLength(3);
    expect(filledBars("performance")).toHaveLength(3);
    expect(filledBars("security")).toHaveLength(2);
    expect(screen.getByText("1 of 3 indicates higher caution")).toBeInTheDocument();
    expect(screen.queryByText("Close")).not.toBeInTheDocument();
    expect(screen.queryByText("Met")).not.toBeInTheDocument();
    expect(screen.queryByText("Partial")).not.toBeInTheDocument();
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
