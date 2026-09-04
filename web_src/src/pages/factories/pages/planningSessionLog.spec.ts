import { describe, expect, it } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import {
  groupPlanningSessionLog,
  isPlanningSessionNoise,
  isPlanningSessionToolPayload,
  mergePlanningSessionNotes,
} from "./planningSessionLog";
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

  it("treats survey JSON as a tool payload", () => {
    expect(isPlanningSessionToolPayload('{"questions":[{"prompt":"Priority?","options":["High"]}]}')).toBe(true);
    expect(isPlanningSessionToolPayload('{"status":"shown"}')).toBe(true);
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
          componentType: "prompt",
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

  it("keeps user follow-up text and survey answers visible when the live log hides the prompt header", () => {
    const groups = groupPlanningSessionLog([
      note({
        id: "agent-step-7",
        componentType: "prompt",
        componentName: "Add a Size field to Puppy",
      }),
      note({
        id: "agent-step-8",
        componentType: "prompt",
        componentName: "Priority: High\nScope: One file",
      }),
      note({
        id: "agent-step-9",
        componentType: "prompt",
        componentName: "The user created the draft task (NEWWO-12). Acknowledge that in one short friendly sentence.",
      }),
    ]);

    const notes = groups.flatMap((group) =>
      group.events.filter((event) => event.kind === "note").map((event) => event.line.componentName),
    );
    expect(notes).toEqual(["Add a Size field to Puppy", "Priority: High\nScope: One file"]);
    expect(notes.join("\n")).not.toContain("The user created the draft task");
    expect(
      groups
        .flatMap((group) => group.events)
        .filter((event) => event.kind === "note" && event.line.componentType === "prompt"),
    ).toHaveLength(2);
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

  it("keeps survey answers marked so the log can label them", () => {
    const groups = groupPlanningSessionLog([
      note({
        id: "agent-step-8",
        componentType: "prompt",
        componentName: "How should the color field be entered? Dropdown\nShould color be required? Optional",
        userTalk: "survey",
      }),
    ]);

    expect(groups[0]?.events[0]).toEqual(
      expect.objectContaining({
        kind: "note",
        line: expect.objectContaining({
          componentType: "prompt",
          userTalk: "survey",
          componentName: "How should the color field be entered? Dropdown\nShould color be required? Optional",
        }),
      }),
    );
  });
});

describe("mergePlanningSessionNotes", () => {
  it("keeps user replies in conversation order instead of stacking them on the greeting", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "greet",
        componentType: "prompt",
        componentName: "Greet the user in plain text. Then stop.",
      }),
      note({
        id: "hi",
        noteParentId: "greet",
        componentType: "note",
        componentName: "Hi! I'm ready to help you plan work in this repo. Tell me what you'd like to do.",
      }),
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
        componentName: 'Let me take a look at the repo to understand what "puppies" refers to here.',
      }),
      note({
        id: "got-it",
        noteParentId: "wait",
        componentType: "note",
        componentName: "Got it. This is a small Express + EJS CRUD app where a Puppy currently has just a name.",
      }),
      note({
        id: "asked",
        noteParentId: "wait",
        componentType: "note",
        componentName:
          "I've asked a few scoping questions above. Once you answer, I'll have what I need to draft the task.",
      }),
    ];
    const surveyReply =
      "How should the color field be entered? Dropdown of preset colors (e.g. black, brown, white, golden)\nShould color be required or optional? Optional, matching how name works today\nWhere should color show up? skipped";

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-1",
        componentType: "prompt",
        componentName: "I want to add color to puppies",
        userTalk: "message",
      }),
      note({
        id: "user-survey",
        componentType: "prompt",
        componentName: surveyReply,
        userTalk: "survey",
      }),
    ]);

    const groups = groupPlanningSessionLog(merged);
    const talk = groups.flatMap((group) =>
      group.events.filter((event) => event.kind === "note").map((event) => event.line),
    );

    expect(talk.map((line) => line.componentName)).toEqual([
      "Hi! I'm ready to help you plan work in this repo. Tell me what you'd like to do.",
      "I want to add color to puppies",
      'Let me take a look at the repo to understand what "puppies" refers to here.',
      "Got it. This is a small Express + EJS CRUD app where a Puppy currently has just a name.",
      "I've asked a few scoping questions above. Once you answer, I'll have what I need to draft the task.",
      surveyReply,
    ]);
    expect(talk[1]?.userTalk).toBe("message");
    expect(talk[5]?.userTalk).toBe("survey");
  });

  it("keeps a later survey reply before a newer agent turn", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "wait",
        componentType: "prompt",
        componentName: "Wait for the next user message",
        status: "running",
      }),
      note({
        id: "asked",
        noteParentId: "wait",
        componentType: "note",
        componentName: "I've asked a few scoping questions above.",
      }),
      note({
        id: "later-turn",
        componentType: "prompt",
        componentName: "Plan with the user",
      }),
      note({
        id: "later-note",
        noteParentId: "later-turn",
        componentType: "note",
        componentName: "I drafted a task.",
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-1",
        componentType: "prompt",
        componentName: "Add color to puppies",
        userTalk: "message",
      }),
      note({
        id: "user-survey",
        componentType: "prompt",
        componentName: "Priority? High",
        userTalk: "survey",
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual([
      "wait",
      "user-1",
      "asked",
      "user-survey",
      "later-turn",
      "later-note",
    ]);
  });

  it("skips a user extra already in the live prompt and marks a matching survey reply", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "agent-step-7",
        componentType: "prompt",
        componentName: "I want to add color to puppies",
      }),
      note({
        id: "look",
        noteParentId: "agent-step-7",
        componentType: "note",
        componentName: "Let me take a look at the repo.",
      }),
      note({
        id: "agent-step-8",
        componentType: "prompt",
        componentName: CREATE_WITH_AGENT_COPY.surveySkipped,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-1",
        componentType: "prompt",
        componentName: "I want to add color to puppies",
        userTalk: "message",
      }),
      note({
        id: "user-survey",
        componentType: "prompt",
        componentName: CREATE_WITH_AGENT_COPY.surveySkipped,
        userTalk: "survey",
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["agent-step-7", "look", "agent-step-8"]);
    expect(merged[2]?.userTalk).toBe("survey");
  });

  it("renders the submitted answer, not the live log's own wording, for a matched survey reply", () => {
    // The live runner log can summarize a user turn in its own words (for
    // example a truncated preview). That summary is not guaranteed to spell
    // out the chosen answer, so the merge must prefer the text the user
    // actually submitted once it recognizes the turn as a survey reply.
    const live: SplitRunStreamLine[] = [
      note({
        id: "agent-step-8",
        componentType: "prompt",
        componentName: "What is the priority? High (agent noted the rest of the form was skipped)",
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-survey",
        componentType: "prompt",
        componentName: "What is the priority? High",
        userTalk: "survey",
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["agent-step-8"]);
    expect(merged[0]?.userTalk).toBe("survey");
    expect(merged[0]?.componentName).toBe("What is the priority? High");
    expect(merged[0]?.componentName).not.toContain("skipped");
  });

  it("keeps a partially answered multi-question reply intact when it matches the live log", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "agent-step-8",
        componentType: "prompt",
        componentName: "Which ORM? skipped\nWhich auth library? Passport",
      }),
    ];

    const surveyReply = "Which ORM? skipped\nWhich auth library? Passport";

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-survey",
        componentType: "prompt",
        componentName: surveyReply,
        userTalk: "survey",
      }),
    ]);

    expect(merged[0]?.componentName).toBe(surveyReply);
    expect(merged[0]?.componentName).toContain("Passport");
  });

  it("keeps the skipped label when every answer in the reply was empty", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "agent-step-8",
        componentType: "prompt",
        componentName: CREATE_WITH_AGENT_COPY.surveySkipped,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-survey",
        componentType: "prompt",
        componentName: CREATE_WITH_AGENT_COPY.surveySkipped,
        userTalk: "survey",
      }),
    ]);

    expect(merged[0]?.componentName).toBe(CREATE_WITH_AGENT_COPY.surveySkipped);
  });

  it("rewrites only the first live prompt that matches a survey reply prefix", () => {
    // Two separate root prompts happen to share the same 48-character prefix as
    // the submitted reply. Only the turn that actually was the survey reply
    // should be rewritten and labeled; the other prompt must stay untouched so a
    // second transcript turn is not duplicated or mislabeled.
    const surveyReply = "Which framework should we use for the new service? React";
    const live: SplitRunStreamLine[] = [
      note({
        id: "agent-step-1",
        componentType: "prompt",
        componentName: "Which framework should we use for the new service? (still deciding)",
      }),
      note({
        id: "agent-step-2",
        componentType: "prompt",
        componentName: "Which framework should we use for the new service? (asked again)",
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "user-survey",
        componentType: "prompt",
        componentName: surveyReply,
        userTalk: "survey",
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["agent-step-1", "agent-step-2"]);
    expect(merged[0]?.userTalk).toBe("survey");
    expect(merged[0]?.componentName).toBe(surveyReply);
    expect(merged[1]?.userTalk).not.toBe("survey");
    expect(merged[1]?.componentName).toBe("Which framework should we use for the new service? (asked again)");
  });
});
