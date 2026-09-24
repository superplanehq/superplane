import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { AgentActivityItem } from "@/lib/agentActivity";
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

function activitySection(items: AgentActivityItem[]) {
  return {
    index: 5,
    text: "Implementation",
    kind: "prompt",
    preview: "You are implementing a fix",
    lines: [] as string[],
    events: [],
    activities: [
      {
        id: "act-1",
        provider: "claude",
        status: "running" as const,
        sequence: 5,
        items,
        truncated: false,
      },
    ],
    status: "running" as const,
    duration_ms: null,
    started_at: 1,
    collapsed: false,
  };
}

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue({
    sections: [],
    orphanLines: [],
    error: null,
    isLoading: false,
    isStreaming: false,
    telemetry: { num_turns: 0, usage: {}, tool_counts: {}, turns: [] },
    usageSeries: [],
    toggleSection: vi.fn(),
    retry: vi.fn(),
    scrollRef: { current: null },
  });
});

describe("LiveLogStreamView", () => {
  it("shows loading while it fetches logs for a finished execution", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: [],
      error: null,
      isLoading: true,
      isStreaming: true,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    render(<LiveLogStreamView execution={finishedExecution} />);

    expect(screen.getByText("Loading logs")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs")).not.toBeInTheDocument();
    expect(screen.queryByText("No logs available")).not.toBeInTheDocument();
  });

  it("shows the empty state when a finished execution has no logs", () => {
    render(<LiveLogStreamView execution={finishedExecution} />);

    expect(screen.getByText("No logs available")).toBeInTheDocument();
    expect(screen.getByText("Run the task again to produce new log output.")).toBeInTheDocument();
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
      isLoading: false,
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
      isLoading: false,
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
      isLoading: false,
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

  it("renders version 2 agent activity as text and a tool row", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        {
          index: 5,
          text: "Implementation",
          kind: "prompt",
          preview: "You are implementing a fix",
          lines: [],
          events: [],
          activities: [
            {
              id: "act-1",
              provider: "claude",
              status: "running",
              sequence: 5,
              items: [
                {
                  type: "content",
                  id: "c1",
                  kind: "assistant",
                  text: "I will verify the seams before updating the plan.",
                  status: "passed",
                  truncated: false,
                },
                {
                  type: "tool",
                  id: "t1",
                  kind: "bash",
                  name: "bash",
                  input: '{"command":"git status"}',
                  output: "",
                  outputStreams: [],
                  status: "passed",
                  durationMs: 549,
                  truncated: false,
                },
              ],
              truncated: false,
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
      isLoading: false,
      isStreaming: false,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    const { container } = render(<LiveLogStreamView execution={startedExecution} />);
    const text = container.textContent ?? "";

    expect(screen.getByText("I will verify the seams before updating the plan.")).toBeInTheDocument();
    expect(screen.getByText("git status")).toBeInTheDocument();
    expect(text).toContain("Passed");
    expect(text.toLowerCase()).not.toContain("bash bash");
    expect(text).not.toContain("schema_version");
    expect(text).not.toContain("event_id");
    expect(text).not.toContain("activity_id");
  });

  it("shows a bash command after empty start input is filled", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        activitySection([
          {
            type: "tool",
            id: "t1",
            kind: "bash",
            name: "Bash",
            input: '{"command":"ls pkg"}',
            output: "",
            outputStreams: [],
            status: "running",
            truncated: false,
          },
        ]),
      ],
      orphanLines: [],
      error: null,
      isLoading: false,
      isStreaming: true,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    const { container } = render(<LiveLogStreamView execution={startedExecution} />);

    expect(screen.getByText("ls pkg")).toBeInTheDocument();
    expect(container.textContent?.toLowerCase()).not.toContain("bash bash");
  });

  it("shows the tool name once while a bash command is still empty", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        activitySection([
          {
            type: "tool",
            id: "t1",
            kind: "bash",
            name: "Bash",
            input: "",
            output: "",
            outputStreams: [],
            status: "running",
            truncated: false,
          },
        ]),
      ],
      orphanLines: [],
      error: null,
      isLoading: false,
      isStreaming: true,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    const { container } = render(<LiveLogStreamView execution={startedExecution} />);
    const text = container.textContent ?? "";

    expect(screen.getByText("Bash")).toBeInTheDocument();
    expect(text).toContain("Running");
    expect(text.toLowerCase()).not.toContain("bash bash");
  });

  it("shows file and search detail on activity rows", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        activitySection([
          {
            type: "tool",
            id: "read-1",
            kind: "read",
            name: "Read",
            input: '{"path":"pkg/foo.go"}',
            output: "",
            outputStreams: [],
            status: "passed",
            durationMs: 20,
            truncated: false,
          },
          {
            type: "tool",
            id: "grep-1",
            kind: "grep",
            name: "Grep",
            input: "rootTriggerRenderer",
            output: "",
            outputStreams: [],
            status: "passed",
            durationMs: 30,
            truncated: false,
          },
        ]),
      ],
      orphanLines: [],
      error: null,
      isLoading: false,
      isStreaming: false,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    render(<LiveLogStreamView execution={startedExecution} />);

    expect(screen.getByText("pkg/foo.go")).toBeInTheDocument();
    expect(screen.getByText("rootTriggerRenderer")).toBeInTheDocument();
  });

  it("shows at most three output lines and a more hint", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        activitySection([
          {
            type: "tool",
            id: "t1",
            kind: "bash",
            name: "Bash",
            input: '{"command":"git status"}',
            output: "one\ntwo\nthree\nfour",
            outputStreams: [],
            status: "passed",
            durationMs: 549,
            truncated: false,
          },
        ]),
      ],
      orphanLines: [],
      error: null,
      isLoading: false,
      isStreaming: false,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    render(<LiveLogStreamView execution={startedExecution} />);

    expect(screen.getByText("one")).toBeInTheDocument();
    expect(screen.getByText("two")).toBeInTheDocument();
    expect(screen.getByText("three")).toBeInTheDocument();
    expect(screen.queryByText("four")).not.toBeInTheDocument();
    expect(screen.getByText("More output is available.")).toBeInTheDocument();
  });

  it("omits output lines when a tool has no output", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        activitySection([
          {
            type: "tool",
            id: "t1",
            kind: "bash",
            name: "Bash",
            input: '{"command":"git status"}',
            output: "",
            outputStreams: [],
            status: "passed",
            durationMs: 549,
            truncated: false,
          },
        ]),
      ],
      orphanLines: [],
      error: null,
      isLoading: false,
      isStreaming: false,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    const { container } = render(<LiveLogStreamView execution={startedExecution} />);
    const text = container.textContent ?? "";

    expect(screen.getByText("git status")).toBeInTheDocument();
    expect(text).toContain("Passed");
    expect(text).not.toContain("More output is available.");
  });

  it("hides a plain line that is a serialized activity record", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        {
          index: 5,
          text: "Implementation",
          kind: "prompt",
          preview: "You are implementing a fix",
          lines: [
            '{"type":"content_start","id":"msg-1","channel":"assistant","schema_version":2,"event_id":"e1:2","activity_id":"e1"}',
          ],
          events: [],
          status: "running",
          duration_ms: null,
          started_at: 1,
          collapsed: false,
        },
      ],
      orphanLines: [],
      error: null,
      isLoading: false,
      isStreaming: false,
      toggleSection: vi.fn(),
      retry: vi.fn(),
      scrollRef: { current: null },
    });

    const { container } = render(<LiveLogStreamView execution={startedExecution} />);
    const text = container.textContent ?? "";

    expect(text).not.toContain("schema_version");
    expect(text).not.toContain("event_id");
    expect(text).not.toContain("activity_id");
    expect(text).not.toContain("content_start");
  });
});
