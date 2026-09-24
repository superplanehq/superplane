import { describe, expect, it } from "vitest";

import {
  currentLiveActivity,
  emptyAgentActivityState,
  mergeAgentActivities,
  reduceAgentActivityRecords,
  type AgentActivity,
} from "./agentActivity";

describe("agent activity reducer", () => {
  it("keeps parallel tools in start order when they finish in reverse order", () => {
    const base = { schema_version: 2, activity_id: "activity-1", provider: "codex" } as const;
    const state = reduceAgentActivityRecords(emptyAgentActivityState, [
      { ...base, type: "activity_start", event_id: "1", sequence: 1, started_at: 100 },
      { ...base, type: "activity_tool_start", event_id: "2", sequence: 2, id: "a", kind: "bash", input: "echo a" },
      { ...base, type: "activity_tool_start", event_id: "3", sequence: 3, id: "b", kind: "read", input: "b.ts" },
      { ...base, type: "activity_tool_end", event_id: "4", sequence: 4, id: "b", status: "passed" },
      { ...base, type: "activity_tool_end", event_id: "5", sequence: 5, id: "a", status: "passed" },
    ]);

    expect(state.activities[0].items.map((item) => item.id)).toEqual(["a", "b"]);
  });

  it("ignores duplicate replay records", () => {
    const record = {
      schema_version: 2,
      activity_id: "activity-1",
      provider: "claude",
      type: "content_start",
      event_id: "one",
      sequence: 1,
      id: "thought",
      channel: "reasoning",
    } as const;
    const state = reduceAgentActivityRecords(emptyAgentActivityState, [record, record]);

    expect(state.activities[0].items).toHaveLength(1);
  });

  it("attributes interleaved output by tool ID", () => {
    const base = { schema_version: 2, activity_id: "activity-1", provider: "claude" } as const;
    const state = reduceAgentActivityRecords(emptyAgentActivityState, [
      { ...base, type: "tool_start", event_id: "1", sequence: 1, id: "a", kind: "bash" },
      { ...base, type: "tool_start", event_id: "2", sequence: 2, id: "b", kind: "bash" },
      { ...base, type: "line", event_id: "3", sequence: 3, channel: "tool_output", tool_id: "b", text: "b" },
      { ...base, type: "line", event_id: "4", sequence: 4, channel: "tool_output", tool_id: "a", text: "a" },
    ]);

    expect(state.activities[0].items).toMatchObject([
      { id: "a", output: "a" },
      { id: "b", output: "b" },
    ]);
  });

  it("reports a sequence gap without discarding later events", () => {
    const base = { schema_version: 2, activity_id: "activity-1", provider: "codex" } as const;
    const state = reduceAgentActivityRecords(emptyAgentActivityState, [
      { ...base, type: "activity_start", event_id: "1", sequence: 1 },
      { ...base, type: "content_start", event_id: "3", sequence: 3, id: "answer", channel: "assistant" },
    ]);

    expect(state.activities[0].items).toMatchObject([
      { type: "notice", code: "sequence_gap" },
      { type: "content", id: "answer" },
    ]);
  });

  it("replaces a persisted snapshot only with an equal or newer live snapshot", () => {
    const persisted = {
      id: "activity-1",
      provider: "claude",
      status: "running" as const,
      sequence: 5,
      items: [],
      truncated: false,
    };
    const staleLive = { ...persisted, sequence: 4 };
    const newerLive = { ...persisted, sequence: 6, provider: "codex" };

    expect(mergeAgentActivities([persisted], [staleLive])).toEqual([persisted]);
    expect(mergeAgentActivities([persisted], [newerLive])).toEqual([newerLive]);
  });

  it("keeps unknown version 2 events visible for diagnosis", () => {
    const state = reduceAgentActivityRecords(emptyAgentActivityState, [
      {
        schema_version: 2,
        activity_id: "activity-1",
        provider: "future-provider",
        type: "future_event",
        event_id: "1",
        sequence: 1,
      },
    ]);

    expect(state.activities[0].items).toMatchObject([{ type: "notice", code: "unknown_event" }]);
  });

  it("keeps start order after a reload while parallel tools finish in reverse", () => {
    const base = { schema_version: 2, activity_id: "activity-1", provider: "codex" } as const;
    const started = [
      { ...base, type: "activity_start", event_id: "1", sequence: 1 },
      { ...base, type: "tool_start", event_id: "2", sequence: 2, id: "first", kind: "bash" },
      { ...base, type: "tool_start", event_id: "3", sequence: 3, id: "second", kind: "bash" },
    ];
    const beforeReload = reduceAgentActivityRecords(emptyAgentActivityState, started);
    const afterReload = reduceAgentActivityRecords(emptyAgentActivityState, [
      ...started,
      { ...base, type: "tool_end", event_id: "4", sequence: 4, id: "second", status: "passed" },
      { ...base, type: "tool_end", event_id: "5", sequence: 5, id: "first", status: "passed" },
      { ...base, type: "activity_end", event_id: "6", sequence: 6, status: "passed" },
    ]);
    const reconciled = mergeAgentActivities(beforeReload.activities, afterReload.activities);

    expect(reconciled[0].items.map((item) => item.id)).toEqual(["first", "second"]);
    expect(reconciled[0]).toMatchObject({ status: "passed", sequence: 6 });
  });

  it("does not reopen a completed prior turn as the live activity", () => {
    const prior: AgentActivity = {
      id: "activity-1",
      provider: "claude",
      status: "passed",
      sequence: 8,
      truncated: false,
      items: [],
    };

    expect(currentLiveActivity([prior], [])).toBeUndefined();
  });

  it("uses a persisted running turn when the live stream has no activity", () => {
    const running: AgentActivity = {
      id: "activity-2",
      provider: "claude",
      status: "running",
      sequence: 3,
      truncated: false,
      items: [],
    };

    expect(currentLiveActivity([running], [])).toMatchObject({
      id: "activity-2",
      sequence: 3,
    });
    expect(currentLiveActivity([running], [{ ...running, sequence: 4 }])).toMatchObject({
      id: "activity-2",
      sequence: 4,
    });
  });

  it("keeps the current tool input when a later delta is empty", () => {
    const base = { schema_version: 2, activity_id: "activity-1", provider: "claude" } as const;
    const state = reduceAgentActivityRecords(emptyAgentActivityState, [
      { ...base, type: "tool_start", event_id: "1", sequence: 1, id: "tool-1", kind: "bash", name: "Bash", input: "" },
      {
        ...base,
        type: "tool_input_delta",
        event_id: "2",
        sequence: 2,
        id: "tool-1",
        partial_json: '{"command":"git status"}',
        complete: true,
      },
      {
        ...base,
        type: "tool_input_delta",
        event_id: "3",
        sequence: 3,
        id: "tool-1",
        partial_json: "",
        complete: true,
      },
    ]);

    expect(state.activities[0].items).toMatchObject([{ id: "tool-1", input: '{"command":"git status"}' }]);
  });

  it("fills an empty bash start from later input deltas", () => {
    const base = { schema_version: 2, activity_id: "activity-1", provider: "claude" } as const;
    const started = reduceAgentActivityRecords(emptyAgentActivityState, [
      { ...base, type: "tool_start", event_id: "1", sequence: 1, id: "tool-1", kind: "bash", name: "Bash", input: "" },
    ]);
    const filled = reduceAgentActivityRecords(started, [
      {
        ...base,
        type: "tool_input_delta",
        event_id: "2",
        sequence: 2,
        id: "tool-1",
        partial_json: '{"command":"',
        complete: false,
      },
      {
        ...base,
        type: "tool_input_delta",
        event_id: "3",
        sequence: 3,
        id: "tool-1",
        partial_json: '{"command":"git status"}',
        complete: true,
      },
    ]);

    expect(started.activities[0].items).toMatchObject([{ id: "tool-1", input: "" }]);
    expect(filled.activities[0].items).toMatchObject([{ id: "tool-1", input: '{"command":"git status"}' }]);
  });

  it("prefers canonical input and ignores incomplete provider JSON", () => {
    const base = { schema_version: 2, activity_id: "activity-1", provider: "claude" } as const;
    const state = reduceAgentActivityRecords(emptyAgentActivityState, [
      { ...base, type: "tool_start", event_id: "1", sequence: 1, id: "tool-1", kind: "bash", name: "Bash" },
      {
        ...base,
        type: "tool_input_delta",
        event_id: "2",
        sequence: 2,
        id: "tool-1",
        partial_json: '{"command":"git',
        complete: false,
      },
      {
        ...base,
        type: "tool_input_delta",
        event_id: "3",
        sequence: 3,
        id: "tool-1",
        input: "git status",
        complete: true,
      },
    ]);

    expect(state.activities[0].items).toMatchObject([{ id: "tool-1", input: "git status" }]);
  });
});
