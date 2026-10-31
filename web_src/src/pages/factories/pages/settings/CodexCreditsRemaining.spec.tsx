import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { CodexCreditsRemaining } from "./CodexCreditsRemaining";

describe("Codex monthly credits", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("hides a saved monthly amount when the reset time passed before page load", () => {
    const resetsAt = new Date(Date.now() - 1000).toISOString();
    render(<CodexCreditsRemaining individualLimit={{ limit: "5000", used: "288.32", resetsAt }} />);

    expect(screen.getByText("Monthly credits")).toBeInTheDocument();
    expect(screen.getByText("Update needed")).toBeInTheDocument();
    expect(screen.queryByText("4,712 credits")).not.toBeInTheDocument();
    expect(screen.getByText(`Reset time passed: ${new Date(resetsAt).toLocaleString()}`)).toBeInTheDocument();
  });

  it.each([
    { name: "one second", delay: 1000 },
    { name: "31 days", delay: 31 * 86400000 },
  ])("hides a saved monthly amount after $name while the page stays open", ({ delay }) => {
    const resetsAt = new Date(Date.now() + delay).toISOString();
    render(<CodexCreditsRemaining individualLimit={{ limit: "5000", used: "288.32", resetsAt }} />);

    act(() => vi.advanceTimersByTime(delay - 1));
    expect(screen.getByText("4,712 credits")).toBeInTheDocument();
    expect(screen.getByText(`Resets: ${new Date(resetsAt).toLocaleString()}`)).toBeInTheDocument();

    act(() => vi.advanceTimersByTime(1));
    expect(screen.getByText("Update needed")).toBeInTheDocument();
    expect(screen.queryByText("4,712 credits")).not.toBeInTheDocument();
    expect(screen.getByText(`Reset time passed: ${new Date(resetsAt).toLocaleString()}`)).toBeInTheDocument();
  });
});
