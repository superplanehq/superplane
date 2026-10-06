import { act, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { createElement } from "react";

import { ChartContainer } from "./chart";
import { toChartColorVarName } from "./chartColorVarName";

const CHART_SIZE_WARNING = "of chart should be greater than 0";

describe("toChartColorVarName", () => {
  it("slugifies display names for CSS custom properties", () => {
    expect(toChartColorVarName("Claude Haiku 4.5")).toBe("claude-haiku-4-5");
    expect(toChartColorVarName("Passed")).toBe("passed");
    expect(toChartColorVarName("(empty)")).toBe("empty");
  });

  it("returns empty for blank keys", () => {
    expect(toChartColorVarName("   ")).toBe("empty");
  });
});

describe("ChartContainer", () => {
  const box = { width: 0, height: 0 };
  const originalGetBoundingClientRect = HTMLElement.prototype.getBoundingClientRect;
  const originalResizeObserver = globalThis.ResizeObserver;
  let notifyResize = () => {};
  let warnSpy: ReturnType<typeof vi.spyOn>;

  afterEach(() => {
    HTMLElement.prototype.getBoundingClientRect = originalGetBoundingClientRect;
    globalThis.ResizeObserver = originalResizeObserver;
    warnSpy?.mockRestore();
  });

  it("draws only after the outer box has a positive width and height", async () => {
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    installChartBoxMeasurement(box, (notify) => {
      notifyResize = notify;
    });

    render(
      createElement(ChartContainer, {
        config: { spend: { label: "Spend", color: "#2563eb" } },
        children: createElement("div", { "data-testid": "chart-child" }, "drawn"),
      }),
    );

    expect(screen.queryByTestId("chart-child")).not.toBeInTheDocument();
    expect(hasChartSizeWarning(warnSpy)).toBe(false);

    box.width = 760;
    box.height = 240;
    await act(async () => {
      notifyResize();
    });

    expect(screen.getByTestId("chart-child")).toBeInTheDocument();
    expect(hasChartSizeWarning(warnSpy)).toBe(false);

    box.width = 0;
    box.height = 0;
    await act(async () => {
      notifyResize();
    });

    expect(screen.getByTestId("chart-child")).toBeInTheDocument();
    expect(hasChartSizeWarning(warnSpy)).toBe(false);
  });
});

function installChartBoxMeasurement(box: { width: number; height: number }, setNotify: (notify: () => void) => void) {
  const originalGetBoundingClientRect = HTMLElement.prototype.getBoundingClientRect;
  HTMLElement.prototype.getBoundingClientRect = function getBoundingClientRect() {
    if (this.getAttribute("data-slot") === "chart") {
      return chartDomRect(box.width, box.height);
    }
    return originalGetBoundingClientRect.call(this);
  };

  globalThis.ResizeObserver = class {
    constructor(callback: ResizeObserverCallback) {
      setNotify(() => {
        callback([], this as unknown as ResizeObserver);
      });
    }
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

function chartDomRect(width: number, height: number): DOMRect {
  return {
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    right: width,
    bottom: height,
    width,
    height,
    toJSON() {
      return { width, height };
    },
  } as DOMRect;
}

function hasChartSizeWarning(spy: ReturnType<typeof vi.spyOn>) {
  return spy.mock.calls.some((args: unknown[]) =>
    args.some((arg) => typeof arg === "string" && arg.includes(CHART_SIZE_WARNING)),
  );
}
