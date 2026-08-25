import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { WorkOrderDescription } from "./WorkOrderDescription";

let notifyResize: () => void;

class MockResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    notifyResize = () => callback([], this as unknown as ResizeObserver);
  }

  observe() {}
  unobserve() {}
  disconnect() {}
}

describe("WorkOrderDescription", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("updates the collapse control when the rendered content height changes", () => {
    let contentHeight = 300;
    render(<WorkOrderDescription description="# Test description" />);

    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    expect(content).not.toBeNull();
    Object.defineProperty(content!, "scrollHeight", {
      configurable: true,
      get: () => contentHeight,
    });

    act(() => notifyResize());
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();

    contentHeight = 100;
    act(() => notifyResize());
    expect(screen.queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
  });

  it("keeps the body open when it fits the scroll pane", () => {
    let contentHeight = 360;
    render(
      <div data-testid="description-pane" style={{ overflowY: "auto", padding: "24px 0" }}>
        <WorkOrderDescription description="# Test description" />
      </div>,
    );

    const pane = screen.getByTestId("description-pane");
    Object.defineProperty(pane, "clientHeight", { configurable: true, get: () => 520 });
    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => contentHeight });

    act(() => notifyResize());
    expect(screen.queryByRole("button", { name: /show more/i })).not.toBeInTheDocument();
    expect(content).not.toHaveStyle({ maxHeight: "220px" });
  });

  it("fills the leftover pane before Show more when the body is taller", () => {
    render(
      <div data-testid="description-pane" style={{ overflowY: "auto", padding: "24px 0" }}>
        <WorkOrderDescription description="# Test description" />
      </div>,
    );

    const pane = screen.getByTestId("description-pane");
    Object.defineProperty(pane, "clientHeight", { configurable: true, get: () => 520 });
    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 640 });

    act(() => notifyResize());
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(content).toHaveStyle({ maxHeight: "440px" });
  });

  it("leaves room for checks in the leftover pane", () => {
    render(
      <div data-testid="description-pane" style={{ overflowY: "auto", padding: "24px 0" }}>
        <WorkOrderDescription description="# Test description" />
        <section data-testid="checks" style={{ marginTop: 40 }}>
          Checks
        </section>
      </div>,
    );

    const pane = screen.getByTestId("description-pane");
    Object.defineProperty(pane, "clientHeight", { configurable: true, get: () => 520 });
    Object.defineProperty(screen.getByTestId("checks"), "offsetHeight", { configurable: true, get: () => 120 });
    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 400 });

    act(() => notifyResize());
    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(content).toHaveStyle({ maxHeight: "280px" });
  });
});
