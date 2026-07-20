import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@xyflow/react", () => ({
  Panel: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  useReactFlow: vi.fn(() => ({
    zoomTo: vi.fn(),
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    fitView: vi.fn(),
    getNodes: vi.fn(() => []),
  })),
  useStore: vi.fn((selector: (state: { minZoom: number; maxZoom: number }) => unknown) =>
    selector({ minZoom: 0.1, maxZoom: 1.5 }),
  ),
  useViewport: vi.fn(() => ({ zoom: 1, x: 0, y: 0 })),
  getNodesBounds: vi.fn(() => ({ x: 0, y: 0, width: 0, height: 0 })),
  getViewportForBounds: vi.fn(() => ({ x: 0, y: 0, zoom: 1 })),
}));

vi.mock("html-to-image", () => ({
  toPng: vi.fn(),
}));

import { ZoomSlider } from "./zoom-slider";

describe("ZoomSlider auto-focus toggle", () => {
  it("does not render the auto-focus toggle when no callback is provided", () => {
    const { queryByTestId } = render(<ZoomSlider usePanel={false} />);

    expect(queryByTestId("canvas-auto-focus-toggle")).toBeNull();
  });

  it("marks the toggle as pressed and labels it for disabling when auto-focus is enabled", () => {
    const { getByTestId } = render(<ZoomSlider usePanel={false} isAutoFocusEnabled onAutoFocusToggle={vi.fn()} />);

    const button = getByTestId("canvas-auto-focus-toggle");
    expect(button.getAttribute("aria-pressed")).toBe("true");
    expect(button.getAttribute("aria-label")).toBe("Disable auto-focus on selection");
  });

  it("marks the toggle as not pressed and labels it for enabling when auto-focus is disabled", () => {
    const { getByTestId } = render(
      <ZoomSlider usePanel={false} isAutoFocusEnabled={false} onAutoFocusToggle={vi.fn()} />,
    );

    const button = getByTestId("canvas-auto-focus-toggle");
    expect(button.getAttribute("aria-pressed")).toBe("false");
    expect(button.getAttribute("aria-label")).toBe("Enable auto-focus on selection");
  });

  it("invokes the callback when clicked", async () => {
    const onAutoFocusToggle = vi.fn();
    const user = userEvent.setup();
    const { getByTestId } = render(
      <ZoomSlider usePanel={false} isAutoFocusEnabled onAutoFocusToggle={onAutoFocusToggle} />,
    );

    await user.click(getByTestId("canvas-auto-focus-toggle"));

    expect(onAutoFocusToggle).toHaveBeenCalledTimes(1);
  });
});
