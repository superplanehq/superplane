"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const { createClient } = require("./client");
const registry = require("./registry");

function recordingFetch(calls) {
  return async (url, init) => {
    calls.push({ url, method: init.method, body: init.body && JSON.parse(init.body) });
    return { ok: true, status: 200, text: async () => '{"id":"1","title":"PLAN.md"}' };
  };
}

const env = {
  SUPERPLANE_BASE_URL: "https://superplane.example",
  SUPERPLANE_RUNNER_TOKEN: "runner-token",
};

test("remove_artifact deletes by key or title", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls));

  await registry.findTool("remove_artifact").handler({ key: " plan " }, client);
  await registry.findTool("remove_artifact").handler({ title: " PLAN.md " }, client);

  assert.equal(calls[0].method, "DELETE");
  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/task/artifacts");
  assert.deepEqual(calls[0].body, { key: "plan" });
  assert.deepEqual(calls[1].body, { title: "PLAN.md" });
});

test("remove_artifact requires a key or a title", async () => {
  const client = createClient(env, recordingFetch([]));
  await assert.rejects(
    () => registry.findTool("remove_artifact").handler({}, client),
    /key or title is required/,
  );
});
