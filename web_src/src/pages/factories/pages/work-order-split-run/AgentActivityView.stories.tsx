import type { Meta, StoryObj } from "@storybook/react-vite";

import { AgentActivityView } from "./AgentActivityView";
import type { AgentActivity } from "./agentActivity";

const runningActivity: AgentActivity = {
  id: "activity-running",
  provider: "codex",
  status: "running",
  sequence: 7,
  truncated: false,
  items: [
    {
      type: "content",
      id: "reasoning",
      kind: "reasoning",
      text: "Checking the command lifecycle and its persisted state.",
      status: "running",
      truncated: false,
    },
    {
      type: "tool",
      id: "command",
      kind: "bash",
      name: "Bash",
      input: "for file in src/**/*.ts; do\n  printf '%s\\n' \"$file\"\ndone",
      output: "src/index.ts\nsrc/server.ts\n",
      outputStreams: [{ stream: "stdout", text: "src/index.ts\nsrc/server.ts\n" }],
      status: "running",
      truncated: false,
    },
  ],
};

const meta = {
  title: "Factories/Refinement/AgentActivityView",
  component: AgentActivityView,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <div className="max-w-2xl rounded-lg bg-background p-4 text-foreground">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof AgentActivityView>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Running: Story = { args: { activity: runningActivity, live: true } };

export const Completed: Story = {
  args: {
    activity: {
      ...runningActivity,
      id: "activity-completed",
      status: "passed",
      items: runningActivity.items.map((item) =>
        item.type === "content"
          ? { ...item, status: "passed", durationMs: 8_200 }
          : { ...item, status: "passed", durationMs: 1_250, exitCode: 0 },
      ),
    },
  },
};

export const CommandGroup: Story = {
  args: {
    activity: {
      id: "activity-command-group",
      provider: "claude",
      status: "passed",
      sequence: 8,
      truncated: false,
      items: [
        {
          type: "tool",
          id: "find-files",
          kind: "bash",
          name: "Bash",
          input:
            "cd /home/node/.superplane/homes/example/repo && find . -type f -not -path './.git/*' | head -50 && cat setup.py",
          output: "./README.md\n./setup.py\n",
          outputStreams: [{ stream: "stdout", text: "./README.md\n./setup.py\n" }],
          status: "passed",
          durationMs: 2_500,
          truncated: false,
        },
        {
          type: "tool",
          id: "read-files",
          kind: "command_execution",
          name: "Shell",
          input: "cd /home/node/.superplane/homes/example/repo && cat README.md && cat src/main.py",
          output: "# Example\n",
          outputStreams: [{ stream: "stdout", text: "# Example\n" }],
          status: "passed",
          durationMs: 1_100,
          truncated: false,
        },
        {
          type: "tool",
          id: "read-source",
          kind: "read",
          name: "Read",
          input: "src/main.py",
          output: "def main():\n    pass\n",
          outputStreams: [{ stream: "stdout", text: "def main():\n    pass\n" }],
          status: "passed",
          durationMs: 180,
          truncated: false,
        },
        {
          type: "tool",
          id: "publish-spec",
          kind: "mcp__superplane__propose_spec",
          name: "mcp__superplane__propose_spec",
          input: '{"body":"# Plan"}',
          output: "",
          outputStreams: [],
          status: "passed",
          durationMs: 900,
          truncated: false,
        },
      ],
    },
  },
};

export const Failed: Story = {
  args: {
    activity: {
      id: "activity-failed",
      provider: "claude",
      status: "failed",
      sequence: 4,
      truncated: false,
      items: [
        {
          type: "tool",
          id: "failed-command",
          kind: "bash",
          name: "Bash",
          input: "make check.build.ui",
          output: "Build failed\n",
          outputStreams: [{ stream: "stderr", text: "Build failed\n" }],
          status: "failed",
          exitCode: 1,
          durationMs: 2_400,
          truncated: false,
        },
      ],
    },
  },
};

export const EditedFiles: Story = {
  args: {
    activity: {
      id: "activity-edited-files",
      provider: "codex",
      status: "passed",
      sequence: 3,
      truncated: false,
      items: [
        {
          type: "tool",
          id: "edited-files",
          kind: "edit",
          name: "file_change",
          input: "src/one.ts\nsrc/two.ts\nsrc/three.ts\nsrc/four.ts\nsrc/five.ts",
          output: "",
          outputStreams: [],
          status: "passed",
          durationMs: 920,
          truncated: false,
        },
      ],
    },
  },
};

export const Disconnected: Story = {
  args: {
    activity: {
      ...runningActivity,
      id: "activity-disconnected",
      status: "interrupted",
      items: [
        ...runningActivity.items,
        {
          type: "notice",
          id: "gap",
          code: "sequence_gap",
          text: "Some live activity could not be loaded.",
        },
      ],
    },
  },
};
