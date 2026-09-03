"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { interpretWaitResponse, nextAction, runLoop } = require("./follow_up_loop.js");

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
  });
  assert.equal(code, 0);
  assert.equal(prompts.length, 2);
  assert.match(prompts[0], /NEWWO-12/);
  assert.match(prompts[1], /skipped/i);
});

test("interpretWaitResponse retries a 502 with retryable body", () => {
  const got = interpretWaitResponse(502, {
    error: "An error occurred with your request. Please try again.",
    retryable: true,
    retry_after: 60,
  });
  assert.deepEqual(got, { status: "pending", retry_after: 60, transient: true });
});

test("interpretWaitResponse retries 503, 504, and 429", () => {
  assert.equal(interpretWaitResponse(503, {}).status, "pending");
  assert.equal(interpretWaitResponse(504, {}).status, "pending");
  assert.equal(interpretWaitResponse(429, { retry_after: 12 }).retry_after, 12);
});

test("interpretWaitResponse retries a Cloudflare error body", () => {
  const got = interpretWaitResponse(500, { cloudflare_error: true, retry_after: 30 });
  assert.deepEqual(got, { status: "pending", retry_after: 30, transient: true });
});

test("interpretWaitResponse caps retry_after at 60 seconds", () => {
  const got = interpretWaitResponse(502, { retryable: true, retry_after: 120 });
  assert.equal(got.retry_after, 60);
});

test("interpretWaitResponse ends on 409", () => {
  assert.deepEqual(interpretWaitResponse(409, { message: "conflict" }), { status: "ended" });
});

test("interpretWaitResponse throws on 401", () => {
  assert.throws(() => interpretWaitResponse(401, { message: "unauthorized" }), /unauthorized/);
});

test("runLoop waits again after a transient wait, then exits 0", async () => {
  const sleeps = [];
  const results = [
    { status: "pending", retry_after: 60 },
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
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [60000]);
});

test("runLoop backs off with the default floor when a transient wait has no retry_after", async () => {
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
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.match(logs[0], /planning wait hit a transient error; retrying in 1s/);
});

test("runLoop clamps a large retry_after to the 60s cap", async () => {
  const sleeps = [];
  const results = [{ status: "pending", retry_after: 99999, transient: true }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [60000]);
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
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000, 1000]);
  assert.deepEqual(logs, []);
});
