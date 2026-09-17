export type AgentActivityStatus = "running" | "passed" | "failed" | "cancelled" | "timed_out" | "interrupted";

export type AgentContentItem = {
  type: "content";
  id: string;
  kind: "reasoning" | "assistant";
  text: string;
  status: "running" | "passed";
  startedAtMs?: number;
  durationMs?: number;
  truncated: boolean;
};

export type AgentToolItem = {
  type: "tool";
  id: string;
  kind: string;
  name: string;
  input: string;
  output: string;
  outputStreams: Array<{ stream: string; text: string }>;
  status: AgentActivityStatus;
  startedAtMs?: number;
  durationMs?: number;
  exitCode?: number;
  signal?: string;
  truncated: boolean;
};

export type AgentNoticeItem = {
  type: "notice";
  id: string;
  code: string;
  text: string;
};

export type AgentActivityItem = AgentContentItem | AgentToolItem | AgentNoticeItem;

export type AgentActivity = {
  id: string;
  provider: string;
  turn?: number;
  status: AgentActivityStatus;
  sequence: number;
  startedAtMs?: number;
  completedAtMs?: number;
  items: AgentActivityItem[];
  truncated: boolean;
};

export type AgentActivityRecord = {
  type?: string;
  schema_version?: number;
  event_id?: string;
  activity_id?: string;
  sequence?: number;
  timestamp?: string;
  provider?: string;
  turn?: number;
  id?: string;
  channel?: string;
  content_id?: string;
  tool_id?: string;
  text?: string;
  message?: string;
  code?: string;
  kind?: string;
  name?: string;
  input?: string;
  partial_json?: string;
  complete?: boolean;
  output_stream?: string;
  status?: string;
  started_at?: number;
  duration_ms?: number;
  exit_code?: number;
  signal?: string;
  truncated?: boolean;
};

export type AgentActivityState = {
  activities: AgentActivity[];
  seenEventIds: ReadonlySet<string>;
};

export const emptyAgentActivityState: AgentActivityState = { activities: [], seenEventIds: new Set() };

export function reduceAgentActivityRecords(
  state: AgentActivityState,
  records: AgentActivityRecord[],
): AgentActivityState {
  return records.reduce(reduceAgentActivityRecord, state);
}

export function reduceAgentActivityRecord(state: AgentActivityState, record: AgentActivityRecord): AgentActivityState {
  if (record.schema_version !== 2 || !record.activity_id || !record.event_id || !record.sequence) {
    return state;
  }
  if (state.seenEventIds.has(record.event_id)) {
    return state;
  }

  const seenEventIds = new Set(state.seenEventIds);
  seenEventIds.add(record.event_id);
  const position = state.activities.findIndex((activity) => activity.id === record.activity_id);
  const activity =
    position >= 0
      ? state.activities[position]
      : newAgentActivity(
          record.activity_id,
          record.provider,
          record.turn,
          record.started_at ?? parseTimestamp(record.timestamp),
        );
  const activityWithGap = appendSequenceGap(activity, record);
  const nextActivity = applyActivityRecord(activityWithGap, record);
  const activities = [...state.activities];
  if (position >= 0) {
    activities[position] = nextActivity;
  } else {
    activities.push(nextActivity);
  }
  return { activities, seenEventIds };
}

export function mergeAgentActivities(persisted: AgentActivity[] = [], live: AgentActivity[] = []): AgentActivity[] {
  const merged = [...persisted];
  const positions = new Map(merged.map((activity, index) => [activity.id, index]));
  for (const activity of live) {
    const position = positions.get(activity.id);
    if (position === undefined) {
      positions.set(activity.id, merged.length);
      merged.push(activity);
      continue;
    }
    if (activity.sequence >= merged[position].sequence) {
      merged[position] = activity;
    }
  }
  return merged;
}

/** Current live turn only. A completed prior turn must stay collapsed in the transcript. */
export function currentLiveActivity(
  persisted: AgentActivity[] = [],
  live: AgentActivity[] = [],
): AgentActivity | undefined {
  const merged = mergeAgentActivities(persisted, live);
  for (let index = merged.length - 1; index >= 0; index -= 1) {
    if (merged[index].status === "running") {
      return merged[index];
    }
  }

  const latestLive = live.at(-1);
  if (!latestLive) {
    return undefined;
  }
  return merged.find((activity) => activity.id === latestLive.id) ?? latestLive;
}

function appendSequenceGap(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  const sequence = record.sequence ?? 0;
  if (activity.sequence === 0 || sequence <= activity.sequence + 1) {
    return activity;
  }
  const id = `sequence-gap-${activity.sequence + 1}-${sequence - 1}`;
  if (activity.items.some((item) => item.id === id)) {
    return activity;
  }
  return {
    ...activity,
    items: [
      ...activity.items,
      {
        type: "notice",
        id,
        code: "sequence_gap",
        text: "Some live activity could not be loaded.",
      },
    ],
  };
}

function newAgentActivity(id: string, provider?: string, turn?: number, startedAtMs?: number): AgentActivity {
  return {
    id,
    provider: provider ?? "agent",
    turn,
    status: "running",
    sequence: 0,
    startedAtMs,
    items: [],
    truncated: false,
  };
}

function applyActivityRecord(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  const withSequence = {
    ...activity,
    provider: record.provider ?? activity.provider,
    turn: record.turn ?? activity.turn,
    sequence: Math.max(activity.sequence, record.sequence ?? 0),
  };
  switch (record.type) {
    case "activity_start":
      return { ...withSequence, status: "running", startedAtMs: record.started_at ?? activity.startedAtMs };
    case "activity_end":
      return {
        ...withSequence,
        status: activityStatus(record.status),
        completedAtMs: parseTimestamp(record.timestamp),
      };
    case "content_start":
      return startContent(withSequence, record);
    case "content_end":
      return endContent(withSequence, record);
    case "tool_start":
      return startTool(withSequence, record);
    case "tool_input_delta":
      return updateToolInput(withSequence, record);
    case "tool_end":
      return endTool(withSequence, record);
    case "line":
      return appendLine(withSequence, record);
    case "activity_notice":
      return appendNotice(withSequence, record);
    default:
      return appendNotice(withSequence, {
        ...record,
        id: `unknown-${record.event_id}`,
        code: "unknown_event",
        message: `Received unsupported activity event: ${record.type || "unknown"}.`,
      });
  }
}

function startContent(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  const id = record.id?.trim();
  if (!id || activity.items.some((item) => item.id === id)) {
    return activity;
  }
  const kind = record.channel === "reasoning" ? "reasoning" : "assistant";
  return {
    ...activity,
    items: [
      ...activity.items,
      { type: "content", id, kind, text: "", status: "running", startedAtMs: record.started_at, truncated: false },
    ],
  };
}

function endContent(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  return updateItem(activity, record.id, (item) =>
    item.type === "content"
      ? {
          ...item,
          status: "passed",
          durationMs: record.duration_ms,
          truncated: item.truncated || Boolean(record.truncated),
        }
      : item,
  );
}

function startTool(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  const id = record.id?.trim();
  if (!id || activity.items.some((item) => item.id === id)) {
    return activity;
  }
  return {
    ...activity,
    items: [
      ...activity.items,
      {
        type: "tool",
        id,
        kind: record.kind?.trim() || "tool",
        name: record.name?.trim() || record.kind?.trim() || "tool",
        input: record.input ?? record.text ?? "",
        output: "",
        outputStreams: [],
        status: "running",
        startedAtMs: record.started_at,
        truncated: Boolean(record.truncated),
      },
    ],
  };
}

function updateToolInput(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  return updateItem(activity, record.id, (item) =>
    item.type === "tool"
      ? {
          ...item,
          input: nextToolInput(item.input, record.partial_json),
          truncated: item.truncated || Boolean(record.truncated),
        }
      : item,
  );
}

function nextToolInput(current: string, partialJson: string | undefined): string {
  if (partialJson == null || partialJson === "") {
    return current;
  }
  return partialJson;
}

function endTool(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  return updateItem(activity, record.id, (item) =>
    item.type === "tool"
      ? {
          ...item,
          status: activityStatus(record.status),
          durationMs: record.duration_ms,
          exitCode: record.exit_code,
          signal: record.signal,
          truncated: item.truncated || Boolean(record.truncated),
        }
      : item,
  );
}

function appendLine(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  if (record.channel === "tool_output") {
    return updateItem(activity, record.tool_id, (item) =>
      item.type === "tool"
        ? {
            ...item,
            output: boundedLiveOutput(item.output + (record.text ?? "")),
            outputStreams: boundedLiveOutputStreams([
              ...item.outputStreams,
              { stream: record.output_stream ?? "stdout", text: record.text ?? "" },
            ]),
            truncated: item.truncated || Boolean(record.truncated),
          }
        : item,
    );
  }
  if (record.channel === "reasoning" || record.channel === "assistant") {
    return updateItem(activity, record.content_id, (item) =>
      item.type === "content"
        ? {
            ...item,
            text: boundedLiveContent(item.text + (record.text ?? "")),
            truncated: item.truncated || Boolean(record.truncated),
          }
        : item,
    );
  }
  return activity;
}

function boundedLiveContent(value: string): string {
  return value.slice(0, 64 * 1024);
}

function boundedLiveOutput(value: string): string {
  const limit = 32 * 1024;
  if (value.length <= limit) {
    return value;
  }
  return `${value.slice(0, 8 * 1024)}${value.slice(-(24 * 1024))}`;
}

function boundedLiveOutputStreams(
  streams: Array<{ stream: string; text: string }>,
): Array<{ stream: string; text: string }> {
  const limit = 32 * 1024;
  if (streams.reduce((total, output) => total + output.text.length, 0) <= limit) return streams;
  return [...takeLiveOutputStreams(streams, 8 * 1024), ...takeLiveOutputStreams(streams, 24 * 1024, true)];
}

function takeLiveOutputStreams(
  streams: Array<{ stream: string; text: string }>,
  limit: number,
  fromEnd = false,
): Array<{ stream: string; text: string }> {
  const selected: Array<{ stream: string; text: string }> = [];
  let remaining = limit;
  const values = fromEnd ? [...streams].reverse() : streams;
  for (const output of values) {
    if (remaining <= 0) break;
    const text = fromEnd ? output.text.slice(-remaining) : output.text.slice(0, remaining);
    if (fromEnd) selected.unshift({ stream: output.stream, text });
    else selected.push({ stream: output.stream, text });
    remaining -= text.length;
  }
  return selected;
}

function appendNotice(activity: AgentActivity, record: AgentActivityRecord): AgentActivity {
  const id = record.id || record.event_id;
  if (!id) {
    return activity;
  }
  return {
    ...activity,
    items: [
      ...activity.items,
      { type: "notice", id, code: record.code ?? "notice", text: record.message ?? "Agent activity notice" },
    ],
  };
}

function updateItem(
  activity: AgentActivity,
  id: string | undefined,
  update: (item: AgentActivityItem) => AgentActivityItem,
): AgentActivity {
  if (!id || !activity.items.some((item) => item.id === id)) {
    return activity;
  }
  return { ...activity, items: activity.items.map((item) => (item.id === id ? update(item) : item)) };
}

function activityStatus(status: string | undefined): AgentActivityStatus {
  if (
    status === "passed" ||
    status === "failed" ||
    status === "cancelled" ||
    status === "timed_out" ||
    status === "interrupted"
  ) {
    return status;
  }
  return "failed";
}

function parseTimestamp(timestamp: string | undefined): number | undefined {
  if (!timestamp) {
    return undefined;
  }
  const value = Date.parse(timestamp);
  return Number.isNaN(value) ? undefined : value;
}
