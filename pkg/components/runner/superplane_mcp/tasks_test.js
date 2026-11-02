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

function call(name, input, client) {
  return registry.findTool(name).handler(input, client);
}

function ownerFetch(calls) {
  return async (url, init) => {
    calls.push({ url, method: init.method, headers: init.headers, body: init.body && JSON.parse(init.body) });
    return { ok: true, status: 200, text: async () => '{"status":"assigned"}' };
  };
}

const ownerEnv = {
  SUPERPLANE_BASE_URL: "https://superplane.example/",
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

test("suggest_task_owner posts normalized paths to the task route", async () => {
  const calls = [];
  const client = createClient(ownerEnv, ownerFetch(calls));
  const many = Array.from({ length: 12 }, (_, index) => `pkg/${index}`);

  await call("suggest_task_owner", { paths: [" pkg/payments ", "", ...many] }, client);
  await call("suggest_task_owner", {}, client);

  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/task/suggest-owners");
  assert.equal(calls[0].method, "POST");
  assert.equal(calls[0].headers.Authorization, "Bearer runner-token");
  assert.equal(calls[0].body.paths.length, 10);
  assert.equal(calls[0].body.paths[0], "pkg/payments");
  assert.deepEqual(calls[1].body, { paths: [] });
});

test("assign_task_owner puts the owner on the task route", async () => {
  const calls = [];
  const client = createClient(ownerEnv, ownerFetch(calls));

  await call("assign_task_owner", { user: "@mijat", reason: " You changed pkg/payments. " }, client);

  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/task/owner");
  assert.equal(calls[0].method, "PUT");
  assert.deepEqual(calls[0].body, { user: "@mijat", reason: "You changed pkg/payments." });
});

test("assign_task_owner can keep an existing owner", async () => {
  const calls = [];
  const client = createClient(ownerEnv, ownerFetch(calls));

  await call("assign_task_owner", { user: "mijat", only_if_unassigned: true }, client);

  assert.deepEqual(calls[0].body, { user: "mijat", only_if_unassigned: true });
});

test("assign_task_owner sends a call with no reason", async () => {
  const calls = [];
  const client = createClient(ownerEnv, ownerFetch(calls));

  await call("assign_task_owner", { user: "mijat" }, client);

  assert.deepEqual(calls[0].body, { user: "mijat" });
  assert.equal(Object.hasOwn(calls[0].body, "reason"), false);
});

test("assign_task_owner requires a user", async () => {
  const client = createClient(ownerEnv, ownerFetch([]));

  await assert.rejects(() => call("assign_task_owner", { reason: "Owns pkg." }, client), /user is required/);
});

test("the client requires the runner token", async () => {
  const client = createClient({ SUPERPLANE_BASE_URL: "https://superplane.example" }, ownerFetch([]));
  await assert.rejects(
    () => call("suggest_task_owner", {}, client),
    /SUPERPLANE_RUNNER_TOKEN/,
  );
});

test("the client returns the backend message on an HTTP error", async () => {
  const client = createClient(env, async () => ({
    ok: false,
    status: 403,
    text: async () => "This run cannot use this tool\n",
  }));
  await assert.rejects(
    () => call("suggest_task_owner", {}, client),
    (error) => error.status === 403 && error.message === "This run cannot use this tool",
  );
});

test("list_members reads the organization members", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls, '{"members":[]}'));

  await call("list_members", {}, client);

  assert.equal(calls[0].method, "GET");
  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/members");
});
