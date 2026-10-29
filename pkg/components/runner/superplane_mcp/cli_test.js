"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const cli = require("./cli");
const registry = require("./registry");

const env = {
  SUPERPLANE_BASE_URL: "https://superplane.example",
  SUPERPLANE_RUNNER_TOKEN: "runner-token",
};

function output() {
  let text = "";
  return { write: (chunk) => (text += chunk), text: () => text };
}

function recordingFetch(calls) {
  return async (url, init) => {
    calls.push({ url, method: init.method, body: init.body && JSON.parse(init.body) });
    return { ok: true, status: 200, text: async () => '{"status":"ok"}' };
  };
}

test("every global tool has a CLI command", () => {
  for (const tool of registry.allTools()) {
    assert.ok(tool.cli, `${tool.definition.name} has no CLI command`);
    assert.ok(tool.cli.words.length > 0, `${tool.definition.name} has no CLI words`);
    assert.match(tool.cli.usage, /^superplane /);
    assert.equal(typeof tool.cli.toInput, "function");
  }
});

test("suggest-owners sends the same request as the MCP tool", async () => {
  const calls = [];
  const stdout = output();

  const code = await cli.main(
    ["task", "suggest-owners", "--path", "pkg/a", "--path", "pkg/b"],
    env,
    recordingFetch(calls),
    stdout,
    output(),
  );

  assert.equal(code, 0);
  assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/task/suggest-owners");
  assert.deepEqual(calls[0].body, { paths: ["pkg/a", "pkg/b"] });
  assert.equal(stdout.text(), '{"status":"ok"}\n');
});

test("assign sends the owner and accepts --assignee", async () => {
  const calls = [];

  await cli.main(["task", "assign", "--user", "u1", "--reason", "Owns pkg."], env, recordingFetch(calls), output(), output());
  await cli.main(["task", "assign", "--assignee", "u2", "--reason", "Owns web."], env, recordingFetch(calls), output(), output());

  assert.equal(calls[0].method, "PUT");
  assert.deepEqual(calls[0].body, { user: "u1", reason: "Owns pkg." });
  assert.deepEqual(calls[1].body, { user: "u2", reason: "Owns web." });
});

test("a missing required flag prints the usage and fails", async () => {
  const calls = [];
  const stderr = output();

  const code = await cli.main(["task", "assign", "--user", "u1"], env, recordingFetch(calls), output(), stderr);

  assert.equal(code, 1);
  assert.equal(calls.length, 0);
  assert.match(stderr.text(), /usage: superplane task assign --user USER --reason TEXT/);
});

test("task help lists every global command", async () => {
  const stdout = output();
  assert.equal(await cli.main(["task", "help"], env, recordingFetch([]), stdout, output()), 0);
  for (const tool of registry.allTools()) {
    assert.ok(stdout.text().includes(tool.cli.usage));
  }
});
