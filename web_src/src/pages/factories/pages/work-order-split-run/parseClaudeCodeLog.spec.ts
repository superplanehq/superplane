import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

import { parseClaudeCodeLog } from "./parseClaudeCodeLog";

const SAMPLE = `$ Prepare Claude Code
Claude Code ready
$ Clone Repo
Cloning into 'superplane'...
remote: Enumerating objects: 7181, done.
$ Provide description
## Goal
Add a menu.
$ Write Implementation Plan
Claude Code started · model=claude-sonnet-5
-> [Bash] cat /tmp/ORDER.md
     ## Goal
     Add a menu.
-> [Read] /home/ubuntu/superplane/web_src/src/pages/factories/pages/LineListCard.tsx
     1	import type { FactoriesFactoryLine } from "@/api-client";
Let me examine the key files.
-> [Bash] cat > /tmp/plan.md << 'EOF' Add an automations-style 3-dots overflow menu on each card and then write a very long implementation plan that should not appear in the tree
$ Use plan as output
Plan found. Using as output
`;

describe("parseClaudeCodeLog", () => {
  it("keeps configured steps and tool headers, and drops prepare plus dumped bodies", () => {
    const steps = parseClaudeCodeLog(SAMPLE, [
      { name: "Clone Repo", type: "bash" },
      { name: "Write Implementation Plan", type: "prompt" },
      { name: "Use plan as output", type: "bash" },
    ]);

    expect(steps.map((step) => ({ name: step.name, type: step.type }))).toEqual([
      { name: "Clone Repo", type: "bash" },
      { name: "Provide description", type: "bash" },
      { name: "Write Implementation Plan", type: "prompt" },
      { name: "Use plan as output", type: "bash" },
    ]);
    expect(steps[0].commands).toEqual([]);
    expect(steps[2].commands).toEqual([
      { type: "bash", name: "cat /tmp/ORDER.md" },
      { type: "read", name: "web_src/src/pages/factories/pages/LineListCard.tsx" },
      { type: "note", name: "Let me examine the key files." },
      {
        type: "bash",
        name: "cat > /tmp/plan.md << 'EOF' Add an automations-style 3-dots overflow…",
      },
    ]);
  });

  it("reads the planning runner example without keeping file dumps", () => {
    const text = readFileSync(join(dirname(fileURLToPath(import.meta.url)), "planning-claude-log.txt"), "utf8");
    const steps = parseClaudeCodeLog(text);

    expect(steps.map((step) => step.name)).toEqual([
      "Clone Repo",
      "Provide description",
      "Write Implementation Plan",
      "Use plan as output",
    ]);
    expect(steps[2].commands[0]).toEqual({ type: "bash", name: "cat /tmp/ORDER.md" });
    expect(steps[2].commands.some((command) => command.name.includes("import type"))).toBe(false);
    expect(steps[2].commands.filter((command) => command.type !== "note")).toHaveLength(42);
    expect(steps[2].commands.map((command) => command.name)).toContain(
      "Let me examine the key reference files in detail.",
    );
    expect(
      steps[2].commands.some(
        (command) => command.type === "note" && command.name.startsWith("Summary of the approach:"),
      ),
    ).toBe(true);
    expect(steps[2].commands.some((command) => command.name.includes("Claude Code started"))).toBe(false);
    expect(steps[2].commands.some((command) => command.name.startsWith("✓ done"))).toBe(false);
  });
});
