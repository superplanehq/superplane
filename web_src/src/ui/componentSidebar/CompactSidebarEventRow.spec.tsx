import type { ComponentProps } from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { SidebarEvent } from "./types";
import { CompactSidebarEventRow } from "./CompactSidebarEventRow";

const event = {
  id: "execution-1",
  title: "Event received at 02/06/2026, 17:28:05",
  state: "success",
  isOpen: false,
  kind: "execution",
  executionId: "execution-1",
  triggerEventId: "root-1",
  receivedAt: new Date("2026-02-06T15:28:05Z"),
} satisfies SidebarEvent;

const actionableExecutionEvent = {
  ...event,
  id: "execution-actionable",
  state: "running",
} satisfies SidebarEvent;

function renderRow(props: Partial<ComponentProps<typeof CompactSidebarEventRow>> = {}) {
  const { event: eventProp = event, ...rest } = props;

  return render(
    <MemoryRouter initialEntries={["/org-1/apps/app-1"]}>
      <Routes>
        <Route
          path="/:organizationId/apps/:appId"
          element={<CompactSidebarEventRow event={eventProp} onSelectRun={vi.fn()} {...rest} />}
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("CompactSidebarEventRow", () => {
  it("renders a router link when the run id is already known", async () => {
    const user = userEvent.setup();
    const onSelectRun = vi.fn();

    renderRow({ runId: "run-1", onSelectRun });

    const link = screen.getByTestId("compact-sidebar-event-row-select");
    expect(link).toHaveAttribute("href", "/org-1/apps/app-1?run=run-1");

    await user.click(link);

    expect(onSelectRun).toHaveBeenCalledWith("run-1");
  });

  it("selects the run when the row label is clicked through the overlay", async () => {
    const user = userEvent.setup();
    const onSelectRun = vi.fn();

    renderRow({ runId: "run-1", onSelectRun });

    await user.click(screen.getByTestId("compact-sidebar-event-row-select"));

    expect(onSelectRun).toHaveBeenCalledTimes(1);
    expect(onSelectRun).toHaveBeenCalledWith("run-1");
  });

  it("fetches the run id on click when the row is not pre-resolved", async () => {
    const user = userEvent.setup();
    const onSelectRun = vi.fn();
    const fetchRunId = vi.fn(async () => "run-from-api");

    renderRow({ runId: null, fetchRunId, onSelectRun });

    const selectControl = screen.getByTestId("compact-sidebar-event-row-select");
    expect(selectControl.tagName).toBe("BUTTON");

    await user.click(selectControl);

    await waitFor(() => {
      expect(fetchRunId).toHaveBeenCalledWith(event);
      expect(onSelectRun).toHaveBeenCalledWith("run-from-api");
    });
  });

  it("shows a run actions trigger without requiring hover", () => {
    renderRow({
      event: actionableExecutionEvent,
      onCancelExecution: vi.fn(),
    });

    expect(screen.getByRole("button", { name: "Run actions" })).toBeInTheDocument();
  });

  it("cancels an execution from the run actions menu", async () => {
    const user = userEvent.setup();
    const onCancelExecution = vi.fn();

    renderRow({
      event: actionableExecutionEvent,
      onCancelExecution,
    });

    await user.click(screen.getByRole("button", { name: "Run actions" }));
    await user.click(screen.getByRole("menuitem", { name: "Cancel" }));

    expect(onCancelExecution).toHaveBeenCalledWith("execution-1");
  });
});
