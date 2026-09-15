import { act, fireEvent, render, screen, within } from "@testing-library/react";
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

  it("renders live commands directly and keeps completed command details unmounted", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWithItems([
          {
            type: "tool",
            id: "command-1",
            kind: "bash",
            name: "Bash",
            input: "printf 'first line\\nsecond line'",
            output: "first line\nsecond line",
            outputStreams: [],
            status: "passed",
            truncated: false,
          },
          {
            type: "content",
            id: "reasoning-empty",
            kind: "reasoning",
            text: "",
            status: "passed",
            durationMs: 500,
            truncated: false,
          },
          {
            type: "tool",
            id: "command-2",
            kind: "command_execution",
            name: "Shell",
            input: "pwd && find . -type f",
            output: "/repo",
            outputStreams: [],
            status: "passed",
            truncated: false,
          },
        ])}
      />,
    );

    expect(screen.queryByRole("button", { name: "Ran 2 commands" })).not.toBeInTheDocument();
    const tool = screen.getByTestId("agent-tool-command-1");
    const summary = within(tool).getByRole("button", { name: "Ran command" });
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(screen.getAllByRole("button", { name: "Ran command" })).toHaveLength(2);
    expect(within(tool).getByTestId("agent-tool-summary-command-1")).toHaveClass("truncate", "whitespace-nowrap");
    expect(tool.querySelector("pre")).not.toBeInTheDocument();

    await user.click(summary);
    expect(tool.querySelectorAll("pre")).toHaveLength(2);
    expect(within(tool).getByTestId("agent-tool-details-command-1")).toHaveClass("ml-6");
    const command = within(tool).getByTestId("agent-detail-command-1-command");
    const output = within(tool).getByTestId("agent-detail-command-1-output");
    expect(command).not.toHaveClass("border-l", "border-l-2");
    expect(output).toHaveClass("border-l", "text-muted-foreground");
    expect(within(command).getByText("Command")).toHaveClass("sr-only");
    expect(within(output).getByText("Output")).toHaveClass("sr-only");
  });

  it("groups concurrent running commands and keeps their details collapsed", () => {
    render(
      <AgentActivityView
        live
        activity={activityWithItems([
          {
            ...completedTool("command-1", "bash", "Bash"),
            input: "cat README.md",
            status: "running",
          },
          {
            ...completedTool("command-2", "bash", "Bash"),
            input: "rg --files",
            status: "running",
          },
        ])}
      />,
    );

    const group = screen.getByRole("button", { name: "Running 2 commands" });
    expect(group).toHaveAttribute("aria-expanded", "true");

    const commands = [
      screen.getByRole("button", { name: "Running cat README.md" }),
      screen.getByRole("button", { name: "Running rg --files" }),
    ];
    expect(commands.every((command) => command.getAttribute("aria-expanded") === "false")).toBe(true);
    expect(screen.queryByTestId("agent-tool-details-command-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-tool-command-1")).toHaveClass("sp-tool-enter");
  });

  it("shows one running command directly with a distinct command gradient", () => {
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

    expect(screen.queryByRole("button", { name: "Running 1 command" })).not.toBeInTheDocument();
    const command = screen.getByRole("button", { name: "Running grep -n retry pkg" });
    expect(command).toHaveAttribute("aria-expanded", "false");
    expect(within(command).getByText("Running")).toHaveClass("sp-ai-thinking");
    expect(within(command).getByText("grep -n retry pkg")).toHaveClass("sp-running-command");
  });

  it("extracts the command from structured live Bash input", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: JSON.stringify({ command: "cd /repo && rg -n retry pkg", timeout: 30_000 }),
          status: "running",
        })}
      />,
    );

    const command = screen.getByRole("button", { name: "Running rg -n retry pkg" });
    expect(within(command).getByText("rg -n retry pkg")).toBeInTheDocument();
    expect(command).not.toHaveTextContent('{"command"');
  });

  it("extracts the available command from partial live Bash JSON", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: String.raw`{"command":"cd /repo && grep -n \"retry`,
          status: "running",
        })}
      />,
    );

    expect(screen.getByRole("button", { name: 'Running grep -n "retry' })).toBeInTheDocument();
  });

  it("summarizes a completed structured Bash input as a command", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: JSON.stringify({ command: "git status" }),
        })}
      />,
    );

    const tool = screen.getByTestId("agent-tool-command-1");
    expect(within(tool).getByTestId("agent-tool-summary-command-1")).toHaveTextContent("git status");
    expect(tool).not.toHaveTextContent('{"command"');
  });

  it("shows the command instead of its structured wrapper in details", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: JSON.stringify({ command: "printf 'ok'", timeout: 30_000 }),
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Ran command" }));

    const details = screen.getByTestId("agent-detail-command-1-command");
    expect(details).toHaveTextContent("printf 'ok'");
    expect(details).not.toHaveTextContent('{"command"');
  });

  it("keeps a running MCP call collapsed", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("mcp-1", "mcp", "mcp__superplane__propose_spec"),
          input: JSON.stringify({ body: "A long specification body" }),
          status: "running",
        })}
      />,
    );

    const mcp = screen.getByRole("button", { name: "Running mcp__superplane__propose_spec" });
    expect(mcp).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("agent-tool-details-mcp-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-tool-summary-mcp-1")).toHaveClass("truncate", "whitespace-nowrap");
  });

  it("reveals copy actions on block interaction and temporarily shows success", async () => {
    vi.useFakeTimers();
    const writeText = vi.fn(() => Promise.resolve());
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    try {
      render(
        <AgentActivityView
          live
          activity={activityWith({
            type: "tool",
            id: "command-1",
            kind: "bash",
            name: "Bash",
            input: "pwd",
            output: "",
            outputStreams: [],
            status: "running",
            truncated: false,
          })}
        />,
      );

      fireEvent.click(screen.getByRole("button", { name: "Running pwd" }));
      const copyButton = screen.getByRole("button", { name: "Copy command" });
      expect(copyButton).toHaveClass(
        "opacity-0",
        "group-hover:opacity-100",
        "focus-visible:opacity-100",
        "data-[copied=true]:opacity-100",
      );

      fireEvent.click(copyButton);
      await act(async () => Promise.resolve());

      expect(writeText).toHaveBeenCalledWith("pwd");
      expect(screen.getByRole("button", { name: "Command copied" })).toHaveAttribute("data-copied", "true");

      act(() => {
        vi.advanceTimersByTime(2000);
      });

      expect(screen.getByRole("button", { name: "Copy command" })).not.toHaveAttribute("data-copied");
    } finally {
      vi.useRealTimers();
      vi.restoreAllMocks();
    }
  });

  it("renders one completed live command without a group wrapper", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          type: "tool",
          id: "command-1",
          kind: "bash",
          name: "Bash",
          input: "pwd",
          output: "/repo",
          outputStreams: [],
          status: "passed",
          truncated: false,
        })}
      />,
    );

    expect(screen.queryByRole("button", { name: "Ran 1 command" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ran command" }).getAttribute("aria-expanded")).toBe("false");
  });

  it("keeps intermediate provider updates between completed tools", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        activity={activityWithItems([
          completedTool("command-1", "bash", "Bash"),
          {
            type: "content",
            id: "update-1",
            kind: "assistant",
            text: "I will inspect the source files next.",
            status: "passed",
            truncated: false,
          },
          completedTool("command-2", "bash", "Bash"),
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

    await user.click(screen.getByRole("button", { name: "Ran 2 commands" }));

    const details = screen.getByTestId("agent-activity-details-activity-1");
    const orderedEntries = Array.from(details.children);
    expect(orderedEntries[0]).toHaveAttribute("data-testid", "agent-tool-command-1");
    expect(orderedEntries[1]).toHaveAttribute("data-testid", "agent-assistant-update-1");
    expect(orderedEntries[2]).toHaveAttribute("data-testid", "agent-tool-command-2");
    expect(screen.getByText("I will inspect the source files next.")).toBeInTheDocument();
    expect(screen.queryByText("The analysis is complete.")).not.toBeInTheDocument();
  });

  it("summarizes all tool activity above a completed turn", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        activity={activityWithItems([
          completedTool("command-1", "bash", "Bash"),
          completedTool("command-2", "command_execution", "Shell"),
          completedTool("mcp-1", "mcp__superplane__propose_spec", "mcp__superplane__propose_spec"),
          completedTool("mcp-2", "propose_confidence", "mcp_tool_call"),
          completedTool("read-1", "read", "Read"),
        ])}
      />,
    );

    const summary = screen.getByRole("button", {
      name: "Ran 2 commands, 2 MCP calls, and 1 file read",
    });
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("button", { name: "Ran 2 commands" })).not.toBeInTheDocument();

    await user.click(summary);

    expect(screen.getByTestId("agent-activity-details-activity-1")).toHaveClass("ml-1", "pl-1.5");
    expect(screen.getByTestId("agent-activity-details-activity-1")).not.toHaveClass("ml-1.5");
    expect(screen.queryByRole("button", { name: "Ran 2 commands" })).not.toBeInTheDocument();
    const commands = screen.getAllByRole("button", { name: "Ran command" });
    expect(commands).toHaveLength(2);
    expect(commands.every((command) => command.getAttribute("aria-expanded") === "false")).toBe(true);
    expect(screen.getByRole("button", { name: "Ran mcp__superplane__propose_spec" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Read file" })).toBeInTheDocument();
  });

  it("keeps a failed activity summary and its failed child collapsed", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          status: "failed",
          exitCode: 1,
        })}
      />,
    );

    const summary = screen.getByTestId("agent-activity-summary-activity-1");
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(summary.querySelector(".text-destructive")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Ran command" })).not.toBeInTheDocument();

    await user.click(summary);

    expect(screen.getByRole("button", { name: "Ran command" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("agent-tool-details-command-1")).not.toBeInTheDocument();
  });

  it("keeps a standalone failed tool collapsed", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          status: "failed",
          exitCode: 1,
        })}
      />,
    );

    expect(screen.getByRole("button", { name: "Ran command" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("agent-tool-details-command-1")).not.toBeInTheDocument();
  });

  it("summarizes multi-file edits factually", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          type: "tool",
          id: "edit-1",
          kind: "edit",
          name: "file_change",
          input: "src/one.ts\nsrc/two.ts\nsrc/three.ts\nsrc/four.ts\nsrc/five.ts",
          output: "",
          outputStreams: [],
          status: "passed",
          truncated: false,
        })}
      />,
    );

    expect(screen.getByRole("button", { name: "Edited 5 files" })).toHaveAttribute("aria-expanded", "false");
  });

  it("summarizes multi-file reads factually", () => {
    render(
      <AgentActivityView
        live
        activity={activityWith({
          type: "tool",
          id: "read-1",
          kind: "read",
          name: "Read",
          input: JSON.stringify({ files: [{ path: "src/one.ts" }, { path: "src/two.ts" }] }),
          output: "",
          outputStreams: [],
          status: "passed",
          truncated: false,
        })}
      />,
    );

    expect(screen.getByRole("button", { name: "Read 2 files" })).toHaveAttribute("aria-expanded", "false");
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
