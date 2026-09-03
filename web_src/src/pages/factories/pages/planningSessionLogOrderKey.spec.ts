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

describe("mergePlanningSessionNotes ordering by orderKey", () => {
  it("places a user message after an earlier agent error even though it sits in the same wait slot", () => {
    // Repro: setup, then the agent posts an error (later resolved), then the
    // user sends a message. With orderKey known on both sides, the merge
    // must not put the user reply before the earlier error note.
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

  it("interleaves a user reply when every live note in a turn shares the section start time", () => {
    // Real live logs stamp every note in an agent turn with the same section
    // start time, so a later user message would otherwise sort after the whole
    // turn (the bug from the report). Stamping each agent note with its own
    // message created_at restores the true order.
    const live: SplitRunStreamLine[] = [
      note({
        id: "wait",
        componentType: "prompt",
        componentName: "Wait for the next user message",
        status: "running",
        orderKey: 1_000,
      }),
      note({
        id: "greet",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Hi! What would you like to work on today?",
        orderKey: 1_000,
      }),
      note({
        id: "greet-done",
        noteParentId: "wait",
        componentType: "note",
        componentName: "✓ done · 1 turns · $0.0018 · 1.5s",
        orderKey: 1_000,
      }),
      note({
        id: "reply",
        noteParentId: "wait",
        componentType: "note",
        componentName: "No problem. Everything looks good on my end.",
        orderKey: 1_000,
      }),
      note({
        id: "reply-done",
        noteParentId: "wait",
        componentType: "note",
        componentName: "✓ done · 1 turns · $0.0184 · 2.2s",
        orderKey: 1_000,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "agent-greet",
        componentType: "note",
        componentName: "Hi! What would you like to work on today?",
        orderKey: 900,
      }),
      note({
        id: "user-1",
        componentType: "prompt",
        componentName: "hey yesy",
        userTalk: "message",
        orderKey: 1_200,
      }),
      note({
        id: "agent-reply",
        componentType: "note",
        componentName: "No problem. Everything looks good on my end.",
        orderKey: 1_500,
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["wait", "greet", "greet-done", "user-1", "reply", "reply-done"]);
    expect(merged[3]?.noteParentId).toBe("wait");
  });

  it("places each follow-up after the done line that closed the previous turn", () => {
    // Real Create with an Agent log: greet is one prompt, then a single
    // "Wait for the next user message" section holds every later agent turn.
    // Agent replies are not persisted as extras, so there is nothing to stamp.
    // User messages after the first must still sit after the done line of the
    // turn they replied to, not in a pile at the bottom.
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
        componentName: "Hi there. I'm ready to help you plan work in this repository. What would you like to do?",
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
        id: "hello-reply",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Hello. What would you like to work on today?",
        orderKey: 1_000,
      }),
      note({
        id: "hello-done",
        noteParentId: "wait",
        componentType: "note",
        componentName: "✓ done · 1 turns · $0.0018 · 1.6s",
        orderKey: 1_000,
      }),
      note({
        id: "alright-reply",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Yes, all good here. The repository is on the master branch with a clean working tree.",
        orderKey: 1_000,
      }),
      note({
        id: "alright-done",
        noteParentId: "wait",
        componentType: "note",
        componentName: "✓ done · 1 turns · $0.0022 · 1.9s",
        orderKey: 1_000,
      }),
      note({
        id: "strange-reply",
        noteParentId: "wait",
        componentType: "note",
        componentName: "What didn't work? Can you share more details, like what you tried and what happened?",
        orderKey: 1_000,
      }),
      note({
        id: "strange-done",
        noteParentId: "wait",
        componentType: "note",
        componentName: "✓ done · 1 turns · $0.0034 · 2.7s",
        orderKey: 1_000,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-hello",
        componentType: "prompt",
        componentName: "hello",
        userTalk: "message",
        orderKey: 1_100,
      }),
      note({
        id: "user-alright",
        componentType: "prompt",
        componentName: "everything alright?",
        userTalk: "message",
        orderKey: 1_200,
      }),
      note({
        id: "user-strange",
        componentType: "prompt",
        componentName: "very strange it didnt work",
        userTalk: "message",
        orderKey: 1_300,
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual([
      "greet",
      "greet-note",
      "greet-done",
      "user-hello",
      "wait",
      "hello-reply",
      "hello-done",
      "user-alright",
      "alright-reply",
      "alright-done",
      "user-strange",
      "strange-reply",
      "strange-done",
    ]);

    const talk = groupPlanningSessionLog(merged).flatMap((group) =>
      group.events.filter((event) => event.kind === "note").map((event) => event.line.componentName),
    );
    expect(talk).toEqual([
      "Hi there. I'm ready to help you plan work in this repository. What would you like to do?",
      "✓ done · 1 turns · $0.0164 · 1.7s",
      "hello",
      "Hello. What would you like to work on today?",
      "✓ done · 1 turns · $0.0018 · 1.6s",
      "everything alright?",
      "Yes, all good here. The repository is on the master branch with a clean working tree.",
      "✓ done · 1 turns · $0.0022 · 1.9s",
      "very strange it didnt work",
      "What didn't work? Can you share more details, like what you tried and what happened?",
      "✓ done · 1 turns · $0.0034 · 2.7s",
    ]);
  });

  it("places follow-ups after Codex and OpenRouter done lines the same way as Claude", () => {
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
      "user-1",
      "openrouter-reply",
      "openrouter-done",
      "user-2",
    ]);
  });

  it("falls back to the wait-slot heuristic when the live log has no orderKey data", () => {
    // Guards the existing behaviour: without timestamps we cannot tell true
    // order, so a user message still lands right after the open wait slot.
    const live: SplitRunStreamLine[] = [
      note({ id: "wait", componentType: "prompt", componentName: "Wait for the next user message", status: "running" }),
      note({ id: "error", noteParentId: "wait", componentType: "note", componentName: "Something went wrong." }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({ id: "user-1", componentType: "prompt", componentName: "Can you check?", userTalk: "message" }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["wait", "user-1", "error"]);
  });
});
