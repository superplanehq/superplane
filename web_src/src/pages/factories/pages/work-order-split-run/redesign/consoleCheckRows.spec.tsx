import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { ConsoleCheckRows } from "./consoleCheckRows";

function check(
  overrides: Partial<WorkOrderCheckPresentation> & Pick<WorkOrderCheckPresentation, "id" | "key" | "name">,
): WorkOrderCheckPresentation {
  return {
    score: 5,
    maxScore: 5,
    level: "positive",
    ...overrides,
  };
}

describe("ConsoleCheckRows", () => {
  it("shows the merge confidence score and every metric", () => {
    render(
      <ConsoleCheckRows
        checks={[
          check({ id: "confidence", key: "confidence", name: "Confidence score", score: 2, level: "caution" }),
          check({ id: "coverage", key: "code-coverage", name: "Code quality", score: 80, maxScore: 100 }),
          check({ id: "risk", key: "risk-review", name: "Risk score", score: 3, level: "caution" }),
          check({ id: "performance", key: "performance-review", name: "Performance" }),
          check({ id: "security", key: "security-review", name: "Security" }),
          check({ id: "drift", key: "drift-review", name: "Drift", score: 2 }),
        ]}
      />,
    );

    expect(screen.queryByText("Confidence score")).not.toBeInTheDocument();
    expect(screen.queryByText("Code quality")).not.toBeInTheDocument();
    const score = screen.getByTestId("split-run-check-merge-confidence");
    expect(within(score).getByRole("heading", { name: "Merge confidence" })).toBeInTheDocument();
    expect(within(score).getByTestId("split-run-merge-confidence-meter").parentElement).toHaveTextContent("Caution");
    expect(
      screen.getByTestId("split-run-merge-confidence-meter").querySelectorAll("[data-filled='true']"),
    ).toHaveLength(3);
    expect(screen.getByTestId("split-run-merge-confidence-meter")).toHaveTextContent("3");
    expect(screen.getByRole("button", { name: /Risk score/ })).toHaveAccessibleName(/Caution.*Read the reason/);
    expect(screen.getByRole("button", { name: /Drift/ })).toHaveAccessibleName(/Healthy/);
    expect(screen.getByText("Risk score")).toBeInTheDocument();
    expect(screen.getByText("Performance")).toBeInTheDocument();
    expect(screen.getByText("Security")).toBeInTheDocument();
    expect(screen.getByText("Drift")).toBeInTheDocument();
  });
});
