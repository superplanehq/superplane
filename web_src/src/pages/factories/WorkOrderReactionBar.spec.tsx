import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WorkOrderReactionBar } from "./WorkOrderReactionBar";

describe("WorkOrderReactionBar", () => {
  it("shows just the Add reaction affordance when there are no reactions yet", () => {
    render(<WorkOrderReactionBar reactions={[]} myReaction={null} onToggle={vi.fn()} />);

    expect(screen.getByTestId("work-order-reaction-add")).toHaveTextContent("Add reaction");
    expect(screen.queryByTestId(/work-order-reaction-pill-/)).toBeNull();
  });

  it("renders a pill per emoji group with its reactor count, and highlights the viewer's own reaction", () => {
    render(
      <WorkOrderReactionBar
        reactions={[
          { emoji: "👍", reactorNames: ["Alex Reviewer", "Storybook User"] },
          { emoji: "🎉", reactorNames: ["Alex Reviewer"] },
        ]}
        myReaction="👍"
        onToggle={vi.fn()}
      />,
    );

    const minePill = screen.getByTestId("work-order-reaction-pill-👍");
    expect(minePill).toHaveTextContent("2");
    expect(minePill).toHaveAttribute("aria-pressed", "true");

    const otherPill = screen.getByTestId("work-order-reaction-pill-🎉");
    expect(otherPill).toHaveTextContent("1");
    expect(otherPill).toHaveAttribute("aria-pressed", "false");
  });

  it("toggles a reaction when its pill is clicked", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(
      <WorkOrderReactionBar
        reactions={[{ emoji: "👍", reactorNames: ["Alex Reviewer"] }]}
        myReaction={null}
        onToggle={onToggle}
      />,
    );

    await user.click(screen.getByTestId("work-order-reaction-pill-👍"));

    expect(onToggle).toHaveBeenCalledWith("👍");
  });

  it("opens the picker from Add reaction, and picking an emoji toggles it and closes the picker", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<WorkOrderReactionBar reactions={[]} myReaction={null} onToggle={onToggle} />);

    expect(screen.queryByTestId("work-order-reaction-picker")).toBeNull();

    await user.click(screen.getByTestId("work-order-reaction-add"));
    expect(screen.getByTestId("work-order-reaction-picker")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-order-reaction-option-🚀"));

    expect(onToggle).toHaveBeenCalledWith("🚀");
    expect(screen.queryByTestId("work-order-reaction-picker")).toBeNull();
  });
});
