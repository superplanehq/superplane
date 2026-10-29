"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const { createClient } = require("./client");
const registry = require("./registry");

function recordingFetch(calls) {
  return async (url, init) => {
    calls.push({ url, method: init.method, headers: init.headers, body: init.body && JSON.parse(init.body) });
    return { ok: true, status: 200, text: async () => '{"status":"assigned"}' };
  };
}

const env = {
  SUPERPLANE_BASE_URL: "https://superplane.example/",
  SUPERPLANE_RUNNER_TOKEN: "runner-token",
};

test("suggest_owners posts normalized paths to the task route", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls));
  const many = Array.from({ length: 12 }, (_, index) => `pkg/${index}`);

  await registry.findTool("suggest_owners").handler({ paths: [" pkg/payments ", "", ...many] }, client);
  await registry.findTool("suggest_owners").handler({}, client);

  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/task/suggest-owners");
  assert.equal(calls[0].method, "POST");
  assert.equal(calls[0].headers.Authorization, "Bearer runner-token");
  assert.equal(calls[0].body.paths.length, 10);
  assert.equal(calls[0].body.paths[0], "pkg/payments");
  assert.deepEqual(calls[1].body, { paths: [] });
});

test("assign_owner puts the owner on the task route", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls));

  await registry.findTool("assign_owner").handler({ user: "@mijat", reason: " You changed pkg/payments. " }, client);

  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/task/owner");
  assert.equal(calls[0].method, "PUT");
  assert.deepEqual(calls[0].body, { user: "@mijat", reason: "You changed pkg/payments." });
});

test("assign_owner requires a user and a reason", async () => {
  const client = createClient(env, recordingFetch([]));
  const { handler } = registry.findTool("assign_owner");

  await assert.rejects(() => handler({ reason: "Owns pkg." }, client), /user is required/);
  await assert.rejects(() => handler({ user: "mijat" }, client), /reason is required/);
});

test("the client requires the runner token", async () => {
  const client = createClient({ SUPERPLANE_BASE_URL: "https://superplane.example" }, recordingFetch([]));
  await assert.rejects(
    () => registry.findTool("suggest_owners").handler({}, client),
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
    () => registry.findTool("suggest_owners").handler({}, client),
    (error) => error.status === 403 && error.message === "This run cannot use this tool",
  );
});
