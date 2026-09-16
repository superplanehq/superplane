import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AgentActivityView } from "./AgentActivityView";
import type { AgentActivity, AgentToolItem } from "./agentActivity";

describe("AgentActivityView", () => {
  it("shows active reasoning as a non-interactive thinking state", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          type: "content",
          id: "reasoning-1",
          kind: "reasoning",
          text: "Inspecting the retry path.",
          status: "running",
          truncated: false,
        })}
      />,
    );

    expect(screen.getByRole("status", { name: "Thinking" })).toHaveClass("sp-ai-thinking");
    expect(screen.queryByRole("button", { name: "Thinking" })).not.toBeInTheDocument();
    expect(screen.getByText("Inspecting the retry path.")).toBeInTheDocument();
  });

  it("collapses completed reasoning behind a thought summary", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWith({
          type: "content",
          id: "reasoning-1",
          kind: "reasoning",
          text: "The command can run in parallel.",
          status: "passed",
          durationMs: 8_000,
          truncated: false,
        })}
      />,
    );

    const thought = screen.getByRole("button", { name: "Thought briefly" });
    expect(thought).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("The command can run in parallel.")).not.toBeInTheDocument();

    await user.click(thought);
    expect(screen.getByText("The command can run in parallel.")).toBeInTheDocument();
  });

  it("does not render completed reasoning without visible text", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          type: "content",
          id: "reasoning-1",
          kind: "reasoning",
          text: "",
          status: "passed",
          durationMs: 2_000,
          truncated: false,
        })}
      />,
    );

    expect(screen.queryByRole("button", { name: "Thought briefly" })).not.toBeInTheDocument();
    expect(screen.queryByText("No reasoning text.")).not.toBeInTheDocument();
  });

  it("keeps assistant output plain and non-collapsible", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          type: "content",
          id: "assistant-1",
          kind: "assistant",
          text: "Here is the refined plan.",
          status: "running",
          truncated: false,
        })}
      />,
    );

    expect(screen.getByText("Here is the refined plan.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /refined plan/i })).not.toBeInTheDocument();
  });

  it("shows completed commands as plain one-line entries without output", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWithItems([
          {
            ...completedTool("command-1", "bash", "Bash"),
            input: "printf 'first line\nsecond line'",
            output: "first line\nsecond line",
          },
          {
            ...completedTool("command-2", "command_execution", "Shell"),
            input: "pwd && find . -type f",
            output: "/repo",
          },
        ])}
      />,
    );

    const summary = screen.getByRole("button", { name: "Explored repository, used terminal" });
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(summary.querySelector("svg")).toHaveClass("opacity-0", "group-hover:opacity-100");
    expect(summary.querySelector("span")).toHaveClass("whitespace-normal", "break-words");
    expect(screen.queryByText("pwd && find . -type f")).not.toBeInTheDocument();

    await user.click(summary);

    const first = screen.getByTestId("agent-tool-command-1");
    expect(first).toHaveTextContent("printf 'first line second line'");
    expect(first).toHaveClass("truncate", "whitespace-nowrap");
    expect(screen.getByText("pwd && find . -type f")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Ran command" })).not.toBeInTheDocument();
    expect(document.querySelector("pre")).not.toBeInTheDocument();
    expect(screen.queryByText("/repo")).not.toBeInTheDocument();
  });

  it("groups one or more running commands behind a text-only summary", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWithItems([
          { ...completedTool("command-1", "bash", "Bash"), input: "cat README.md", status: "running" },
          { ...completedTool("command-2", "bash", "Bash"), input: "rg --files", status: "running" },
        ])}
      />,
    );

    const summary = screen.getByRole("button", { name: "Exploring 1 file, searching code" });
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(summary.querySelector(".sp-thinking-state-current")).toBeInTheDocument();
    expect(summary.querySelector("svg")).toHaveClass("opacity-0", "group-hover:opacity-100");
    expect(screen.queryByText("cat README.md")).not.toBeInTheDocument();

    await user.click(summary);

    expect(screen.getByText("cat README.md")).toBeInTheDocument();
    expect(screen.getByText("rg --files")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-tool-details-command-1")).not.toBeInTheDocument();
  });

  it("animates a running tool group when its factual summary changes", () => {
    const firstTool = { ...completedTool("command-1", "bash", "Bash"), input: "rg retry", status: "running" as const };
    const { rerender } = render(<AgentActivityView live activity={activityWith(firstTool)} />);

    expect(screen.getByRole("button", { name: "Searching code" })).toBeInTheDocument();

    rerender(
      <AgentActivityView
        live
        activity={activityWithItems([
          firstTool,
          { ...completedTool("command-2", "bash", "Bash"), input: "cat retry.go", status: "running" },
        ])}
      />,
    );

    const summary = screen.getByRole("button", { name: "Exploring 1 file, searching code" });
    expect(summary.querySelector(".sp-thinking-state-current")).toHaveTextContent("Exploring 1 file, searching code");
    expect(summary.querySelector(".sp-thinking-state-outgoing")).toHaveTextContent("Searching code");
  });

  it("keeps one running command inside the same summary pattern", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: "cd /repo && grep -n retry pkg",
          status: "running",
        })}
      />,
    );

    const summary = screen.getByRole("button", { name: "Searching code" });
    await user.click(summary);

    expect(screen.getByTestId("agent-tool-command-1")).toHaveTextContent("cd /repo && grep -n retry pkg");
    expect(screen.queryByText("Running", { exact: true })).not.toBeInTheDocument();
  });

  it.each([
    [JSON.stringify({ command: "cd /repo && rg -n retry pkg", timeout: 30_000 }), "cd /repo && rg -n retry pkg"],
    [String.raw`{"command":"cd /repo && grep -n \"retry`, 'cd /repo && grep -n "retry'],
  ])("extracts a Bash command from provider input", async (input, command) => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input,
          status: "running",
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Searching code" }));

    expect(screen.getByTestId("agent-tool-command-1")).toHaveTextContent(command);
    expect(screen.getByTestId("agent-tool-command-1")).not.toHaveTextContent('{"command"');
  });

  it("summarizes mixed tool activity like Cursor", () => {
    render(
      <AgentActivityView
        activity={activityWithItems([
          completedTool("command-1", "bash", "Bash"),
          completedTool("command-2", "command_execution", "Shell"),
          completedTool("mcp-1", "mcp__superplane__propose_spec", "mcp__superplane__propose_spec"),
          completedTool("mcp-2", "propose_confidence", "mcp_tool_call"),
          { ...completedTool("read-1", "read", "Read"), input: JSON.stringify({ path: "src/index.ts" }) },
          { ...completedTool("search-1", "grep", "Grep"), input: "planning session" },
          { ...completedTool("web-1", "web_search", "WebSearch"), input: "streaming UI" },
        ])}
      />,
    );

    const summary = screen.getByRole("button", {
      name: "Explored 1 file, searched code, researched 1 source, prepared specification, scored task, used terminal 2 times",
    });
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(summary.querySelector("svg")).toBeInTheDocument();
  });

  it("keeps chronological tool batches collapsed inside a completed turn", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        activity={activityWithItems([
          { ...completedTool("search-1", "bash", "Bash"), input: "rg retry" },
          {
            type: "content",
            id: "update-1",
            kind: "assistant",
            text: "I will inspect the source files next.",
            status: "passed",
            truncated: false,
          },
          { ...completedTool("read-1", "bash", "Bash"), input: "cat retry.go" },
          { ...completedTool("find-1", "bash", "Bash"), input: "find . -type f" },
        ])}
      />,
    );

    const summary = screen.getByRole("button", { name: "Explored 1 file, searched code, explored repository" });
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("rg retry")).not.toBeInTheDocument();

    await user.click(summary);

    const searchBatch = screen.getByRole("button", { name: "Searched code" });
    const explorationBatch = screen.getByRole("button", { name: "Explored 1 file, explored repository" });
    expect(searchBatch).toHaveAttribute("aria-expanded", "false");
    expect(explorationBatch).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("I will inspect the source files next.")).toBeInTheDocument();
    expect(screen.queryByText("rg retry")).not.toBeInTheDocument();
    expect(screen.queryByText("cat retry.go")).not.toBeInTheDocument();

    await user.click(searchBatch);
    expect(screen.getByText("rg retry")).toBeInTheDocument();
    expect(screen.queryByText("cat retry.go")).not.toBeInTheDocument();

    await user.click(explorationBatch);
    expect(screen.getByText("cat retry.go")).toBeInTheDocument();
    expect(screen.getByText("find . -type f")).toBeInTheDocument();
  });

  it("keeps tool batches around intermediate provider updates", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWithItems([
          { ...completedTool("command-1", "bash", "Bash"), input: "rg retry" },
          {
            type: "content",
            id: "update-1",
            kind: "assistant",
            text: "I will inspect the source files next.",
            status: "passed",
            truncated: false,
          },
          { ...completedTool("command-2", "bash", "Bash"), input: "cat retry.go" },
          {
            type: "content",
            id: "final-answer",
            kind: "assistant",
            text: "The analysis is complete.",
            status: "passed",
            truncated: false,
          },
        ])}
      />,
    );

    const summaries = [
      screen.getByRole("button", { name: "Searched code" }),
      screen.getByRole("button", { name: "Explored 1 file" }),
    ];
    expect(summaries).toHaveLength(2);
    expect(screen.getByText("I will inspect the source files next.")).toBeInTheDocument();
    expect(screen.getByText("The analysis is complete.")).toBeInTheDocument();

    await user.click(summaries[0]);
    await user.click(summaries[1]);
    expect(screen.getByText("rg retry")).toBeInTheDocument();
    expect(screen.getByText("cat retry.go")).toBeInTheDocument();
  });

  it("keeps failed commands non-expandable and marks only their line", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: "false",
          status: "failed",
          exitCode: 1,
        })}
      />,
    );

    const summary = screen.getByRole("button", { name: "Used terminal" });
    expect(summary.querySelector(".text-destructive")).not.toBeInTheDocument();
    await user.click(summary);

    expect(screen.getByTestId("agent-tool-command-1")).toHaveClass("text-destructive");
    expect(screen.queryByTestId("agent-tool-details-command-1")).not.toBeInTheDocument();
    expect(screen.queryByText("Exit code 1")).not.toBeInTheDocument();
  });

  it("reveals command copy on hover and temporarily shows success", async () => {
    vi.useFakeTimers();
    const writeText = vi.fn(() => Promise.resolve());
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });

    try {
      render(
        <AgentActivityView
          live
          activity={activityWith({
            ...completedTool("command-1", "bash", "Bash"),
            input: "pwd",
            status: "running",
          })}
        />,
      );

      fireEvent.click(screen.getByRole("button", { name: "Using terminal" }));
      const copyButton = screen.getByRole("button", { name: "Copy command" });
      expect(copyButton).toHaveClass("opacity-0", "group-hover:opacity-100", "data-[copied=true]:opacity-100");

      fireEvent.click(copyButton);
      await act(async () => Promise.resolve());
      expect(writeText).toHaveBeenCalledWith("pwd");
      expect(screen.getByRole("button", { name: "Command copied" })).toHaveAttribute("data-copied", "true");

      act(() => vi.advanceTimersByTime(2000));
      expect(screen.getByRole("button", { name: "Copy command" })).not.toHaveAttribute("data-copied");
    } finally {
      vi.useRealTimers();
      vi.restoreAllMocks();
    }
  });

  it("counts all files in multi-file activity summaries", () => {
    render(
      <AgentActivityView
        live
        activity={activityWithItems([
          {
            ...completedTool("edit-1", "edit", "file_change"),
            input: "src/one.ts\nsrc/two.ts\nsrc/three.ts",
          },
          {
            ...completedTool("read-1", "read", "Read"),
            input: JSON.stringify({ files: [{ path: "src/four.ts" }, { path: "src/five.ts" }] }),
          },
        ])}
      />,
    );

    expect(screen.getByRole("button", { name: "Edited 3 files, explored 2 files" })).toBeInTheDocument();
  });

  it("uses factual names for common shell activity", () => {
    render(
      <AgentActivityView
        activity={activityWithItems([
          { ...completedTool("read-1", "bash", "Bash"), input: "cat README.md setup.py" },
          { ...completedTool("search-1", "bash", "Bash"), input: "rg -n retry pkg" },
          { ...completedTool("explore-1", "bash", "Bash"), input: "find src -type f" },
          { ...completedTool("git-1", "bash", "Bash"), input: "git status --short" },
          { ...completedTool("check-1", "bash", "Bash"), input: "make test" },
          { ...completedTool("web-1", "bash", "Bash"), input: "curl https://example.com" },
        ])}
      />,
    );

    expect(
      screen.getByRole("button", {
        name: "Explored 2 files, searched code, explored repository, researched 1 source, inspected Git, ran 1 check",
      }),
    ).toBeInTheDocument();
  });
});

function activityWith(item: AgentActivity["items"][number]): AgentActivity {
  return activityWithItems([item]);
}

function activityWithItems(items: AgentActivity["items"]): AgentActivity {
  return {
    id: "activity-1",
    provider: "codex",
    status: "running",
    sequence: 1,
    items,
    truncated: false,
  };
}

function completedTool(id: string, kind: string, name: string): AgentToolItem {
  return {
    type: "tool",
    id,
    kind,
    name,
    input: "",
    output: "",
    outputStreams: [],
    status: "passed",
    truncated: false,
  };
}
