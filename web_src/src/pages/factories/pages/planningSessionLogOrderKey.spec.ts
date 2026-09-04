import { describe, expect, it } from "vitest";

import { groupPlanningSessionLog, mergePlanningSessionNotes } from "./planningSessionLog";
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

function talkNames(lines: SplitRunStreamLine[]): string[] {
  return groupPlanningSessionLog(lines).flatMap((group) =>
    group.events.filter((event) => event.kind === "note").map((event) => event.line.componentName),
  );
}

describe("mergePlanningSessionNotes ordering by orderKey", () => {
  it("places a user message after an earlier agent error even though it sits in the same wait slot", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "wait",
        componentType: "prompt",
        componentName: "Wait for the next user message",
        status: "running",
        orderKey: 1_000,
      }),
      note({
        id: "error",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Something went wrong: the branch already exists.",
        orderKey: 1_005,
      }),
      note({
        id: "resolved",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Never mind, I found the existing branch and picked up from there.",
        orderKey: 1_010,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-1",
        componentType: "prompt",
        componentName: "Can you check what went wrong?",
        userTalk: "message",
        orderKey: 1_008,
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["wait", "error", "user-1", "resolved"]);
    expect(merged[2]?.noteParentId).toBe("wait");
  });

  it("interleaves multiple user messages around agent notes by increasing orderKey", () => {
    const live: SplitRunStreamLine[] = [
      note({ id: "wait", componentType: "prompt", componentName: "Wait for the next user message", orderKey: 0 }),
      note({
        id: "note-a",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Looking into it.",
        orderKey: 10,
      }),
      note({
        id: "note-b",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Still working.",
        orderKey: 30,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({ id: "user-1", componentType: "prompt", componentName: "First question", orderKey: 20 }),
      note({ id: "user-2", componentType: "prompt", componentName: "Second question", orderKey: 40 }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["wait", "note-a", "user-1", "note-b", "user-2"]);
  });

  it("keeps a user after agent notes already on screen when no done line exists", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "wait",
        componentType: "prompt",
        componentName: "Wait for the next user message",
        status: "running",
      }),
      note({
        id: "look",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Let me take a look at the repo.",
      }),
      note({
        id: "got-it",
        noteParentId: "wait",
        componentType: "note",
        componentName: "This is a small Express app.",
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-1",
        componentType: "prompt",
        componentName: "Add color to puppies",
        userTalk: "message",
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["wait", "look", "got-it", "user-1"]);
  });

  it("does not pull a later user into a hole made by an extra mid-turn done line", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "wait",
        componentType: "prompt",
        componentName: "Wait for the next user message",
        status: "running",
      }),
      note({
        id: "first",
        noteParentId: "wait",
        componentType: "note",
        componentName: "I started looking.",
      }),
      note({
        id: "retry-done",
        noteParentId: "wait",
        componentType: "note",
        componentName: "✓ done · 1 turns · $0.0018 · 1.5s",
      }),
      note({
        id: "second",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Still working through the repo.",
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-1",
        componentType: "prompt",
        componentName: "any update?",
        userTalk: "message",
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["wait", "first", "retry-done", "second", "user-1"]);
    expect(talkNames(merged)).toEqual(["I started looking.", "Still working through the repo.", "any update?"]);
  });

  it("uses a follow-up cmd_start section as the user bubble and hides runner done footers", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "greet",
        componentType: "prompt",
        componentName: "Greet the user in plain text. Then stop.",
        orderKey: 500,
      }),
      note({
        id: "greet-note",
        noteParentId: "greet",
        componentType: "note",
        componentName: "Hi there. What would you like to do?",
        orderKey: 500,
      }),
      note({
        id: "greet-done",
        noteParentId: "greet",
        componentType: "note",
        componentName: "✓ done · 1 turns · $0.0164 · 1.7s",
        orderKey: 500,
      }),
      note({
        id: "wait",
        componentType: "prompt",
        componentName: "Wait for the next user message",
        status: "running",
        orderKey: 1_000,
      }),
      note({
        id: "agent-step-7",
        componentType: "prompt",
        componentName: "hello",
        orderKey: 1_100,
      }),
      note({
        id: "hello-reply",
        noteParentId: "agent-step-7",
        componentType: "note",
        componentName: "Hello. What would you like to work on today?",
        orderKey: 1_100,
      }),
      note({
        id: "hello-done",
        noteParentId: "agent-step-7",
        componentType: "note",
        componentName: "✓ done · 1 turns · $0.0018 · 1.6s",
        orderKey: 1_100,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-hello",
        componentType: "prompt",
        componentName: "hello",
        userTalk: "message",
        orderKey: 1_050,
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual([
      "greet",
      "greet-note",
      "greet-done",
      "wait",
      "agent-step-7",
      "hello-reply",
      "hello-done",
    ]);
    expect(talkNames(merged)).toEqual([
      "Hi there. What would you like to do?",
      "hello",
      "Hello. What would you like to work on today?",
    ]);
  });

  it("places unmatched follow-ups between later turn sections by orderKey, not after each done line", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "greet",
        componentType: "prompt",
        componentName: "Greet the user in plain text. Then stop.",
        orderKey: 500,
      }),
      note({
        id: "greet-note",
        noteParentId: "greet",
        componentType: "note",
        componentName: "Hi there. I'm ready to help you plan work in this repository.",
        orderKey: 500,
      }),
      note({
        id: "hello-turn",
        componentType: "prompt",
        componentName: "Plan with the user",
        orderKey: 1_100,
      }),
      note({
        id: "hello-reply",
        noteParentId: "hello-turn",
        componentType: "note",
        componentName: "Hello. What would you like to work on today?",
        orderKey: 1_100,
      }),
      note({
        id: "alright-turn",
        componentType: "prompt",
        componentName: "Plan with the user",
        orderKey: 1_300,
      }),
      note({
        id: "alright-reply",
        noteParentId: "alright-turn",
        componentType: "note",
        componentName: "Yes, all good here.",
        orderKey: 1_300,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-hello",
        componentType: "prompt",
        componentName: "hello",
        userTalk: "message",
        orderKey: 1_050,
      }),
      note({
        id: "user-alright",
        componentType: "prompt",
        componentName: "everything alright?",
        userTalk: "message",
        orderKey: 1_200,
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual([
      "greet",
      "greet-note",
      "user-hello",
      "hello-turn",
      "hello-reply",
      "user-alright",
      "alright-turn",
      "alright-reply",
    ]);
    expect(talkNames(merged)).toEqual([
      "Hi there. I'm ready to help you plan work in this repository.",
      "hello",
      "Hello. What would you like to work on today?",
      "everything alright?",
      "Yes, all good here.",
    ]);
  });

  it("hides Codex and OpenRouter done footers and keeps users after the notes already on screen", () => {
    const live: SplitRunStreamLine[] = [
      note({ id: "wait", componentType: "prompt", componentName: "Wait for the next user message", status: "running" }),
      note({
        id: "codex-reply",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Hello. What would you like to work on today?",
      }),
      note({
        id: "codex-done",
        noteParentId: "wait",
        componentType: "note",
        componentName: "✓ done · 1 turns · 1.6s",
      }),
      note({
        id: "openrouter-reply",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Yes, all good here.",
      }),
      note({
        id: "openrouter-done",
        noteParentId: "wait",
        componentType: "note",
        componentName: "✓ done · 2 turns · $0.0020 · 2.2s",
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({ id: "user-1", componentType: "prompt", componentName: "hello", userTalk: "message" }),
      note({ id: "user-2", componentType: "prompt", componentName: "everything alright?", userTalk: "message" }),
    ]);

    expect(merged.map((line) => line.id)).toEqual([
      "wait",
      "codex-reply",
      "codex-done",
      "openrouter-reply",
      "openrouter-done",
      "user-1",
      "user-2",
    ]);
    expect(talkNames(merged)).toEqual([
      "Hello. What would you like to work on today?",
      "Yes, all good here.",
      "hello",
      "everything alright?",
    ]);
  });

  it("falls back to the end of the wait group when the live log has no orderKey data", () => {
    const live: SplitRunStreamLine[] = [
      note({ id: "wait", componentType: "prompt", componentName: "Wait for the next user message", status: "running" }),
      note({ id: "error", noteParentId: "wait", componentType: "note", componentName: "Something went wrong." }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({ id: "user-1", componentType: "prompt", componentName: "Can you check?", userTalk: "message" }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["wait", "error", "user-1"]);
  });
});
