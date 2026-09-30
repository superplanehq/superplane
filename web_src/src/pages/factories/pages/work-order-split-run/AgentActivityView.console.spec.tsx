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
    expect(screen.getByText("2 lines")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "curl -fL bun.zip" }));

    const command = screen.getByTestId("agent-tool-command-1");
    expect(command.tagName).toBe("PRE");
    expect(command).toHaveClass("whitespace-pre", "text-foreground/90");
    expect(command).not.toHaveClass("text-muted-foreground");
    expect(command.textContent).toBe("cd /tmp/opencode && curl -fL bun.zip\nunzip bun.zip");
  });

  it("expands a one-line chain when the headline omits the start", async () => {
    const user = userEvent.setup();
    const script = "cd /tmp/opencode && curl -fL bun.zip";
    render(
      <AgentActivityView
        collapseCompleted={false}
        expandableCommands
        tone="log"
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: script,
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Researched 1 source" }));
    await user.click(screen.getByRole("button", { name: "curl -fL bun.zip" }));
    expect(screen.getByTestId("agent-tool-command-1").textContent).toBe(script);
  });

  it("keeps blank lines in file output", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        collapseCompleted={false}
        expandableCommands
        tone="log"
        activity={activityWith({
          ...completedTool("read-1", "read", "read"),
          input: "README.md",
          output: "one\n\n\n\ntwo",
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Explored 1 file" }));
    expect(document.querySelector("pre")?.textContent).toBe("one\n\n\n\ntwo");
    expect(screen.queryByText("Output")).not.toBeInTheDocument();
  });

  it("shows the clone command and keeps its line breaks", async () => {
    const user = userEvent.setup();
    const script = [
      "set -euo pipefail",
      'if [ -z "${REPO_URL:-}" ]; then',
      '  echo "This workspace has no repository to analyze." >&2',
      "  exit 1",
      "fi",
      'git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"',
      "rm -rf repo",
      'git clone --depth 1 --branch "${BASE:-main}" "${REPO_URL}" repo',
    ].join("\n");
    render(
      <AgentActivityView
        collapseCompleted={false}
        expandableCommands
        tone="log"
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: script,
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Inspected Git" }));
    await user.click(
      screen.getByRole("button", { name: 'git clone --depth 1 --branch "${BASE:-main}" "${REPO_URL}" repo' }),
    );

    expect(screen.getByTestId("agent-tool-command-1").textContent).toBe(script);
    expect(screen.queryByText("Output")).not.toBeInTheDocument();
  });

  it("collapses carriage-return progress into finished output lines", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        collapseCompleted={false}
        expandableCommands
        tone="log"
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: "git clone repo",
          output: "remote: Enumerating objects: 1\rremote: Enumerating objects: 9364, done.\n",
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Inspected Git" }));
    expect(screen.getByText("Output")).toBeInTheDocument();
    expect(screen.getByText("remote: Enumerating objects: 9364, done.")).toBeInTheDocument();
    expect(screen.queryByText(/Enumerating objects: 1/)).not.toBeInTheDocument();
  });

  it("shows the exit code when a console command fails", async () => {
    const user = userEvent.setup();
    render(
      <AgentActivityView
        collapseCompleted={false}
        expandableCommands
        tone="log"
        activity={activityWith({
          ...completedTool("command-1", "bash", "Bash"),
          input: "false",
          status: "failed",
          exitCode: 1,
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Used terminal" }));
    expect(screen.getByText("Exit code 1")).toBeInTheDocument();
    expect(screen.queryByText("Output")).not.toBeInTheDocument();
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
