import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WidgetScorecard } from "./WidgetScorecard";

const openCountRows = [
  { openCount: 127 },
  { openCount: 120 },
  { openCount: 118 },
  { openCount: 112 },
  { openCount: 105 },
  { openCount: 98 },
];

describe("WidgetScorecard", () => {
  it("renders the aggregated value, change chip, and caption for a shrinking KPI", () => {
    render(
      <WidgetScorecard
        rows={openCountRows}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "last",
          field: "openCount",
          label: "Open UX papercuts",
          format: "number",
          better: "down",
          sparklineField: "openCount",
          showChange: "both",
          changeCaption: "vs start of range",
        }}
      />,
    );

    const root = screen.getByTestId("widget-scorecard");
    expect(root).toHaveTextContent("98");
    const change = screen.getByTestId("widget-scorecard-change");
    expect(change).toHaveTextContent("-29 (-22.8%)");
    // "down" + shrinking value → better polarity → green.
    expect(root).toHaveAttribute("data-scorecard-status", "better");
    expect(screen.getByTestId("widget-scorecard-caption")).toHaveTextContent("vs start of range");
  });

  it("hides the change chip when only one point is loaded", () => {
    render(
      <WidgetScorecard
        rows={[{ openCount: 42 }]}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "last",
          field: "openCount",
          label: "Open items",
          sparklineField: "openCount",
        }}
      />,
    );

    expect(screen.queryByTestId("widget-scorecard-change")).toBeNull();
  });

  it("uses target-based status coloring when no change baseline is available", () => {
    render(
      <WidgetScorecard
        rows={[{ openCount: 42 }]}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "last",
          field: "openCount",
          label: "Open items",
          better: "down",
          target: "80",
          showProgress: true,
        }}
      />,
    );

    const root = screen.getByTestId("widget-scorecard");
    // 42 <= target 80 → met for "down" direction → green.
    expect(root).toHaveAttribute("data-scorecard-status", "better");
    expect(screen.getByTestId("widget-scorecard-progress")).toBeInTheDocument();
    expect(screen.getByTestId("widget-scorecard-progress-label")).toHaveTextContent("100% of 80");
  });

  it("colors the sparkline and value red when the change is worse", () => {
    render(
      <WidgetScorecard
        rows={openCountRows}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "last",
          field: "openCount",
          label: "Open items",
          // "up" (higher is better) + shrinking value → worse polarity.
          better: "up",
          sparklineField: "openCount",
        }}
      />,
    );

    const root = screen.getByTestId("widget-scorecard");
    expect(root).toHaveAttribute("data-scorecard-status", "worse");
  });

  it("renders the empty state when no value can be aggregated", () => {
    render(
      <WidgetScorecard
        rows={[]}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "last",
          field: "openCount",
          label: "Open items",
        }}
      />,
    );

    expect(screen.getByTestId("widget-scorecard-empty")).toBeInTheDocument();
  });
});
