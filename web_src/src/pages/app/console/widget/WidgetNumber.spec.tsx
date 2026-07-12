import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { CONSOLE_WIDGET_LABEL_CLASSES } from "../consoleTableStyles";
import { WidgetNumber } from "./WidgetNumber";

describe("WidgetNumber", () => {
  it("renders metric labels with the same typography as widget table headers", () => {
    render(
      <WidgetNumber
        variant="inline"
        rows={[{ total: 3 }]}
        isLoading={false}
        render={{ kind: "number", aggregation: "sum", field: "total", label: "Pending Action" }}
      />,
    );

    const label = screen.getByTestId("widget-number-label");
    expect(label).toHaveTextContent("Pending Action");
    expect(label.className).toBe(CONSOLE_WIDGET_LABEL_CLASSES);
    expect(label.className).toContain("text-[11px]");
    expect(label.className).toContain("font-semibold");
    expect(label.className).not.toContain("text-xs");
    expect(label.className).not.toContain("font-medium");
  });

  it("keeps a prefix-only currency symbol flush against the value (no flex gap)", () => {
    render(
      <WidgetNumber
        rows={[{ cost: 1234 }]}
        isLoading={false}
        render={{ kind: "number", aggregation: "sum", field: "cost", format: "number", prefix: "R$" }}
      />,
    );

    const root = screen.getByTestId("widget-number");
    const valueRow = root.querySelector(".flex.items-baseline");
    expect(valueRow).not.toBeNull();
    expect(valueRow!.className).not.toContain("gap-");
    expect(root).toHaveTextContent("R$1,234");
  });

  it("uses a baseline gap when a suffix is present", () => {
    render(
      <WidgetNumber
        rows={[{ cost: 42 }]}
        isLoading={false}
        render={{
          kind: "number",
          aggregation: "sum",
          field: "cost",
          format: "number",
          prefix: "$",
          suffix: " /mo",
        }}
      />,
    );

    const root = screen.getByTestId("widget-number");
    const valueRow = root.querySelector(".flex.items-baseline");
    expect(valueRow).not.toBeNull();
    expect(valueRow!.className).toContain("gap-0.5");
    expect(root).toHaveTextContent("$42 /mo");
  });

  it("omits null/blank sparkline points so the series matches aggregateNumber", () => {
    // Bare Number(null)/Number("") would inject two zero points and draw a
    // 4-point sparkline that disagrees with sum=13 over the same rows.
    render(
      <WidgetNumber
        rows={[{ total: null }, { total: "" }, { total: 5 }, { total: 8 }]}
        isLoading={false}
        render={{
          kind: "number",
          aggregation: "sum",
          field: "total",
          sparklineField: "total",
        }}
      />,
    );

    const root = screen.getByTestId("widget-number");
    expect(root).toHaveTextContent("13");
    const polyline = root.querySelector("polyline");
    expect(polyline).not.toBeNull();
    // Two finite points → one segment (two "x,y" pairs), not four from coerced zeros.
    expect(polyline!.getAttribute("points")!.trim().split(/\s+/)).toHaveLength(2);
  });
});
