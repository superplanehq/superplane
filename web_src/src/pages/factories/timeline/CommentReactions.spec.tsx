import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CommentReactions } from "./CommentReactions";
import type { WorkOrderTimelineCommentReaction } from "../lib/workOrderTimelineEvents";

const REACTIONS: WorkOrderTimelineCommentReaction[] = [
  { emoji: "+1", count: 2, reactedByMe: true },
  { emoji: "eyes", count: 1, reactedByMe: false },
];

describe("CommentReactions", () => {
  it("renders a pill per reaction with its count, highlighting the caller's own reaction", () => {
    render(<CommentReactions reactions={REACTIONS} canReact onAddReaction={vi.fn()} onRemoveReaction={vi.fn()} />);

    const mine = screen.getByTestId("work-order-comment-reaction-+1");
    expect(mine).toHaveTextContent("2");
    expect(mine).toHaveAttribute("aria-pressed", "true");

    const others = screen.getByTestId("work-order-comment-reaction-eyes");
    expect(others).toHaveTextContent("1");
    expect(others).toHaveAttribute("aria-pressed", "false");
  });

  it("removes the caller's own reaction when its pill is clicked", async () => {
    const user = userEvent.setup();
    const onRemoveReaction = vi.fn();

    render(
      <CommentReactions reactions={REACTIONS} canReact onAddReaction={vi.fn()} onRemoveReaction={onRemoveReaction} />,
    );

    await user.click(screen.getByTestId("work-order-comment-reaction-+1"));
    expect(onRemoveReaction).toHaveBeenCalledWith("+1");
  });

  it("adds a reaction that isn't the caller's own when its pill is clicked", async () => {
    const user = userEvent.setup();
    const onAddReaction = vi.fn();

    render(
      <CommentReactions reactions={REACTIONS} canReact onAddReaction={onAddReaction} onRemoveReaction={vi.fn()} />,
    );

    await user.click(screen.getByTestId("work-order-comment-reaction-eyes"));
    expect(onAddReaction).toHaveBeenCalledWith("eyes");
  });

  it("only offers the fixed emoji set in the picker", async () => {
    const user = userEvent.setup();

    render(<CommentReactions reactions={[]} canReact onAddReaction={vi.fn()} onRemoveReaction={vi.fn()} />);

    await user.click(screen.getByTestId("work-order-comment-add-reaction"));

    const expected = ["+1", "-1", "laugh", "hooray", "confused", "heart", "rocket", "eyes"];
    for (const emoji of expected) {
      expect(screen.getByTestId(`work-order-comment-reaction-picker-${emoji}`)).toBeInTheDocument();
    }
    expect(screen.queryByTestId("work-order-comment-reaction-picker-not-a-real-emoji")).not.toBeInTheDocument();
  });

  it("adds an unreacted emoji picked from the picker", async () => {
    const user = userEvent.setup();
    const onAddReaction = vi.fn();

    render(<CommentReactions reactions={[]} canReact onAddReaction={onAddReaction} onRemoveReaction={vi.fn()} />);

    await user.click(screen.getByTestId("work-order-comment-add-reaction"));
    await user.click(screen.getByTestId("work-order-comment-reaction-picker-heart"));

    expect(onAddReaction).toHaveBeenCalledWith("heart");
  });

  it("disables interaction and hides the empty picker when the viewer can't react", () => {
    render(<CommentReactions reactions={[]} canReact={false} onAddReaction={vi.fn()} onRemoveReaction={vi.fn()} />);

    expect(screen.queryByTestId("work-order-comment-reactions")).not.toBeInTheDocument();
  });

  it("still renders existing reactions read-only when the viewer can't react", () => {
    render(
      <CommentReactions reactions={REACTIONS} canReact={false} onAddReaction={vi.fn()} onRemoveReaction={vi.fn()} />,
    );

    expect(screen.getByTestId("work-order-comment-reaction-+1")).toBeDisabled();
    expect(screen.getByTestId("work-order-comment-add-reaction")).toBeDisabled();
  });
});
