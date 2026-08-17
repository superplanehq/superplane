import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesWorkOrderReaction } from "@/api-client";
import { WorkOrderReactionBar } from "./WorkOrderReactionBar";

const reactions: FactoriesWorkOrderReaction[] = [
  { content: "+1", count: 2, reactedByMe: true },
  { content: "heart", count: 1, reactedByMe: false },
];

describe("WorkOrderReactionBar", () => {
  it("renders a pill per reaction with its emoji and count, highlighting the caller's own reaction", () => {
    render(<WorkOrderReactionBar reactions={reactions} canReact onToggleReaction={vi.fn()} />);

    const thumbsUp = screen.getByTestId("work-order-reaction-+1");
    expect(thumbsUp).toHaveTextContent("👍");
    expect(thumbsUp).toHaveTextContent("2");
    expect(thumbsUp).toHaveAttribute("aria-pressed", "true");

    const heart = screen.getByTestId("work-order-reaction-heart");
    expect(heart).toHaveTextContent("❤️");
    expect(heart).toHaveTextContent("1");
    expect(heart).toHaveAttribute("aria-pressed", "false");
  });

  it("clicking a reacted-to pill calls onToggleReaction with reactedByMe=true (removes it)", () => {
    const onToggleReaction = vi.fn();
    render(<WorkOrderReactionBar reactions={reactions} canReact onToggleReaction={onToggleReaction} />);

    fireEvent.click(screen.getByTestId("work-order-reaction-+1"));

    expect(onToggleReaction).toHaveBeenCalledWith("+1", true);
  });

  it("clicking an emoji from the add-reaction picker calls onToggleReaction with reactedByMe=false (adds it)", () => {
    const onToggleReaction = vi.fn();
    render(<WorkOrderReactionBar reactions={reactions} canReact onToggleReaction={onToggleReaction} />);

    fireEvent.click(screen.getByTestId("work-order-add-reaction-button"));
    fireEvent.click(screen.getByTestId("work-order-add-reaction-option-rocket"));

    expect(onToggleReaction).toHaveBeenCalledWith("rocket", false);
  });

  it("omits already-active reactions from the add-reaction picker", () => {
    render(<WorkOrderReactionBar reactions={reactions} canReact onToggleReaction={vi.fn()} />);

    fireEvent.click(screen.getByTestId("work-order-add-reaction-button"));

    expect(screen.queryByTestId("work-order-add-reaction-option-+1")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-add-reaction-option-heart")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-add-reaction-option-rocket")).toBeInTheDocument();
  });

  it("disables interaction and shows a permission tooltip when the caller cannot react", () => {
    const onToggleReaction = vi.fn();
    render(<WorkOrderReactionBar reactions={reactions} canReact={false} onToggleReaction={onToggleReaction} />);

    const thumbsUp = screen.getByTestId("work-order-reaction-+1");
    expect(thumbsUp).toBeDisabled();

    fireEvent.click(thumbsUp);
    expect(onToggleReaction).not.toHaveBeenCalled();
  });

  it("renders no reaction pills when there are none, but keeps the add-reaction picker", () => {
    render(<WorkOrderReactionBar reactions={[]} canReact onToggleReaction={vi.fn()} />);

    expect(screen.queryAllByRole("button", { pressed: true })).toHaveLength(0);
    expect(screen.queryAllByRole("button", { pressed: false })).toHaveLength(0);
    expect(screen.getByTestId("work-order-add-reaction-button")).toBeInTheDocument();
  });
});
