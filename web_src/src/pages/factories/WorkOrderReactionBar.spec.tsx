import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { WorkOrderReactionBar } from "./WorkOrderReactionBar";

function renderBar(props: Partial<React.ComponentProps<typeof WorkOrderReactionBar>> = {}) {
  const onToggleReaction = vi.fn();
  render(
    <TooltipProvider>
      <WorkOrderReactionBar
        reactions={[{ emoji: "👍", count: 2, mine: false, reactorNames: ["Alice", "Bob"] }]}
        canReact
        onToggleReaction={onToggleReaction}
        {...props}
      />
    </TooltipProvider>,
  );
  return { onToggleReaction };
}

describe("WorkOrderReactionBar", () => {
  it("shows only the add control when there are no reactions", () => {
    renderBar({ reactions: [] });

    expect(screen.queryByTestId(/work-order-reaction-chip-/)).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-reaction-add-button")).toBeInTheDocument();
  });

  it("calls onToggleReaction when clicking someone else's chip", () => {
    const { onToggleReaction } = renderBar();

    fireEvent.click(screen.getByTestId("work-order-reaction-chip-thumbs-up"));

    expect(onToggleReaction).toHaveBeenCalledWith("👍");
  });

  it("highlights the viewer's own reaction and still toggles it on click", () => {
    const { onToggleReaction } = renderBar({
      reactions: [{ emoji: "🎉", count: 1, mine: true, reactorNames: ["You"] }],
    });

    const chip = screen.getByTestId("work-order-reaction-chip-party");
    expect(chip).toHaveAttribute("data-mine", "true");
    expect(chip).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(chip);
    expect(onToggleReaction).toHaveBeenCalledWith("🎉");
  });

  it("opens the curated emoji picker from the add control and picks an emoji", () => {
    const { onToggleReaction } = renderBar({ reactions: [] });

    fireEvent.click(screen.getByTestId("work-order-reaction-add-button"));
    expect(screen.getByTestId("work-order-reaction-picker")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("work-order-reaction-picker-option-rocket"));
    expect(onToggleReaction).toHaveBeenCalledWith("🚀");
  });

  it("disables chips and the add control, without hiding existing reactions, when read-only", () => {
    const { onToggleReaction } = renderBar({ canReact: false });

    const chip = screen.getByTestId("work-order-reaction-chip-thumbs-up");
    expect(chip).toBeDisabled();
    fireEvent.click(chip);
    expect(onToggleReaction).not.toHaveBeenCalled();

    expect(screen.getByTestId("work-order-reaction-add-button")).toBeDisabled();
  });
});
