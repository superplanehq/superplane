import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WidgetScorecard } from "./WidgetScorecard";

// Six rows in chronological order (oldest first, newest last). With
// `aggregation: last` the current value is 98 and the immediately
// previous value is 105 — delta -7, ~-6.7%.
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
          changeCaption: "vs previous",
        }}
      />,
    );

    const root = screen.getByTestId("widget-scorecard");
    expect(root).toHaveTextContent("98");
    const change = screen.getByTestId("widget-scorecard-change");
    expect(change).toHaveTextContent("-7 (-6.7%)");
    // "down" + shrinking value → better polarity → green.
    expect(root).toHaveAttribute("data-scorecard-status", "better");
    expect(screen.getByTestId("widget-scorecard-caption")).toHaveTextContent("vs previous");
  });

  it("renders the change chip using the primary field when sparklineField is not set", () => {
    render(
      <WidgetScorecard
        rows={openCountRows}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "last",
          field: "openCount",
          better: "down",
          showChange: "both",
        }}
      />,
    );

    const change = screen.getByTestId("widget-scorecard-change");
    expect(change).toHaveTextContent("-7 (-6.7%)");
    // Sparkline is opt-in via sparklineField; without it the widget draws
    // no polyline (arrow icons still use <svg>, but the sparkline is the
    // only element that renders a <polyline>).
    expect(screen.getByTestId("widget-scorecard").querySelector("polyline")).toBeNull();
  });

  it("hides the change chip for combining aggregations like sum", () => {
    render(
      <WidgetScorecard
        rows={openCountRows}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "sum",
          field: "openCount",
          better: "down",
          sparklineField: "openCount",
        }}
      />,
    );

    // sum has no natural "immediate previous", so the chip is hidden even
    // though the sparkline still renders from `sparklineField`.
    expect(screen.queryByTestId("widget-scorecard-change")).toBeNull();
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
    // 42 <= target 80 → met for "down" direction → green. Label shows the
    // raw ratio (42 / 80 = 52.5%) so authors can see how much headroom
    // they have before the ceiling.
    expect(root).toHaveAttribute("data-scorecard-status", "better");
    expect(screen.getByTestId("widget-scorecard-progress")).toBeInTheDocument();
    expect(screen.getByTestId("widget-scorecard-progress-label")).toHaveTextContent("52.5% of 80");
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

  it("shows 0 on the change chip when the delta is flat", () => {
    render(
      <WidgetScorecard
        rows={[{ openCount: 42 }, { openCount: 42 }, { openCount: 42 }]}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "last",
          field: "openCount",
          better: "up",
          showChange: "both",
          changeCaption: "vs previous",
        }}
      />,
    );

    const change = screen.getByTestId("widget-scorecard-change");
    expect(change).toHaveAttribute("data-scorecard-change-kind", "flat");
    expect(change).toHaveTextContent("0");
    expect(screen.getByTestId("widget-scorecard")).toHaveAttribute("data-scorecard-status", "flat");
  });

  it("evaluates a dynamic target against the newest filtered row", () => {
    // Widget data sources return newest-first, so index 0 is current.
    const newestFirstRows = [
      { openCount: 98, goal: 80 },
      { openCount: 105, goal: 90 },
      { openCount: 112, goal: 100 },
    ];
    render(
      <WidgetScorecard
        rows={newestFirstRows}
        isLoading={false}
        render={{
          kind: "scorecard",
          aggregation: "first",
          field: "openCount",
          better: "down",
          target: "{{ goal }}",
          showProgress: true,
        }}
      />,
    );

    // Target must come from the newest row (goal: 80), not the oldest (100).
    expect(screen.getByTestId("widget-scorecard-progress-label")).toHaveTextContent("122.5% of 80");
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
