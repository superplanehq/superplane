import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { WorkOrderReactions, type WorkOrderReactionSummary } from "./WorkOrderReactions";

const REACTIONS: WorkOrderReactionSummary[] = [
  { emoji: "👍", count: 3, reactedByMe: true, reactorNames: ["You", "Alice", "Bob"] },
  { emoji: "✅", count: 1, reactedByMe: false, reactorNames: ["Alice"] },
];

describe("WorkOrderReactions", () => {
  it("calls onToggle when clicking a pill you've already reacted with", () => {
    const onToggle = vi.fn();
    render(<WorkOrderReactions reactions={REACTIONS} canReact onToggle={onToggle} onPickNew={vi.fn()} />);

    fireEvent.click(screen.getByTestId("work-order-reaction-pill-👍"));
    expect(onToggle).toHaveBeenCalledWith("👍");
  });

  it("calls onToggle when clicking a pill you haven't reacted with", () => {
    const onToggle = vi.fn();
    render(<WorkOrderReactions reactions={REACTIONS} canReact onToggle={onToggle} onPickNew={vi.fn()} />);

    fireEvent.click(screen.getByTestId("work-order-reaction-pill-✅"));
    expect(onToggle).toHaveBeenCalledWith("✅");
  });

  it("opens the picker and adds a new emoji via a curated option", () => {
    const onPickNew = vi.fn();
    render(<WorkOrderReactions reactions={REACTIONS} canReact onToggle={vi.fn()} onPickNew={onPickNew} />);

    fireEvent.click(screen.getByTestId("work-order-add-reaction-button"));
    expect(screen.getByTestId("work-order-reaction-picker")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("work-order-reaction-picker-curated-🎉"));
    expect(onPickNew).toHaveBeenCalledWith("🎉");
  });

  it("filters the full picker by search keyword", () => {
    render(<WorkOrderReactions reactions={[]} canReact onToggle={vi.fn()} onPickNew={vi.fn()} />);

    fireEvent.click(screen.getByTestId("work-order-add-reaction-button"));
    fireEvent.change(screen.getByTestId("work-order-reaction-picker-search"), { target: { value: "fire" } });

    expect(screen.getByTestId("work-order-reaction-picker-option-🔥")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-reaction-picker-option-👍")).not.toBeInTheDocument();
  });

  it("renders view-only pills with no add button when the viewer can't react", () => {
    render(<WorkOrderReactions reactions={REACTIONS} canReact={false} onToggle={vi.fn()} onPickNew={vi.fn()} />);

    expect(screen.queryByTestId("work-order-add-reaction-button")).not.toBeInTheDocument();
    const pill = screen.getByTestId("work-order-reaction-pill-👍");
    expect(pill.tagName).toBe("DIV");
    fireEvent.click(pill);
  });

  it("renders nothing when read-only with no reactions", () => {
    const { container } = render(
      <WorkOrderReactions reactions={[]} canReact={false} onToggle={vi.fn()} onPickNew={vi.fn()} />,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it("disables pills and the add button while permissions are loading", () => {
    render(
      <WorkOrderReactions
        reactions={REACTIONS}
        canReact={false}
        permissionsLoading
        onToggle={vi.fn()}
        onPickNew={vi.fn()}
      />,
    );

    expect(screen.getByTestId("work-order-add-reaction-button")).toBeDisabled();
    expect(screen.getByTestId("work-order-reaction-pill-👍")).toBeDisabled();
  });

  it("removes the pill once its count drops to zero", () => {
    function Host() {
      const singleReaction: WorkOrderReactionSummary[] = [
        { emoji: "🎉", count: 1, reactedByMe: true, reactorNames: ["You"] },
      ];
      return (
        <WorkOrderReactions
          reactions={singleReaction}
          canReact
          onToggle={() => {
            /* simulated by re-render below */
          }}
          onPickNew={vi.fn()}
        />
      );
    }

    const { rerender } = render(<Host />);
    expect(screen.getByTestId("work-order-reaction-pill-🎉")).toBeInTheDocument();

    rerender(<WorkOrderReactions reactions={[]} canReact onToggle={vi.fn()} onPickNew={vi.fn()} />);
    expect(screen.queryByTestId("work-order-reaction-pill-🎉")).not.toBeInTheDocument();
  });
});
