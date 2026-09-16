"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");

const {
  ACTIVITY_SCHEMA_VERSION,
  createActivityStream,
  normalizeTerminalText,
  redactSensitiveText,
} = require("./activity_stream");

function analysisEnvironment() {
  return {
    SUPERPLANE_PLANNING_SESSION_KIND: "work_order_analysis",
    SUPERPLANE_PLANNING_SESSION_ID: "session-1",
  };
}

test("emits ordered records with stable activity metadata", () => {
  const records = [];
  let now = 1000;
  const stream = createActivityStream({
    provider: "codex",
    turn: 3,
    activityId: "activity-1",
    env: analysisEnvironment(),
    now: () => now,
    writeRecord: (record) => records.push(record),
  });

  stream.start();
  stream.startContent("reasoning", "reasoning-1");
  now = 1010;
  stream.appendContent("reasoning", "reasoning-1", "Inspecting files");
  stream.endContent("reasoning-1");
  stream.startTool({ id: "tool-a", kind: "bash", name: "Bash", input: "printf ok" });
  stream.startTool({ id: "tool-b", kind: "read", name: "Read", input: "README.md" });
  stream.appendToolOutput("tool-b", "read result", "stdout");
  stream.endTool("tool-b", { status: "passed" });
  stream.appendToolOutput("tool-a", "ok", "stdout");
  stream.endTool("tool-a", { status: "passed", exitCode: 0 });
  stream.end("passed");

  assert.deepEqual(
    records.map((record) => record.type),
    [
      "activity_start",
      "content_start",
      "line",
      "content_end",
      "tool_start",
      "tool_start",
      "line",
      "tool_end",
      "line",
      "tool_end",
      "activity_end",
    ],
  );
  assert.deepEqual(
    records.map((record) => record.sequence),
    records.map((_, index) => index + 1),
  );
  assert.ok(records.every((record) => record.schema_version === ACTIVITY_SCHEMA_VERSION));
  assert.ok(records.every((record) => record.activity_id === "activity-1"));
  assert.ok(records.every((record) => record.turn === 3));
  assert.equal(records[2].channel, "reasoning");
  assert.equal(records[6].tool_id, "tool-b");
  assert.equal(records[8].tool_id, "tool-a");
});

test("normalizes terminal control sequences and carriage-return progress", () => {
  assert.equal(normalizeTerminalText("\u001b[31mfailed\u001b[0m\rworking\rdone\n"), "done\n");
});

test("redacts credentials from activity text", () => {
  const env = { OPENAI_API_KEY: "secret-value-123", OTHER: "safe" };
  const input = "Bearer abc.def.ghi secret-value-123 https://example.test/?token=visible";
  const output = redactSensitiveText(input, env);

  assert.equal(output.includes("secret-value-123"), false);
  assert.equal(output.includes("abc.def.ghi"), false);
  assert.equal(output.includes("token=visible"), false);
  assert.match(output, /\[REDACTED\]/);
});

test("redacts the planning session runner token from activity text", () => {
  const token = "eyJhbGciOiJIUzI1NiJ9.eyJwdXJwb3NlIjoicGxhbm5pbmdfc2Vzc2lvbiJ9.signature";
  const output = redactSensitiveText(`SUPERPLANE_RUN_TOKEN=${token}\n${token}`, {
    SUPERPLANE_RUN_TOKEN: token,
  });

  assert.equal(output.includes(token), false);
  assert.equal(output, "SUPERPLANE_RUN_TOKEN=[REDACTED]\n[REDACTED]");
});

test("keeps the first and last output when a tool exceeds its limit", () => {
  const records = [];
  const stream = createActivityStream({
    provider: "claude",
    activityId: "activity-2",
    env: analysisEnvironment(),
    writeRecord: (record) => records.push(record),
    limits: { toolOutputBytes: 20, toolOutputHeadBytes: 8 },
  });

  stream.startTool({ id: "tool-1", kind: "bash", input: "run" });
  stream.appendToolOutput("tool-1", "abcdefghij");
  stream.appendToolOutput("tool-1", "klmnopqrstuv");
  stream.endTool("tool-1", { status: "failed" });

  const snapshot = stream.snapshot();
  const tool = snapshot.items.find((item) => item.type === "tool");
  assert.equal(tool.truncated, true);
  assert.equal(tool.output.startsWith("abcdefgh"), true);
  assert.equal(tool.output.endsWith("mnopqrstuv"), true);
});

test("does not emit activity records outside analysis sessions", () => {
  const records = [];
  const stream = createActivityStream({ provider: "codex", env: {}, writeRecord: (record) => records.push(record) });

  stream.start();
  stream.startContent("assistant", "answer");
  stream.appendContent("assistant", "answer", "Hello");
  stream.end("passed");

  assert.deepEqual(records, []);
});

test("keeps persisted snapshots within the configured activity limit", () => {
  const stream = createActivityStream({
    provider: "codex",
    activityId: "bounded-activity",
    env: analysisEnvironment(),
    writeRecord: () => undefined,
    limits: { snapshotBytes: 1024 },
  });

  for (let index = 0; index < 20; index += 1) {
    const id = `tool-${index}`;
    stream.startTool({ id, kind: "bash", input: `printf ${"x".repeat(200)}` });
    stream.appendToolOutput(id, "y".repeat(1000));
    stream.endTool(id, { status: index === 19 ? "failed" : "passed" });
  }

  const snapshot = stream.snapshot();
  assert.ok(Buffer.byteLength(JSON.stringify(snapshot), "utf8") <= 1024);
  assert.equal(snapshot.truncated, true);
  assert.ok(snapshot.items.some((item) => item.code === "snapshot_truncated"));
});

test("preserves every supported terminal tool status", () => {
  const statuses = ["passed", "failed", "cancelled", "timed_out", "interrupted"];
  const stream = createActivityStream({
    provider: "codex",
    activityId: "terminal-states",
    env: analysisEnvironment(),
    writeRecord: () => undefined,
  });

  for (const status of statuses) {
    stream.startTool({ id: status, kind: "bash", input: status });
    stream.endTool(status, { status, exitCode: status === "failed" ? 7 : undefined, signal: "SIGTERM" });
  }

  assert.deepEqual(
    stream.snapshot().items.map((item) => item.status),
    statuses,
  );
  assert.equal(stream.snapshot().items[1].exit_code, 7);
  assert.equal(stream.snapshot().items[1].signal, "SIGTERM");
});
