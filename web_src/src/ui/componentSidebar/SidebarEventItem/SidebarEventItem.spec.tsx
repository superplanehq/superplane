import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { SidebarEvent } from "../types";
import { SidebarEventItem } from "./SidebarEventItem";

const actionableEvent = {
  id: "event-1",
  title: "Execution in progress",
  state: "running",
  isOpen: false,
  kind: "execution",
  executionId: "execution-1",
  receivedAt: new Date("2026-06-10T12:00:00Z"),
} satisfies SidebarEvent;

describe("SidebarEventItem", () => {
  it("shows a run actions trigger without requiring hover", () => {
    render(
      <SidebarEventItem
        event={actionableEvent}
        index={0}
        isOpen={false}
        onToggleOpen={vi.fn()}
        onCancelExecution={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "Run actions" })).toBeInTheDocument();
  });

  it("cancels an execution from the visible run actions menu", async () => {
    const user = userEvent.setup();
    const onCancelExecution = vi.fn();
    const onToggleOpen = vi.fn();

    render(
      <SidebarEventItem
        event={actionableEvent}
        index={0}
        isOpen={false}
        onToggleOpen={onToggleOpen}
        onCancelExecution={onCancelExecution}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run actions" }));
    await user.click(screen.getByRole("menuitem", { name: "Cancel" }));

    expect(onCancelExecution).toHaveBeenCalledWith("execution-1");
    expect(onToggleOpen).not.toHaveBeenCalled();
  });
});
