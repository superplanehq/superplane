import { render, screen } from "@testing-library/react";
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
    toggleSection: vi.fn(),
    scrollRef: { current: null },
  });
});

describe("LiveLogStreamView", () => {
  it("shows a loading message while the stream is connecting for a finished execution", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: [],
      error: null,
      isStreaming: true,
      toggleSection: vi.fn(),
      scrollRef: { current: null },
    });

    render(<LiveLogStreamView execution={finishedExecution} />);

    expect(screen.getByText("Waiting for logs…")).toBeInTheDocument();
    expect(screen.queryByText("No log lines yet.")).not.toBeInTheDocument();
  });

  it("shows the empty message only after the stream settles with no lines", () => {
    render(<LiveLogStreamView execution={finishedExecution} />);

    expect(screen.getByText("No log lines yet.")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
  });

  it("keeps waiting while an in-flight execution has no lines yet", () => {
    render(<LiveLogStreamView execution={startedExecution} />);

    expect(screen.getByText("Waiting for logs…")).toBeInTheDocument();
    expect(screen.queryByText("No log lines yet.")).not.toBeInTheDocument();
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
