import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { WorkOrderReactionBar, formatReactorNames, type WorkOrderReactionGroup } from "./WorkOrderReactionBar";

const ME = "me";
const ALEX = { id: "alex", name: "Alex Chen" };
const PRIYA = { id: "priya", name: "Priya Patel" };
const JORDAN = { id: "jordan", name: "Jordan Lee" };

const REACTIONS: WorkOrderReactionGroup[] = [
  { emoji: "👍", users: [{ id: ME, name: "Me" }, ALEX] },
  { emoji: "🎉", users: [PRIYA] },
];

describe("WorkOrderReactionBar", () => {
  it("shows the empty-state 'Add reaction' trigger with no pills", () => {
    render(<WorkOrderReactionBar reactions={[]} currentUserId={ME} canReact onToggleReaction={vi.fn()} />);

    expect(screen.getByTestId("work-order-add-reaction-trigger")).toHaveTextContent("Add reaction");
    expect(screen.queryByTestId(/work-order-reaction-pill-/)).not.toBeInTheDocument();
  });

  it("renders a pill per emoji with its count and highlights the viewer's own reaction", () => {
    render(<WorkOrderReactionBar reactions={REACTIONS} currentUserId={ME} canReact onToggleReaction={vi.fn()} />);

    const mine = screen.getByTestId("work-order-reaction-pill-👍");
    expect(mine).toHaveTextContent("2");
    expect(mine).toHaveAttribute("aria-pressed", "true");

    const notMine = screen.getByTestId("work-order-reaction-pill-🎉");
    expect(notMine).toHaveTextContent("1");
    expect(notMine).toHaveAttribute("aria-pressed", "false");
  });

  it("toggles an existing pill by emoji when clicked", () => {
    const onToggleReaction = vi.fn();
    render(
      <WorkOrderReactionBar reactions={REACTIONS} currentUserId={ME} canReact onToggleReaction={onToggleReaction} />,
    );

    fireEvent.click(screen.getByTestId("work-order-reaction-pill-👍"));
    expect(onToggleReaction).toHaveBeenCalledWith("👍");
  });

  it("opens the picker and adds a new reaction, closing the popover afterward", () => {
    const onToggleReaction = vi.fn();
    render(<WorkOrderReactionBar reactions={[]} currentUserId={ME} canReact onToggleReaction={onToggleReaction} />);

    fireEvent.click(screen.getByTestId("work-order-add-reaction-trigger"));
    fireEvent.click(screen.getByTestId("work-order-reaction-option-🚀"));

    expect(onToggleReaction).toHaveBeenCalledWith("🚀");
    expect(screen.queryByTestId("work-order-reaction-option-🚀")).not.toBeInTheDocument();
  });

  it("disables the add-reaction trigger and existing pills without permission", () => {
    const onToggleReaction = vi.fn();
    render(
      <WorkOrderReactionBar
        reactions={REACTIONS}
        currentUserId={ME}
        canReact={false}
        onToggleReaction={onToggleReaction}
      />,
    );

    expect(screen.getByTestId("work-order-add-reaction-trigger")).toBeDisabled();
    expect(screen.getByTestId("work-order-reaction-pill-👍")).toBeDisabled();

    fireEvent.click(screen.getByTestId("work-order-reaction-pill-👍"));
    expect(onToggleReaction).not.toHaveBeenCalled();
  });

  it("shows a subdued look for a pending (in-flight) reaction", () => {
    render(
      <WorkOrderReactionBar
        reactions={REACTIONS}
        currentUserId={ME}
        canReact
        pendingEmoji="👍"
        onToggleReaction={vi.fn()}
      />,
    );

    expect(screen.getByTestId("work-order-reaction-pill-👍").className).toContain("opacity-60");
  });
});

describe("formatReactorNames", () => {
  it("shows a single reactor's name", () => {
    expect(formatReactorNames([ALEX], "someone-else")).toBe("Alex Chen");
  });

  it("joins two reactors with 'and'", () => {
    expect(formatReactorNames([ALEX, PRIYA], "someone-else")).toBe("Alex Chen and Priya Patel");
  });

  it("truncates 3+ reactors to two names plus an 'others' count", () => {
    expect(formatReactorNames([ALEX, PRIYA, JORDAN], "someone-else")).toBe("Alex Chen, Priya Patel, and 1 other");
  });

  it("lists the viewer first as 'You' when they're among the reactors", () => {
    expect(formatReactorNames([ALEX, { id: ME, name: "Me" }], ME)).toBe("You and Alex Chen");
  });
});
