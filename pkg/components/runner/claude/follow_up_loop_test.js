"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { nextAction, runLoop } = require("./follow_up_loop.js");

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

test("tells Claude when the user created or skipped a draft", () => {
  const created = nextAction({ status: "created", work_order_key: "NEWWO-12" });
  assert.equal(created.type, "prompt");
  assert.match(created.text, /NEWWO-12/);
  const skipped = nextAction({ status: "skipped" });
  assert.equal(skipped.type, "prompt");
  assert.match(skipped.text, /skipped/i);
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
  });
  assert.equal(code, 0);
  assert.deepEqual(prompts, ["Add color"]);
});
