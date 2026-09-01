import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CanvasPageLoadingOverlay } from "./CanvasPageLoadingOverlay";

describe("CanvasPageLoadingOverlay", () => {
  it("renders a full-page loading message", () => {
    render(<CanvasPageLoadingOverlay message="Loading version..." testId="canvas-version-loading" />);

    expect(screen.getByTestId("canvas-version-loading")).toBeInTheDocument();
    expect(screen.getByText("Loading version...")).toBeInTheDocument();
  });

  it("can cover the canvas with an opaque loading screen", () => {
    render(<CanvasPageLoadingOverlay message="Loading canvas..." opaque testId="factory-configure-enter-loading" />);

    expect(screen.getByTestId("factory-configure-enter-loading")).toHaveClass("bg-background");
    expect(screen.getByText("Loading canvas...")).toBeInTheDocument();
  });
});
