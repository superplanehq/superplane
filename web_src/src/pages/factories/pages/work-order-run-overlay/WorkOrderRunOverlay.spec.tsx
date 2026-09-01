import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { WorkOrderRunOverlayPlayground } from "./WorkOrderRunOverlayPlayground";

function renderPlayground(initialConcept: "a" | "b" | "c" = "a") {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <WorkOrderRunOverlayPlayground initialConcept={initialConcept} />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WorkOrderRunOverlayPlayground", () => {
  it("opens concept A as a run overlay without ticket chrome", () => {
    renderPlayground("a");

    expect(screen.getByTestId("run-overlay-concept-a")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Ship idempotent refund retries" })).toBeInTheDocument();
    expect(screen.getByText(/Run 4182/)).toBeInTheDocument();
    expect(screen.queryByText("Factory Lines")).not.toBeInTheDocument();
    expect(screen.queryByText("Assignees")).not.toBeInTheDocument();
    expect(screen.getByText("Code quality")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Continue to Verify" })).toBeInTheDocument();
  });

  it("moves from Implement to Verify on continue", async () => {
    const user = userEvent.setup();
    renderPlayground("a");

    await user.click(screen.getByRole("button", { name: "Continue to Verify" }));

    expect(screen.getByRole("button", { name: "Complete run" })).toBeInTheDocument();
    expect(screen.getByText(/Verify starts after Implement finishes/)).toBeInTheDocument();
  });

  it("opens concept B with a phase rail and concept C with a canvas", async () => {
    const user = userEvent.setup();
    renderPlayground("b");

    expect(screen.getByTestId("run-overlay-concept-b")).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Run phases" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /C · Live canvas/ }));

    expect(screen.getByTestId("run-overlay-concept-c")).toBeInTheDocument();
    expect(screen.getByTestId("run-overlay-compact-canvas")).toBeInTheDocument();
  });
});
