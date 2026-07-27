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
});
