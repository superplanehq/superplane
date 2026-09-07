import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { JumpToLatestPill } from "./JumpToLatestPill";

describe("JumpToLatestPill", () => {
  it("renders the default copy and testid", () => {
    render(<JumpToLatestPill onJumpToLatest={vi.fn()} />);

    expect(screen.getByTestId("jump-to-latest")).toBeInTheDocument();
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.viewingOlder)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest })).toBeInTheDocument();
  });

  it("fires onJumpToLatest when the action button is clicked", async () => {
    const user = userEvent.setup();
    const onJumpToLatest = vi.fn();
    render(<JumpToLatestPill onJumpToLatest={onJumpToLatest} testId="custom-pill" />);

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest }));

    expect(onJumpToLatest).toHaveBeenCalledOnce();
    expect(screen.getByTestId("custom-pill")).toBeInTheDocument();
  });

  it("supports overriding the message and action copy", () => {
    render(<JumpToLatestPill onJumpToLatest={vi.fn()} message="Custom message" action="Custom action" />);

    expect(screen.getByText("Custom message")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Custom action" })).toBeInTheDocument();
  });
});
