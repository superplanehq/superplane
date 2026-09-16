import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AnalysisLiveWork } from "./IntentAnalysisLiveWork";
import { useAgentActivityStream } from "./useAgentActivityStream";

vi.mock("./useAgentActivityStream", () => ({ useAgentActivityStream: vi.fn() }));

describe("AnalysisLiveWork", () => {
  beforeEach(() => {
    vi.mocked(useAgentActivityStream).mockReturnValue({ activities: [], isConnected: true, hasConnectedOnce: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not reopen a completed prior turn when a new analysis starts", () => {
    render(
      <AnalysisLiveWork
        machineStatus="running"
        activities={[
          {
            id: "activity-prior",
            provider: "claude",
            status: "passed",
            sequence: 8,
            truncated: false,
            items: [
              {
                type: "tool",
                id: "prior-tool",
                kind: "bash",
                name: "Bash",
                input: "rg retry",
                output: "ok",
                outputStreams: [],
                status: "passed",
                truncated: false,
              },
            ],
          },
        ]}
      />,
    );

    expect(screen.queryByRole("button", { name: "Ran command" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Ran / })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent("Starting analysis…");
  });

  it("shows a persisted running activity when the live stream missed its start", () => {
    render(
      <AnalysisLiveWork
        machineStatus="running"
        activities={[
          {
            id: "activity-running",
            provider: "claude",
            status: "running",
            sequence: 3,
            truncated: false,
            items: [
              {
                type: "tool",
                id: "running-tool",
                kind: "bash",
                name: "Bash",
                input: "rg -n survey pkg",
                output: "",
                outputStreams: [],
                status: "running",
                truncated: false,
              },
            ],
          },
        ]}
      />,
    );

    const activity = screen.getByRole("button", { name: "Searching code" });
    expect(activity).toBeInTheDocument();
    expect(activity.querySelector(".sp-thinking-state-current")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-thinking")).not.toBeInTheDocument();
  });

  it("shows an honest starting state before the first provider event", () => {
    render(<AnalysisLiveWork machineStatus="running" />);
    const status = screen.getByTestId("split-run-intent-thinking");
    const label = within(status).getByText("Starting analysis…");

    expect(status).toHaveClass("px-3");
    expect(status).not.toHaveClass("sp-ai-thinking");
    expect(label).toHaveClass("sp-ai-thinking");
  });

  it("shows stale elapsed time in seconds", () => {
    vi.useFakeTimers();
    render(<AnalysisLiveWork machineStatus="running" />);

    act(() => vi.advanceTimersByTime(31_000));

    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent("Still working · 31s");
  });

  it("shows exact reasoning and an active command", () => {
    vi.mocked(useAgentActivityStream).mockReturnValue({
      isConnected: true,
      hasConnectedOnce: true,
      activities: [
        {
          id: "activity-1",
          provider: "codex",
          status: "running",
          sequence: 4,
          truncated: false,
          items: [
            {
              type: "content",
              id: "thought-1",
              kind: "reasoning",
              text: "I need to inspect the retry path.",
              status: "running",
              truncated: false,
            },
            {
              type: "tool",
              id: "tool-1",
              kind: "bash",
              name: "Bash",
              input: "rg -n retry pkg",
              output: "",
              outputStreams: [],
              status: "running",
              truncated: false,
            },
          ],
        },
      ],
    });

    render(<AnalysisLiveWork machineStatus="running" />);
    expect(screen.getByText("I need to inspect the retry path.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Searching code" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("split-run-intent-thinking")).not.toBeInTheDocument();
  });

  it("keeps an animated writing status until assistant text arrives", () => {
    vi.mocked(useAgentActivityStream).mockReturnValue({
      isConnected: true,
      hasConnectedOnce: true,
      activities: [
        {
          id: "activity-1",
          provider: "claude",
          status: "running",
          sequence: 1,
          truncated: false,
          items: [
            {
              type: "content",
              id: "assistant-1",
              kind: "assistant",
              text: "",
              status: "running",
              truncated: false,
            },
          ],
        },
      ],
    });

    render(<AnalysisLiveWork machineStatus="running" />);

    const status = screen.getByTestId("split-run-intent-thinking");
    expect(status).toHaveAccessibleName("Writing response…");
    expect(status.querySelector(".sp-thinking-state-current")).toBeInTheDocument();
  });

  it("keeps successful and failed live commands collapsed", async () => {
    const user = userEvent.setup();
    vi.mocked(useAgentActivityStream).mockReturnValue({
      isConnected: false,
      hasConnectedOnce: true,
      activities: [
        {
          id: "activity-1",
          provider: "claude",
          status: "failed",
          sequence: 5,
          truncated: false,
          items: [
            {
              type: "tool",
              id: "passed",
              kind: "bash",
              name: "Bash",
              input: "true",
              output: "",
              outputStreams: [],
              status: "passed",
              truncated: false,
            },
            {
              type: "tool",
              id: "failed",
              kind: "bash",
              name: "Bash",
              input: "false",
              output: "failed",
              outputStreams: [{ stream: "stderr", text: "failed" }],
              status: "failed",
              exitCode: 1,
              truncated: false,
            },
          ],
        },
      ],
    });

    render(<AnalysisLiveWork machineStatus="running" />);
    const summary = screen.getByRole("button", { name: "Used terminal 2 times" });
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("failed")).not.toBeInTheDocument();
    await user.click(summary);
    expect(screen.getByTestId("agent-tool-passed")).toHaveTextContent("true");
    expect(screen.getByTestId("agent-tool-failed")).toHaveTextContent("false");
    expect(screen.getByTestId("agent-tool-failed")).toHaveClass("text-destructive");
    expect(screen.queryByText("failed")).not.toBeInTheDocument();
  });

  it("hides after the machine waits", () => {
    render(<AnalysisLiveWork machineStatus="waiting" />);
    expect(screen.queryByTestId("split-run-intent-live-work")).not.toBeInTheDocument();
  });

  it("does not describe an initial connection attempt as a reconnect", () => {
    vi.mocked(useAgentActivityStream).mockReturnValue({
      activities: [],
      error: "The stream is not available yet.",
      isConnected: false,
      hasConnectedOnce: false,
    });

    render(<AnalysisLiveWork machineStatus="starting" />);

    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent("Starting analysis…");
    expect(screen.queryByText("Live activity disconnected. Reconnecting…")).not.toBeInTheDocument();
  });

  it("shows reconnecting only after a live connection is lost", () => {
    vi.mocked(useAgentActivityStream).mockReturnValue({
      activities: [],
      error: "The stream closed.",
      isConnected: false,
      hasConnectedOnce: true,
    });

    render(<AnalysisLiveWork machineStatus="running" />);

    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent(
      "Live activity disconnected. Reconnecting…",
    );
  });

  it("shows one animated status for active work and resumes the tail after it completes", () => {
    const activity = {
      id: "activity-1",
      provider: "codex",
      status: "running" as const,
      sequence: 1,
      truncated: false,
      items: [
        {
          type: "content" as const,
          id: "thought-1",
          kind: "reasoning" as const,
          text: "Inspecting the request.",
          status: "running" as const,
          truncated: false,
        },
      ],
    };
    vi.mocked(useAgentActivityStream).mockReturnValue({
      activities: [activity],
      isConnected: true,
      hasConnectedOnce: true,
    });

    const { rerender } = render(<AnalysisLiveWork machineStatus="running" />);
    expect(screen.getByRole("status", { name: "Thinking" })).toHaveClass("sp-ai-thinking");
    expect(screen.queryByTestId("split-run-intent-thinking")).not.toBeInTheDocument();

    vi.mocked(useAgentActivityStream).mockReturnValue({
      activities: [
        {
          ...activity,
          sequence: 2,
          items: [
            { ...activity.items[0], status: "passed" as const },
            {
              type: "tool",
              id: "search-1",
              kind: "bash",
              name: "Bash",
              input: "rg -n activity web_src",
              output: "",
              outputStreams: [],
              status: "running",
              truncated: false,
            },
          ],
        },
      ],
      isConnected: true,
      hasConnectedOnce: true,
    });
    rerender(<AnalysisLiveWork machineStatus="running" />);

    const activityStatus = screen.getByRole("button", { name: "Searching code" });
    expect(activityStatus.querySelector(".sp-thinking-state-current")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-thinking")).not.toBeInTheDocument();

    vi.mocked(useAgentActivityStream).mockReturnValue({
      activities: [
        {
          ...activity,
          sequence: 3,
          items: [
            { ...activity.items[0], status: "passed" as const },
            {
              type: "tool",
              id: "search-1",
              kind: "bash",
              name: "Bash",
              input: "rg -n activity web_src",
              output: "",
              outputStreams: [],
              status: "passed",
              truncated: false,
            },
          ],
        },
      ],
      isConnected: true,
      hasConnectedOnce: true,
    });
    rerender(<AnalysisLiveWork machineStatus="running" />);

    expect(screen.getByTestId("split-run-intent-thinking")).toHaveAccessibleName("Planning next step…");
  });
});
