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
Now let me check factories.proto Delete rpc absence explicitly and PermissionTooltip component quickly, plus check showSuccessToast import paths.
-> [Bash] cat > /tmp/plan.md << 'EOF' Add an automations-style 3-dots overflow menu on each card and then write a very long implementation plan that should not appear in the tree
$ Use plan as output
Plan found. Using as output
$ Run Tests
FAIL pkg/foo
✗ tests failed
`;

describe("parseClaudeCodeLog", () => {
  it("keeps configured steps and tool headers, and drops prepare plus dumped bodies", () => {
    const steps = parseClaudeCodeLog(SAMPLE, [
      { name: "Clone Repo", type: "bash" },
      { name: "Write Implementation Plan", type: "prompt" },
      { name: "Use plan as output", type: "bash" },
    ]);

    expect(steps.map((step) => ({ name: step.name, type: step.type, status: step.status }))).toEqual([
      { name: "Clone Repo", type: "bash", status: "passed" },
      { name: "Provide description", type: "bash", status: "passed" },
      { name: "Write Implementation Plan", type: "prompt", status: "passed" },
      { name: "Use plan as output", type: "bash", status: "passed" },
      { name: "Run Tests", type: "bash", status: "failed" },
    ]);
    expect(steps[0]).toMatchObject({
      commands: [],
      output: "Cloning into 'superplane'...\nremote: Enumerating objects: 7181, done.",
    });
    expect(steps[3].output).toBe("Plan found. Using as output");
    expect(steps[4].output).toBe("FAIL pkg/foo");
    expect(steps[2].commands).toEqual([
      { type: "bash", name: "cat /tmp/ORDER.md", status: "passed", output: "## Goal\nAdd a menu." },
      {
        type: "read",
        name: "web_src/src/pages/factories/pages/LineListCard.tsx",
        status: "passed",
        output: '1\timport type { FactoriesFactoryLine } from "@/api-client";',
      },
      { type: "note", name: "Let me examine the key files.", status: "passed" },
      {
        type: "note",
        name: "Now let me check factories.proto Delete rpc absence explicitly and PermissionTooltip component quickly, plus check showSuccessToast import paths.",
        status: "passed",
      },
      {
        type: "bash",
        name: "cat > /tmp/plan.md << 'EOF' Add an automations-style 3-dots overflow…",
        status: "passed",
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
    expect(steps[0].output).toContain("Cloning into 'superplane'...");
    expect(steps[2].commands[0]).toMatchObject({
      type: "bash",
      name: "cat /tmp/ORDER.md",
      status: "passed",
      output: expect.stringContaining("## Goal"),
    });
    expect(steps[2].commands.some((command) => command.name.includes("import type"))).toBe(false);
    expect(steps[2].commands.filter((command) => command.type !== "note")).toHaveLength(42);
    expect(steps[2].commands.map((command) => command.name)).toContain(
      "Let me examine the key reference files in detail.",
    );
    expect(steps[2].commands.some((command) => command.name.includes("Delete is deliberately left out"))).toBe(true);
    expect(steps[2].commands.some((command) => command.name.includes("Claude Code started"))).toBe(false);
    expect(steps[2].commands.some((command) => command.name.startsWith("✓ done"))).toBe(false);
  });

  it("reads the implementation runner example without keeping file dumps", () => {
    const text = readFileSync(join(dirname(fileURLToPath(import.meta.url)), "implementation-claude-log.txt"), "utf8");
    const steps = parseClaudeCodeLog(text, [
      { name: "Set Up Git User", type: "bash" },
      { name: "Provide order", type: "bash" },
      { name: "Provide Plan", type: "bash" },
      { name: "Checkout Branch", type: "bash" },
      { name: "Set Up DCO Signing", type: "bash" },
      { name: "Implementation", type: "prompt" },
      { name: "Commit and Push", type: "bash" },
    ]);

    expect(steps.map((step) => step.name)).toEqual([
      "Set Up Git User",
      "Provide order",
      "Provide Plan",
      "Checkout Branch",
      "Set Up DCO Signing",
      "Set Up Environment",
      "Implementation",
      "Format Code",
      "Commit and Push",
    ]);
    const implementation = steps.find((step) => step.name === "Implementation");
    expect(implementation?.type).toBe("prompt");
    expect(steps.find((step) => step.name === "Set Up Git User")?.output).toContain("superplaneagent@superplane.com");
    expect(implementation?.commands[0]).toMatchObject({
      type: "bash",
      name: "cat /tmp/ORDER.md",
      status: "passed",
    });
    expect(implementation?.commands.map((command) => command.name)).toContain(
      "Now let's look at the messages file, factory_notification_consumer.go, and other referenced files.",
    );
    expect(implementation?.commands.some((command) => command.name.includes("package models"))).toBe(false);
    expect(implementation?.commands.filter((command) => command.type !== "note")).toHaveLength(139);
    expect(implementation?.commands.some((command) => command.name.startsWith("✓ done"))).toBe(false);
  });
});
