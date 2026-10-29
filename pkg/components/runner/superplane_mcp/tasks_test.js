"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const { createClient } = require("./client");
const registry = require("./registry");

function recordingFetch(calls, body = '{"tasks":[]}') {
  return async (url, init) => {
    calls.push({ url, method: init.method, body: init.body && JSON.parse(init.body) });
    return { ok: true, status: 200, text: async () => body };
  };
}

const env = {
  SUPERPLANE_BASE_URL: "https://superplane.example",
  SUPERPLANE_RUNNER_TOKEN: "runner-token",
};

test("list_tasks sends state and limit", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls));

  await registry.findTool("list_tasks").handler({ state: [" Open ", "draft"], limit: 5 }, client);

  assert.equal(calls[0].method, "GET");
  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/tasks?state=open&state=draft&limit=5");
});

test("create_task posts the title", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls, '{"id":"1","state":"draft"}'));

  await registry.findTool("create_task").handler({ title: " Ship it ", description: " Notes " }, client);

  assert.equal(calls[0].method, "POST");
  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/tasks");
  assert.deepEqual(calls[0].body, { title: "Ship it", description: "Notes" });
});

test("list_tasks rejects a limit above 100", async () => {
  const client = createClient(env, recordingFetch([]));
  await assert.rejects(
    () => registry.findTool("list_tasks").handler({ limit: 101 }, client),
    /limit must be an integer from 1 to 100/,
  );
});

test("create_task counts an emoji as one character", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls, '{"id":"1","state":"draft"}'));
  const title = `${"a".repeat(255)}😀`;

  await registry.findTool("create_task").handler({ title }, client);

  assert.equal(calls[0].body.title, title);
  await assert.rejects(
    () => registry.findTool("create_task").handler({ title: `${title}b` }, client),
    /title must be at most 256 characters/,
  );
});

test("create_task requires a title", async () => {
  const client = createClient(env, recordingFetch([]));
  await assert.rejects(() => registry.findTool("create_task").handler({}, client), /title is required/);
});

test("close_task posts the result and defaults to an empty body", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls, '{"state":"closed"}'));

  await registry.findTool("close_task").handler({ result: " Rejected " }, client);
  await registry.findTool("close_task").handler({}, client);

  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/task/close");
  assert.deepEqual(calls[0].body, { result: "rejected" });
  assert.deepEqual(calls[1].body, {});
});

test("close_task rejects an unknown result", async () => {
  const client = createClient(env, recordingFetch([]));
  await assert.rejects(
    () => registry.findTool("close_task").handler({ result: "archived" }, client),
    /result must be completed, rejected, or failed/,
  );
});
