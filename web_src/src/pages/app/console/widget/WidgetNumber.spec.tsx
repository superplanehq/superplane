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
});
