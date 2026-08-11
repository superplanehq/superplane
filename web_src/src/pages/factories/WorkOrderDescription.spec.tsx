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
});
