import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { CanvasRequestFailure, EarlyCanvasView } from "./CanvasRequestFailure";

describe("canvas load failure", () => {
  it("shows the request failure instead of a missing canvas", () => {
    render(
      <EarlyCanvasView
        canvas={undefined}
        canvasLoading={false}
        canvasError={Object.assign(new Error("unavailable"), { status: 503 })}
        bootstrapLoading
        showDraftCanvasLoadingOverlay={false}
        onRetry={() => undefined}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("Could not load this canvas");
    expect(screen.getByText("The request failed.")).toBeInTheDocument();
    expect(screen.queryByText("Canvas not found")).toBeNull();
    expect(screen.queryByText("Loading canvas...")).toBeNull();
  });

  it("keeps a missing canvas on the not-found message", () => {
    render(
      <EarlyCanvasView
        canvas={undefined}
        canvasLoading={false}
        canvasError={Object.assign(new Error("Not Found"), { status: 404 })}
        bootstrapLoading={false}
        showDraftCanvasLoadingOverlay={false}
        onRetry={() => undefined}
      />,
    );

    expect(screen.getByText("Canvas not found")).toBeInTheDocument();
    expect(screen.queryByTestId("canvas-request-failure")).toBeNull();
  });

  it("retries the canvas request", async () => {
    const onRetry = vi.fn();
    const user = userEvent.setup();
    render(<CanvasRequestFailure onRetry={onRetry} />);

    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
