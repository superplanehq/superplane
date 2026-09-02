import { describe, expect, it } from "vitest";

import { groupPlanningSessionLog, isPlanningSessionNoise, isPlanningSessionToolPayload } from "./planningSessionLog";
import type { SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

function note(
  partial: Partial<SplitRunStreamLine> & Pick<SplitRunStreamLine, "id" | "componentName">,
): SplitRunStreamLine {
  return {
    nodeId: "agent",
    at: "",
    note: true,
    status: "passed",
    ...partial,
  };
}

describe("isPlanningSessionNoise", () => {
  it("matches runner setup lines", () => {
    expect(isPlanningSessionNoise("Planning session tools enabled")).toBe(true);
    expect(isPlanningSessionNoise("permission mode: bypassPermissions")).toBe(true);
    expect(isPlanningSessionNoise("allowed tools: Bash,Read,Edit,Write,mcp__superplane")).toBe(true);
    expect(isPlanningSessionNoise("Claude Code started · model=claude-opus-4-8 · cwd=/home/node/repo")).toBe(true);
    expect(isPlanningSessionNoise("planning tools: mcp__superplane__propose_draft, mcp__superplane__say")).toBe(true);
  });

  it("keeps agent talk and user text", () => {
    expect(isPlanningSessionNoise("Plan with the user")).toBe(false);
    expect(isPlanningSessionNoise("The repository is ready. What do you want to do?")).toBe(false);
  });

  it("treats say and draft JSON as tool payloads", () => {
    expect(isPlanningSessionToolPayload('{"message":"Hi! I am ready to help you plan work in this repository."}')).toBe(
      true,
    );
    expect(isPlanningSessionToolPayload('{"text":"I drafted a work order to add size."}')).toBe(true);
    expect(isPlanningSessionToolPayload('{"title":"Add size","description":"Add a size field"}')).toBe(true);
    expect(
      isPlanningSessionToolPayload(
        '{"title":"Add \\"size\\" field to Puppy","description":"Extend the Puppy entity...curren…',
      ),
    ).toBe(true);
    expect(isPlanningSessionToolPayload("I'll explore the repository.")).toBe(false);
  });
});

describe("groupPlanningSessionLog", () => {
  it("collapses bash and setup noise and keeps prompts and agent talk open", () => {
    const groups = groupPlanningSessionLog([
      note({
        id: "clone",
        componentType: "bash",
        componentName: 'git clone --depth 1 "https://x-access-token:${GITHUB_TOKEN}@github.com/${REPO}.git" repo',
        detail: "Cloning into 'repo'...",
      }),
      note({
        id: "prompt",
        componentType: "prompt",
        componentName: "Plan with the user",
      }),
      note({
        id: "noise-1",
        noteParentId: "prompt",
        componentType: "note",
        componentName: "Planning session tools enabled",
      }),
      note({
        id: "noise-2",
        noteParentId: "prompt",
        componentType: "note",
        componentName: "permission mode: bypassPermissions",
      }),
      note({
        id: "noise-3",
        noteParentId: "prompt",
        componentType: "note",
        componentName: "Claude Code started · model=claude-opus-4-8 · cwd=/home/node/repo",
      }),
      note({
        id: "hello",
        noteParentId: "prompt",
        componentType: "note",
        componentName: "The repository is ready. What do you want to do?",
      }),
      note({
        id: "read",
        noteParentId: "prompt",
        componentType: "read",
        componentName: "README.md",
      }),
    ]);

    expect(groups).toHaveLength(2);
    expect(groups[0]?.line.componentName).toBe("");
    expect(groups[0]?.events).toEqual([
      expect.objectContaining({
        kind: "tools",
        tools: [expect.objectContaining({ id: "clone" })],
      }),
    ]);
    expect(groups[1]?.line.componentName).toBe("");
    expect(groups[1]?.line.componentType).toBeUndefined();
    expect(groups[1]?.events.map((event) => event.kind)).toEqual(["note", "tools"]);
    expect(groups[1]?.events[0]).toEqual(
      expect.objectContaining({
        kind: "note",
        line: expect.objectContaining({ id: "hello" }),
      }),
    );
    expect(groups[1]?.events[1]).toEqual(
      expect.objectContaining({
        kind: "tools",
        tools: [expect.objectContaining({ id: "read" })],
      }),
    );
  });

  it("hides prompt headers and collapses say or draft JSON", () => {
    const groups = groupPlanningSessionLog([
      note({
        id: "agent-step-5",
        componentType: "prompt",
        componentName: "You are in a SuperPlane planning session. The repository is cloned in the working directory.",
      }),
      note({
        id: "think",
        noteParentId: "agent-step-5",
        componentType: "note",
        componentName: "I'll greet you to start our planning session.",
      }),
      note({
        id: "say-json",
        noteParentId: "agent-step-5",
        componentType: "note",
        componentName: '{"message":"Hi! I am ready to help you plan work in this repository."}',
      }),
      note({
        id: "user-1",
        componentType: "prompt",
        componentName: "I want to extend the puppies to include size",
        detail: "I want to extend the puppies to include size",
      }),
      note({
        id: "agent-step-6",
        componentType: "prompt",
        componentName: "Wait for the next user message",
      }),
      note({
        id: "draft-json",
        noteParentId: "agent-step-6",
        componentType: "note",
        componentName: '{"title":"Add size","description":"Add a size field"}',
      }),
    ]);

    expect(groups.map((group) => group.line.componentName)).toEqual(["", ""]);
    expect(groups[0]?.events.map((event) => event.kind)).toEqual(["note", "tools", "note"]);
    expect(groups[0]?.events[0]).toEqual(
      expect.objectContaining({
        kind: "note",
        line: expect.objectContaining({ id: "think" }),
      }),
    );
    expect(groups[0]?.events[1]).toEqual(
      expect.objectContaining({
        kind: "tools",
        tools: [expect.objectContaining({ id: "say-json" })],
      }),
    );
    expect(groups[0]?.events[2]).toEqual(
      expect.objectContaining({
        kind: "note",
        line: expect.objectContaining({
          id: "user-1",
          componentType: "note",
          componentName: "I want to extend the puppies to include size",
        }),
      }),
    );
    expect(groups[1]?.events).toEqual([
      expect.objectContaining({
        kind: "tools",
        tools: [expect.objectContaining({ id: "draft-json" })],
      }),
    ]);
  });

  it("hides setup and collapses truncated draft tool JSON", () => {
    const groups = groupPlanningSessionLog([
      note({
        id: "agent-step-6",
        componentType: "prompt",
        componentName: "Wait for the next user message",
      }),
      note({
        id: "noise-1",
        noteParentId: "agent-step-6",
        componentType: "note",
        componentName: "Planning session tools enabled",
      }),
      note({
        id: "intro",
        noteParentId: "agent-step-6",
        componentType: "note",
        componentName: "Here's a draft work order for your review.",
      }),
      note({
        id: "draft",
        noteParentId: "agent-step-6",
        componentType: "mcp__superplane__propose_draft",
        componentName:
          '{"title":"Add \\"size\\" field to Puppy","description":"Extend the Puppy entity with a size attribute...curren…',
      }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0]?.events.map((event) => event.kind)).toEqual(["note", "tools"]);
    expect(groups[0]?.events[0]).toEqual(
      expect.objectContaining({
        kind: "note",
        line: expect.objectContaining({ id: "intro" }),
      }),
    );
    expect(groups[0]?.events[1]).toEqual(
      expect.objectContaining({
        kind: "tools",
        tools: [expect.objectContaining({ id: "draft" })],
      }),
    );
  });
});
