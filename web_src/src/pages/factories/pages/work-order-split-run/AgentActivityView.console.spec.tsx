import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import type { AgentActivity, AgentToolItem } from "./agentActivity";
import { AgentActivityView } from "./AgentActivityView";

describe("AgentActivityView console", () => {
  it("hides an empty thinking row in the console", () => {
    render(
      <AgentActivityView
        live
        collapseReasoning={false}
        activity={activityWith({
          type: "content",
          id: "reasoning-1",
          kind: "reasoning",
          text: "",
          status: "running",
          truncated: false,
        })}
      />,
    );

    expect(screen.queryByRole("status", { name: "Thinking" })).not.toBeInTheDocument();
  });

  it("expands console commands like finished log rows", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        live
        collapseCompleted={false}
        collapseReasoning={false}
        expandableCommands
        tone="log"
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: "cd /tmp/opencode && curl -fL bun.zip\nunzip bun.zip",
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Researched 1 source" }));
    const command = screen.getByTestId("agent-tool-command-1");
    expect(command.tagName).toBe("PRE");
    expect(command).toHaveClass("text-foreground/90");
    expect(command).not.toHaveClass("text-muted-foreground");
  });

  it("shows finished tool output after the console command row", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        collapseCompleted={false}
        expandableCommands
        tone="log"
        activity={activityWith({
          ...completedTool("read-1", "read", "read"),
          input: "README.md",
          output: "bun is not installed on the host",
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Explored 1 file" }));
    expect(screen.getByText("README.md")).toBeInTheDocument();
    expect(screen.getByText("bun is not installed on the host")).toBeInTheDocument();
  });

  it("keeps completed reasoning visible when collapse is off", () => {
    render(
      <AgentActivityView
        live
        collapseReasoning={false}
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

    expect(screen.getByText("The command can run in parallel.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Thought briefly" })).not.toBeInTheDocument();
  });
});

function activityWith(item: AgentActivity["items"][number]): AgentActivity {
  return {
    id: "activity-1",
    provider: "codex",
    status: "running",
    sequence: 1,
    items: [item],
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
