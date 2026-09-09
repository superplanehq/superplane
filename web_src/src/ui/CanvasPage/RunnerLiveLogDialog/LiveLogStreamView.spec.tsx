import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ExecutionInfo } from "@/pages/app/mappers/types";
import { LiveLogStreamView } from "./LiveLogStreamView";

const useLiveLogStreamMock = vi.fn();

vi.mock("./useLiveLogStream", () => ({
  terminalCommandStatusForExecution: vi.fn(() => null),
  terminalTimeMsForExecution: vi.fn(() => null),
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

const finishedExecution = {
  id: "execution-1",
  state: "STATE_FINISHED",
} as ExecutionInfo;

const startedExecution = {
  id: "execution-1",
  state: "STATE_STARTED",
} as ExecutionInfo;

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue({
    sections: [],
    orphanLines: [],
    error: null,
    isStreaming: false,
    telemetry: { num_turns: 0, usage: {}, tool_counts: {}, turns: [] },
    usageSeries: [],
    toggleSection: vi.fn(),
    retry: vi.fn(),
    scrollRef: { current: null },
  });
});

describe("LiveLogStreamView", () => {
  it("does not wait for more logs after an execution finishes", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: [],
      error: null,
      isStreaming: true,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    render(<LiveLogStreamView execution={finishedExecution} />);

    expect(screen.getByText("No logs available")).toBeInTheDocument();
    expect(screen.getByText("This execution did not produce log output.")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs")).not.toBeInTheDocument();
  });

  it("shows the empty state when a finished execution has no logs", () => {
    render(<LiveLogStreamView execution={finishedExecution} />);

    expect(screen.getByText("No logs available")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs")).not.toBeInTheDocument();
  });

  it("keeps waiting while an in-flight execution has no lines yet", () => {
    render(<LiveLogStreamView execution={startedExecution} />);

    expect(screen.getByText("Waiting for logs")).toBeInTheDocument();
    expect(screen.getByText("Logs will appear when the runner sends output.")).toBeInTheDocument();
    expect(screen.queryByText("No logs available")).not.toBeInTheDocument();
  });

  it("shows the error and lets the user retry after an execution finishes", () => {
    const retry = vi.fn();
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: [],
      error: "Failed to fetch",
      isStreaming: false,
      toggleSection: vi.fn(),
      retry,
      scrollRef: { current: null },
    });

    render(<LiveLogStreamView execution={finishedExecution} />);

    expect(screen.getByRole("alert")).toHaveTextContent("Could not load logs");
    expect(screen.getByText("Failed to fetch")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledOnce();
  });

  it("shows a retrying error while the execution is in flight", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: [],
      error: "The log stream is not available yet",
      isStreaming: false,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    render(<LiveLogStreamView execution={startedExecution} />);

    expect(screen.getByRole("alert")).toHaveTextContent("Logs are temporarily unavailable");
    expect(screen.getByText("SuperPlane will try again automatically.")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs")).not.toBeInTheDocument();
  });

  it("passes canvas session ids into the live log hook", () => {
    render(
      <LiveLogStreamView execution={startedExecution} session={{ organizationId: "org-1", canvasId: "canvas-1" }} />,
    );

    expect(useLiveLogStreamMock).toHaveBeenCalledWith("execution-1", true, null, null, {
      organizationId: "org-1",
      canvasId: "canvas-1",
    });
  });

  it("badges kind and lists nested prompt tools", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        {
          index: 5,
          text: "Implementation",
          kind: "prompt",
          preview: "You are implementing a fix",
          lines: [],
          events: [
            { kind: "note", text: "Gathering context." },
            {
              kind: "tools",
              id: "5-tools-0",
              tools: [
                {
                  id: "5-tool-0",
                  kind: "read",
                  text: "pkg/foo.go",
                  lines: ["package workers"],
                  status: "passed",
                  duration_ms: 80,
                },
              ],
            },
          ],
          status: "running",
          duration_ms: null,
          started_at: 1,
          collapsed: false,
        },
      ],
      orphanLines: [],
      error: null,
      isStreaming: false,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    render(<LiveLogStreamView execution={startedExecution} />);

    expect(screen.getByText("prompt")).toBeInTheDocument();
    expect(screen.getByText("You are implementing a fix")).toBeInTheDocument();
    expect(screen.getByText("Gathering context.")).toBeInTheDocument();
    expect(screen.getByText("read")).toBeInTheDocument();
    expect(screen.getByText("pkg/foo.go")).toBeInTheDocument();
    expect(screen.getByText("package workers")).toBeInTheDocument();
  });
});
