import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router";
import type { CanvasesCanvasRun } from "@/api-client";
import { RunRow } from "./RunRow";

function renderRunRow(run: CanvasesCanvasRun) {
  return render(
    <MemoryRouter>
      <RunRow
        run={run}
        triggerName="Push"
        title="Run 1"
        status="passed"
        isSelected={false}
        componentIconMap={{}}
        onSelectRun={vi.fn()}
      />
    </MemoryRouter>,
  );
}

describe("RunRow — replay badge", () => {
  it("badges a replay run", () => {
    renderRunRow({ id: "run-replay", canvasId: "canvas-1", isReplay: true });

    expect(screen.getByTestId("run-replay-badge")).toHaveTextContent(/replay/i);
  });

  it("does not badge an ordinary run", () => {
    renderRunRow({ id: "run-normal", canvasId: "canvas-1", isReplay: false });

    expect(screen.queryByTestId("run-replay-badge")).not.toBeInTheDocument();
  });

  it("does not badge a run whose replay flag is absent entirely (boundary)", () => {
    renderRunRow({ id: "run-legacy", canvasId: "canvas-1" });

    expect(screen.queryByTestId("run-replay-badge")).not.toBeInTheDocument();
  });
});
