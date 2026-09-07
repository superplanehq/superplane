"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const {
  FOLLOW_UP_CMD_INDEX_BASE,
  interpretWaitResponse,
  nextAction,
  runLoop,
  runPromptFile,
  safeWaitRequest,
} = require("./follow_up_loop.js");

const CLOUDFLARE_502 = {
  error: "An error occurred with your request. Please try again.",
  retryable: true,
  retry_after: 60,
};

test("waits while SuperPlane has no event", () => {
  assert.deepEqual(nextAction({ status: "pending" }), { type: "wait" });
  assert.deepEqual(nextAction({}), { type: "wait" });
});

test("exits when the session ends", () => {
  assert.deepEqual(nextAction({ status: "ended" }), { type: "exit", code: 0 });
});

test("turns a user message into the next prompt", () => {
  assert.deepEqual(nextAction({ status: "message", text: " Add a Size field " }), {
    type: "prompt",
    text: "Add a Size field",
  });
});

test("ignores an empty user message", () => {
  assert.deepEqual(nextAction({ status: "message", text: "   " }), { type: "wait" });
});

test("asks Claude to acknowledge create or skip, not to draft the next task", () => {
  const created = nextAction({ status: "created", work_order_key: "NEWWO-12" });
  assert.equal(created.type, "prompt");
  assert.match(created.text, /NEWWO-12/);
  assert.match(created.text, /Acknowledge/i);
  assert.match(created.text, /Do not call propose_draft/);
  assert.doesNotMatch(created.text, /Propose the next/);
  const skipped = nextAction({ status: "skipped" });
  assert.equal(skipped.type, "prompt");
  assert.match(skipped.text, /skipped/i);
  assert.match(skipped.text, /Acknowledge/i);
  assert.match(skipped.text, /Do not call propose_draft/);
});

test("runLoop runs the user prompt then exits on ended", async () => {
  const prompts = [];
  const results = [{ status: "pending" }, { status: "message", text: "Add color" }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async (text) => {
      prompts.push(text);
      return 0;
    },
    sleep: async () => {},
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(prompts, ["Add color"]);
});

test("runLoop prompts after create or skip", async () => {
  const prompts = [];
  const results = [
    { status: "created", work_order_key: "NEWWO-12" },
    { status: "skipped" },
    { status: "ended" },
  ];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async (text) => {
      prompts.push(text);
      return 0;
    },
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.equal(prompts.length, 2);
  assert.match(prompts[0], /NEWWO-12/);
  assert.match(prompts[1], /skipped/i);
});

test("interpretWaitResponse treats a Cloudflare 502 as idle pending", () => {
  const got = interpretWaitResponse(502, CLOUDFLARE_502);
  assert.deepEqual(got, { status: "pending" });
});

test("interpretWaitResponse retries 503, 504, and 429 as idle pending", () => {
  assert.deepEqual(interpretWaitResponse(503, {}), { status: "pending" });
  assert.deepEqual(interpretWaitResponse(504, {}), { status: "pending" });
  assert.deepEqual(interpretWaitResponse(429, { retry_after: 12 }), { status: "pending" });
});

test("interpretWaitResponse treats a Cloudflare error body as idle pending", () => {
  const got = interpretWaitResponse(500, { cloudflare_error: true, retry_after: 30 });
  assert.deepEqual(got, { status: "pending" });
});

test("interpretWaitResponse ignores retry_after on a transient wait", () => {
  const got = interpretWaitResponse(502, { retryable: true, retry_after: 120 });
  assert.deepEqual(got, { status: "pending" });
});

test("interpretWaitResponse ends on 409", () => {
  assert.deepEqual(interpretWaitResponse(409, { message: "conflict" }), { status: "ended" });
});

test("interpretWaitResponse throws on 401", () => {
  assert.throws(() => interpretWaitResponse(401, { message: "unauthorized" }), /unauthorized/);
});

test("runLoop sleeps 1s with no log after a Cloudflare 502, then runs the next message", async () => {
  const sleeps = [];
  const logs = [];
  const prompts = [];
  const results = [interpretWaitResponse(502, CLOUDFLARE_502), { status: "message", text: "hello" }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async (text) => {
      prompts.push(text);
      return 0;
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.deepEqual(logs, []);
  assert.deepEqual(prompts, ["hello"]);
});

test("runLoop sleeps 1s with no log when a transient wait has no retry_after", async () => {
  const sleeps = [];
  const logs = [];
  const results = [{ status: "pending", transient: true }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.deepEqual(logs, []);
});

test("runLoop ignores a large retry_after and stays silent", async () => {
  const sleeps = [];
  const logs = [];
  const results = [{ status: "pending", retry_after: 99999, transient: true }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.deepEqual(logs, []);
});

test("runLoop backs off silently on idle pending and empty-message waits", async () => {
  const sleeps = [];
  const logs = [];
  const results = [
    { status: "pending" },
    { status: "message", text: "   " },
    { status: "ended" },
  ];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000, 1000]);
  assert.deepEqual(logs, []);
});

test("safeWaitRequest treats a fetch throw as pending", async () => {
  const got = await safeWaitRequest(async () => {
    throw new TypeError("fetch failed");
  });
  assert.deepEqual(got, { status: "pending" });
});

test("safeWaitRequest treats an abort as pending", async () => {
  const got = await safeWaitRequest(async () => {
    const err = new Error("This operation was aborted");
    err.name = "AbortError";
    throw err;
  });
  assert.deepEqual(got, { status: "pending" });
});

test("safeWaitRequest keeps a delivered user message", async () => {
  const got = await safeWaitRequest(async () => ({
    status: 200,
    text: async () => JSON.stringify({ status: "message", text: "hello" }),
  }));
  assert.deepEqual(got, { status: "message", text: "hello" });
});

test("safeWaitRequest still throws on 401", async () => {
  await assert.rejects(
    () =>
      safeWaitRequest(async () => ({
        status: 401,
        text: async () => JSON.stringify({ message: "unauthorized" }),
      })),
    /unauthorized/,
  );
});

test("runLoop runs a user message after a dropped wait", async () => {
  const prompts = [];
  const logs = [];
  const results = [
    await safeWaitRequest(async () => {
      throw new TypeError("fetch failed");
    }),
    { status: "message", text: "hello" },
    { status: "ended" },
  ];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async (text) => {
      prompts.push(text);
      return 0;
    },
    sleep: async () => {},
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(prompts, ["hello"]);
  assert.deepEqual(logs, []);
});

test("runLoop stays alive when waitOnce returns pending after a fetch throw", async () => {
  const sleeps = [];
  const logs = [];
  const results = [await safeWaitRequest(async () => { throw new TypeError("fetch failed"); }), { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.deepEqual(logs, []);
});

test("runLoop emits cmd_start then cmd_end for each follow-up prompt", async () => {
  const records = [];
  const nowValues = [5_000, 5_250, 6_000, 6_400];
  const results = [
    { status: "message", text: "Add color" },
    { status: "created", work_order_key: "NEWWO-12" },
    { status: "ended" },
  ];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => 0,
    writeLiveLogRecord: (rec) => records.push(rec),
    now: () => nowValues.shift(),
  });
  assert.equal(code, 0);
  assert.deepEqual(records, [
    {
      type: "cmd_start",
      index: FOLLOW_UP_CMD_INDEX_BASE,
      text: "Add color",
      kind: "prompt",
      preview: "Add color",
      started_at: 5_000,
    },
    {
      type: "cmd_end",
      index: FOLLOW_UP_CMD_INDEX_BASE,
      status: "passed",
      duration_ms: 250,
    },
    {
      type: "cmd_start",
      index: FOLLOW_UP_CMD_INDEX_BASE + 1,
      text: nextAction({ status: "created", work_order_key: "NEWWO-12" }).text,
      kind: "prompt",
      preview: nextAction({ status: "created", work_order_key: "NEWWO-12" }).text,
      started_at: 6_000,
    },
    {
      type: "cmd_end",
      index: FOLLOW_UP_CMD_INDEX_BASE + 1,
      status: "passed",
      duration_ms: 400,
    },
  ]);
});

test("runLoop marks a failed follow-up cmd_end and keeps waiting", async () => {
  const records = [];
  const results = [{ status: "message", text: "hello" }, { status: "ended" }];
  const logs = [];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => 2,
    writeLiveLogRecord: (rec) => records.push(rec),
    now: () => 1_000,
    log: (msg) => logs.push(msg),
  });
  assert.equal(code, 0);
  assert.equal(records[1].type, "cmd_end");
  assert.equal(records[1].status, "failed");
  assert.match(logs[0], /follow-up prompt failed with exit 2/);
});

test("runPromptFile forwards extra argv to run.js", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-loop-"));
  const argvFile = path.join(taskDir, "argv.json");
  fs.writeFileSync(
    path.join(taskDir, "run.js"),
    `require("fs").writeFileSync(${JSON.stringify(argvFile)}, JSON.stringify(process.argv.slice(2)));\n`,
  );
  const promptFile = path.join(taskDir, "prompt.txt");
  fs.writeFileSync(promptFile, "hello\n");

  const code = await runPromptFile(taskDir, promptFile, "openai/gpt-4.1", ["64"]);
  assert.equal(code, 0);
  const argv = JSON.parse(fs.readFileSync(argvFile, "utf8"));
  assert.deepEqual(argv, [promptFile, "openai/gpt-4.1", "64"]);
});
