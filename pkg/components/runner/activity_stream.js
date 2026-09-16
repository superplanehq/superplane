#!/usr/bin/env node
"use strict";

const crypto = require("crypto");

const ACTIVITY_SCHEMA_VERSION = 2;
const DEFAULT_LIMITS = Object.freeze({
  commandBytes: 16 * 1024,
  contentBytes: 64 * 1024,
  toolOutputBytes: 32 * 1024,
  toolOutputHeadBytes: 8 * 1024,
  snapshotBytes: 256 * 1024,
});
const SECRET_NAME_PATTERN = /(api[_-]?key|access[_-]?token|auth[_-]?token|run[_-]?token|secret|password|credential)/i;
const ANSI_PATTERN = /[\u001B\u009B][[\]()#;?]*(?:(?:(?:[a-zA-Z\d]*(?:;[-a-zA-Z\d\/#&.:=?%@~_]+)*)?\u0007)|(?:(?:\d{1,4}(?:[;:]\d{0,4})*)?[\dA-PR-TZcf-nq-uy=><~]))/g;
const UNSAFE_CONTROL_PATTERN = /[\u0000-\u0008\u000B\u000C\u000E-\u001A\u001C-\u001F\u007F]/g;

function analysisActivityEnabled(env) {
  return (
    String((env && env.SUPERPLANE_PLANNING_SESSION_KIND) || "") === "work_order_analysis" &&
    Boolean(String((env && env.SUPERPLANE_PLANNING_SESSION_ID) || "").trim())
  );
}

function normalizeTerminalText(value) {
  const withoutAnsi = String(value || "").replace(ANSI_PATTERN, "").replace(UNSAFE_CONTROL_PATTERN, "");
  const lines = withoutAnsi.split("\n").map((line) => line.split("\r").filter(Boolean).at(-1) || "");
  return lines.join("\n");
}

function secretValues(env) {
  return Object.entries(env || {})
    .filter(([name, value]) => SECRET_NAME_PATTERN.test(name) && String(value || "").length >= 8)
    .map(([, value]) => String(value))
    .sort((left, right) => right.length - left.length);
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function redactSensitiveText(value, env = process.env) {
  let text = normalizeTerminalText(value);
  for (const secret of secretValues(env)) {
    text = text.replace(new RegExp(escapeRegExp(secret), "g"), "[REDACTED]");
  }
  text = text
    .replace(/\bBearer\s+[A-Za-z0-9._~+/=-]+/gi, "Bearer [REDACTED]")
    .replace(/\b(?:sk|rk|gh[pousr]|github_pat|xox[baprs]|glpat|npm_)-?[A-Za-z0-9_-]{8,}\b/g, "[REDACTED]")
    .replace(/\b(?:AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_-]{35})\b/g, "[REDACTED]")
    .replace(
      /([?&](?:access[_-]?token|api[_-]?key|token|key|secret|password|signature|sig|credential)=)[^&#\s]*/gi,
      "$1[REDACTED]",
    )
    .replace(/(https?:\/\/)[^/@\s]+:[^/@\s]+@/gi, "$1[REDACTED]@");
  return text;
}

function byteLength(value) {
  return Buffer.byteLength(String(value || ""), "utf8");
}

function sliceBytes(value, maxBytes, fromEnd = false) {
  const text = String(value || "");
  if (byteLength(text) <= maxBytes) {
    return text;
  }
  const buffer = Buffer.from(text, "utf8");
  return (fromEnd ? buffer.subarray(buffer.length - maxBytes) : buffer.subarray(0, maxBytes)).toString("utf8");
}

function boundedText(value, maxBytes) {
  const text = String(value || "");
  return byteLength(text) <= maxBytes ? { text, truncated: false } : { text: sliceBytes(text, maxBytes), truncated: true };
}

function boundedOutput(value, maxBytes, headBytes) {
  const text = String(value || "");
  if (byteLength(text) <= maxBytes) {
    return { text, truncated: false };
  }
  const safeHeadBytes = Math.min(headBytes, maxBytes);
  const tailBytes = Math.max(0, maxBytes - safeHeadBytes);
  return {
    text: `${sliceBytes(text, safeHeadBytes)}${sliceBytes(text, tailBytes, true)}`,
    truncated: true,
  };
}

function boundedOutputStreams(streams, maxBytes, headBytes) {
  if (streams.reduce((total, output) => total + byteLength(output.text), 0) <= maxBytes) {
    return streams;
  }
  return [
    ...takeOutputStreams(streams, headBytes, false),
    ...takeOutputStreams(streams, maxBytes - headBytes, true),
  ];
}

function takeOutputStreams(streams, maxBytes, fromEnd) {
  const selected = [];
  let remaining = maxBytes;
  const values = fromEnd ? [...streams].reverse() : streams;
  for (const output of values) {
    if (remaining <= 0) break;
    const text = sliceBytes(output.text, remaining, fromEnd);
    const entry = { stream: output.stream, text };
    if (fromEnd) selected.unshift(entry);
    else selected.push(entry);
    remaining -= byteLength(text);
  }
  return selected;
}

function terminalStatus(status) {
  const value = String(status || "");
  return ["passed", "failed", "cancelled", "timed_out", "interrupted"].includes(value) ? value : "failed";
}

function createActivityStream(options = {}) {
  const env = options.env || process.env;
  const enabled = analysisActivityEnabled(env);
  const provider = String(options.provider || "agent");
  const turn = Number.isFinite(Number(options.turn)) ? Number(options.turn) : 1;
  const activityId = String(options.activityId || crypto.randomUUID());
  const now = options.now || Date.now;
  const writeRecord = options.writeRecord || ((record) => process.stdout.write(`${JSON.stringify(record)}\n`));
  const limits = { ...DEFAULT_LIMITS, ...(options.limits || {}) };
  const items = [];
  const contents = new Map();
  const tools = new Map();
  const seenToolIds = new Set();
  let sequence = 0;
  let startedAt = null;
  let completedAt = null;
  let status = "running";
  let persistence;
  const pendingLines = new Map();
  let lineTimer;

  if (enabled) {
    env.SUPERPLANE_ACTIVITY_ID = activityId;
    persistence = createSnapshotPersistence({ env, activityId, getSnapshot: snapshot, fetchImpl: options.fetch });
  }

  function emit(record, persistImmediately = false) {
    if (!enabled) {
      return;
    }
    sequence += 1;
    writeRecord({
      ...record,
      schema_version: ACTIVITY_SCHEMA_VERSION,
      event_id: `${activityId}:${sequence}`,
      activity_id: activityId,
      sequence,
      timestamp: new Date(now()).toISOString(),
      provider,
      turn,
    });
    persistence.schedule(persistImmediately);
  }

  function queueLine(key, record) {
    const existing = pendingLines.get(key);
    const text = `${existing ? existing.text : ""}${record.text || ""}`;
    pendingLines.set(key, { ...record, text });
    if (text.includes("\n") || byteLength(text) >= 1024) {
      flushLine(key);
      return;
    }
    if (!lineTimer) {
      lineTimer = setTimeout(flushLines, 50);
      lineTimer.unref?.();
    }
  }

  function flushLine(key) {
    const record = pendingLines.get(key);
    if (!record) {
      return;
    }
    pendingLines.delete(key);
    emit(record);
  }

  function flushLines() {
    if (lineTimer) {
      clearTimeout(lineTimer);
      lineTimer = undefined;
    }
    for (const key of [...pendingLines.keys()]) {
      flushLine(key);
    }
  }

  function start() {
    if (startedAt !== null || !enabled) {
      return;
    }
    startedAt = now();
    emit({ type: "activity_start", started_at: startedAt }, true);
  }

  function ensureStarted() {
    if (startedAt === null) {
      start();
    }
  }

  function startContent(kind, id) {
    if (!enabled || contents.has(id)) {
      return;
    }
    ensureStarted();
    const item = { type: "content", id, kind, text: "", status: "running", started_at: now(), truncated: false };
    contents.set(id, item);
    items.push(item);
    emit({ type: "content_start", id, channel: kind, started_at: item.started_at }, true);
  }

  function appendContent(kind, id, rawText) {
    if (!enabled || !rawText) {
      return;
    }
    if (!contents.has(id)) {
      startContent(kind, id);
    }
    const item = contents.get(id);
    const text = redactSensitiveText(rawText, env);
    const previousText = item.text;
    const bounded = boundedText(item.text + text, limits.contentBytes);
    const becameTruncated = bounded.truncated && !item.truncated;
    item.text = bounded.text;
    item.truncated ||= bounded.truncated;
    const acceptedText = bounded.text.startsWith(previousText) ? bounded.text.slice(previousText.length) : "";
    if (acceptedText || becameTruncated) {
      queueLine(`content:${id}`, {
        type: "line",
        text: acceptedText,
        channel: kind,
        content_id: id,
        truncated: item.truncated,
      });
    }
  }

  function endContent(id) {
    const item = contents.get(id);
    if (!item || item.status !== "running") {
      return;
    }
    flushLine(`content:${id}`);
    item.status = "passed";
    item.duration_ms = Math.max(0, now() - item.started_at);
    emit({ type: "content_end", id, channel: item.kind, duration_ms: item.duration_ms, truncated: item.truncated }, true);
  }

  function startTool(input) {
    if (!enabled) {
      return String((input && input.id) || "");
    }
    ensureStarted();
    const id = String((input && input.id) || `tool-${sequence + 1}`);
    if (seenToolIds.has(id)) {
      return id;
    }
    seenToolIds.add(id);
    const command = boundedText(redactSensitiveText((input && input.input) || "", env), limits.commandBytes);
    const item = {
      type: "tool",
      id,
      kind: String((input && input.kind) || "tool"),
      name: String((input && input.name) || (input && input.kind) || "tool"),
      input: command.text,
      output: "",
      output_streams: [],
      status: "running",
      started_at: now(),
      truncated: command.truncated,
    };
    tools.set(id, item);
    items.push(item);
    emit(
      {
        type: "tool_start",
        id,
        kind: item.kind,
        name: item.name,
        text: item.input || item.name,
        input: item.input,
        started_at: item.started_at,
        truncated: item.truncated,
      },
      true,
    );
    return id;
  }

  function updateToolInput(id, rawInput, complete = false) {
    const item = tools.get(id);
    if (!item || item.status !== "running") {
      return;
    }
    const input = boundedText(redactSensitiveText(rawInput, env), limits.commandBytes);
    item.input = input.text;
    item.truncated ||= input.truncated;
    emit({ type: "tool_input_delta", id, partial_json: input.text, complete, truncated: item.truncated });
  }

  function appendToolOutput(id, rawText, outputStream = "stdout") {
    const item = tools.get(id);
    if (!item || item.status !== "running" || !rawText) {
      return;
    }
    const text = redactSensitiveText(rawText, env);
    const output = boundedOutput(item.output + text, limits.toolOutputBytes, limits.toolOutputHeadBytes);
    item.output = output.text;
    item.truncated ||= output.truncated;
    item.output_streams = boundedOutputStreams(
      [...item.output_streams, { stream: outputStream, text }],
      limits.toolOutputBytes,
      limits.toolOutputHeadBytes,
    );
    queueLine(`tool:${id}:${outputStream}`, {
      type: "line",
      text,
      channel: "tool_output",
      tool_id: id,
      output_stream: outputStream,
      truncated: item.truncated,
    });
  }

  function endTool(id, result = {}) {
    const item = tools.get(id);
    if (!item || item.status !== "running") {
      return;
    }
    flushLine(`tool:${id}:stdout`);
    flushLine(`tool:${id}:stderr`);
    item.status = terminalStatus(result.status);
    item.duration_ms = Math.max(0, now() - item.started_at);
    if (result.exitCode !== undefined) {
      item.exit_code = Number(result.exitCode);
    }
    if (result.signal) {
      item.signal = String(result.signal);
    }
    emit(
      {
        type: "tool_end",
        id,
        kind: item.kind,
        status: item.status,
        duration_ms: item.duration_ms,
        exit_code: item.exit_code,
        signal: item.signal,
        truncated: item.truncated,
      },
      true,
    );
  }

  function notice(code, message) {
    if (!enabled) {
      return;
    }
    ensureStarted();
    const text = redactSensitiveText(message, env);
    items.push({ type: "notice", id: `notice-${sequence + 1}`, code, text, created_at: now() });
    emit({ type: "activity_notice", code, message: text }, true);
  }

  function end(nextStatus, details = {}) {
    if (!enabled || completedAt !== null) {
      return;
    }
    ensureStarted();
    for (const item of contents.values()) {
      if (item.status === "running") {
        endContent(item.id);
      }
    }
    for (const item of tools.values()) {
      if (item.status === "running") {
        endTool(item.id, { status: nextStatus === "passed" ? "passed" : nextStatus });
      }
    }
    status = terminalStatus(nextStatus);
    completedAt = now();
    emit(
      {
        type: "activity_end",
        status,
        duration_ms: Math.max(0, completedAt - startedAt),
        message: details.message ? redactSensitiveText(details.message, env) : undefined,
      },
      true,
    );
  }

  function snapshot() {
    const value = {
      schema_version: ACTIVITY_SCHEMA_VERSION,
      activity_id: activityId,
      provider,
      turn,
      sequence,
      status,
      started_at: startedAt,
      completed_at: completedAt,
      items: items.map(snapshotItem),
    };
    if (byteLength(JSON.stringify(value)) <= limits.snapshotBytes) {
      return value;
    }
    const compactItems = value.items.map((item) => {
      if (item.type !== "tool" || item.status !== "passed") {
        return item;
      }
      return { ...item, output: item.output ? "… output omitted …" : "", output_streams: [], truncated: true };
    });
    return fitSnapshotToLimit({ ...value, items: compactItems, truncated: true }, limits.snapshotBytes);
  }

  function snapshotItem(item) {
    if (item.type !== "tool") {
      return { ...item };
    }
    return {
      ...item,
      output_streams: item.output_streams,
    };
  }

  return {
    enabled,
    activityId,
    start,
    startContent,
    appendContent,
    endContent,
    startTool,
    updateToolInput,
    appendToolOutput,
    endTool,
    notice,
    end,
    snapshot,
    flush: () => {
      flushLines();
      return persistence ? persistence.flush() : Promise.resolve();
    },
  };
}

function fitSnapshotToLimit(snapshot, maxBytes) {
  if (byteLength(JSON.stringify(snapshot)) <= maxBytes) {
    return snapshot;
  }
  const items = snapshot.items.map((item) => {
    if (item.type === "content") {
      return { ...item, text: boundedText(item.text, 8 * 1024).text, truncated: true };
    }
    if (item.type === "tool") {
      return {
        ...item,
        input: boundedText(item.input, 4 * 1024).text,
        output: item.status === "passed" ? "" : boundedOutput(item.output, 8 * 1024, 2 * 1024).text,
        output_streams: [],
        truncated: true,
      };
    }
    return { ...item, text: boundedText(item.text, 2 * 1024).text };
  });
  const compact = { ...snapshot, items };
  if (byteLength(JSON.stringify(compact)) <= maxBytes) {
    return compact;
  }

  let omittedItems = 0;
  while (compact.items.length > 0 && byteLength(JSON.stringify(compact)) > maxBytes) {
    const successfulTool = compact.items.findIndex((item) => item.type === "tool" && item.status === "passed");
    compact.items.splice(successfulTool >= 0 ? successfulTool : 0, 1);
    omittedItems += 1;
  }
  if (omittedItems > 0) {
    const marker = {
      type: "notice",
      id: "snapshot-truncated",
      code: "snapshot_truncated",
      text: `${omittedItems} older activity items were omitted.`,
    };
    compact.items.unshift(marker);
    while (compact.items.length > 1 && byteLength(JSON.stringify(compact)) > maxBytes) {
      compact.items.splice(1, 1);
      omittedItems += 1;
      marker.text = `${omittedItems} older activity items were omitted.`;
    }
  }
  return compact;
}

function createSnapshotPersistence({ env, activityId, getSnapshot, fetchImpl }) {
  const baseURL = String(env.SUPERPLANE_BASE_URL || "").replace(/\/$/, "");
  const token = String(env.SUPERPLANE_RUN_TOKEN || "");
  const request = fetchImpl || globalThis.fetch;
  const enabled = Boolean(baseURL && token && typeof request === "function");
  let timer;
  let dirty = false;
  let pending = Promise.resolve();

  async function persist() {
    if (!enabled) {
      return;
    }
    const response = await request(
      `${baseURL}/api/v1/runner/planning-sessions/activities/${encodeURIComponent(activityId)}`,
      {
        method: "PUT",
        headers: {
          Authorization: `Bearer ${token}`,
          Accept: "application/json",
          "Content-Type": "application/json",
          "ngrok-skip-browser-warning": "1",
        },
        body: JSON.stringify(getSnapshot()),
        signal: AbortSignal.timeout(10000),
      },
    );
    if (!response.ok) {
      throw new Error(`activity snapshot request failed (${response.status})`);
    }
  }

  function queue() {
    if (!dirty) {
      return;
    }
    dirty = false;
    pending = pending.catch(() => undefined).then(persist).catch(() => {
      dirty = true;
    });
  }

  function schedule(immediate) {
    if (!enabled) {
      return;
    }
    dirty = true;
    if (immediate) {
      if (timer) {
        clearTimeout(timer);
        timer = undefined;
      }
      queue();
      return;
    }
    if (!timer) {
      timer = setTimeout(() => {
        timer = undefined;
        queue();
      }, 1000);
      timer.unref?.();
    }
  }

  async function flush() {
    if (timer) {
      clearTimeout(timer);
      timer = undefined;
    }
    queue();
    await pending;
    if (dirty) {
      queue();
      await pending;
    }
  }

  return { schedule, flush };
}

module.exports = {
  ACTIVITY_SCHEMA_VERSION,
  analysisActivityEnabled,
  createActivityStream,
  normalizeTerminalText,
  redactSensitiveText,
};
