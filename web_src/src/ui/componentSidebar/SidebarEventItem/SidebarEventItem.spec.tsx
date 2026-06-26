import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { SidebarEvent } from "../types";
import { SidebarEventItem } from "./SidebarEventItem";

const baseEvent: SidebarEvent = {
  id: "event-1",
  title: "Run event",
  state: "success",
  isOpen: false,
  kind: "trigger",
};

describe("SidebarEventItem", () => {
  it("shows a visible actions trigger for actionable events", () => {
    render(<SidebarEventItem event={baseEvent} index={0} isOpen={false} onToggleOpen={vi.fn()} onReEmit={vi.fn()} />);

    const trigger = screen.getByTestId("sidebar-event-actions-trigger");
    expect(trigger).toBeInTheDocument();
    expect(trigger).toHaveTextContent("Actions");
  });

  it("does not render an actions trigger when no actions are available", () => {
    render(
      <SidebarEventItem
        event={{ ...baseEvent, id: "event-2", kind: "execution", state: "success", executionId: undefined }}
        index={0}
        isOpen={false}
        onToggleOpen={vi.fn()}
      />,
    );

    expect(screen.queryByTestId("sidebar-event-actions-trigger")).not.toBeInTheDocument();
  });
});
