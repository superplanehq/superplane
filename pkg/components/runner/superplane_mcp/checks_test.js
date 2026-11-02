"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const registry = require("./registry");

function call(name, input, client) {
  return registry.findTool(name).handler(input, client);
}

function recordingClient() {
  const requests = [];
  return {
    requests,
    request: async (method, path, body) => {
      requests.push({ method, path, body });
      return { checks: [] };
    },
  };
}

const CHECK = { key: "api-latency", name: "API latency", score: 2, max_score: 5, level: "caution", reason: "p99 grew." };

test("list_checks reads the checks of a task", async () => {
  const client = recordingClient();

  await call("list_checks", { task: " SP-12 " }, client);

  assert.deepEqual(client.requests, [{ method: "GET", path: "/api/v1/runner/checks?task=SP-12", body: undefined }]);
  await assert.rejects(call("list_checks", {}, client), /task is required/);
});

test("add_checks posts normalized checks", async () => {
  const client = recordingClient();

  await call("add_checks", { checks: [{ ...CHECK, key: " api-latency ", score: "2", level: "Caution" }] }, client);

  assert.deepEqual(client.requests, [{ method: "POST", path: "/api/v1/runner/task/checks", body: { checks: [CHECK] } }]);
});

test("add_checks rejects an invalid check before it calls the backend", async () => {
  const client = recordingClient();
  const cases = [
    [{}, /checks is required/],
    [{ checks: [] }, /checks is required/],
    [{ checks: Array.from({ length: 21 }, (_, index) => ({ ...CHECK, key: `k${index}` })) }, /at most 20 items/],
    [{ checks: [{ ...CHECK, key: "" }] }, /checks\[0\]\.key is required/],
    [{ checks: [{ ...CHECK, name: "n".repeat(256) }] }, /checks\[0\]\.name must be at most 255 bytes/],
    [{ checks: [{ ...CHECK, score: 6 }] }, /checks\[0\]\.score must be from 0 to max_score/],
    [{ checks: [{ ...CHECK, max_score: 0, score: 0 }] }, /checks\[0\]\.max_score must be greater than 0/],
    [{ checks: [{ ...CHECK, score: "high" }] }, /checks\[0\]\.score must be a number/],
    [{ checks: [{ ...CHECK, level: "warning" }] }, /checks\[0\]\.level must be positive, caution, or critical/],
    [{ checks: [{ ...CHECK, reason: " " }] }, /checks\[0\]\.reason is required/],
    [{ checks: [CHECK, CHECK] }, /checks\[1\]\.key repeats an earlier key/],
  ];

  for (const [input, error] of cases) {
    await assert.rejects(call("add_checks", input, client), error);
  }
  assert.deepEqual(client.requests, []);
});

test("remove_checks deletes unique keys", async () => {
  const client = recordingClient();

  await call("remove_checks", { keys: [" api-latency ", "api-latency", "coverage"] }, client);

  assert.deepEqual(client.requests, [
    { method: "DELETE", path: "/api/v1/runner/task/checks", body: { keys: ["api-latency", "coverage"] } },
  ]);
  await assert.rejects(call("remove_checks", { keys: [] }, client), /keys is required/);
});
