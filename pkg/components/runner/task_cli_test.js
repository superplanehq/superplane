"use strict";

const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const path = require("node:path");
const test = require("node:test");

const { main } = require("./task_cli");

const env = {
  SUPERPLANE_BASE_URL: "https://app.example",
  SUPERPLANE_ARTIFACT_TOKEN: "token",
};

const placeholders = [
  ["task", "artifact", "remove", "--name", "Staging"],
  ["task", "check", "add", "--name", "Tests pass"],
  ["task", "check", "remove", "--name", "Tests pass"],
  ["task", "create", "--title", "Slow query", "--description", "The daily report query."],
  ["task", "close"],
  ["task", "list", "--state", "open", "--column", "verify"],
  ["task", "assign", "--assignee", "alice@example.com"],
  ["task", "plan", "--file", "./PLAN.md"],
  ["task", "move", "--column", "done"],
];

test("superplane task artifact add attaches the url and label", async () => {
  const requests = [];
  const fetchImpl = async (_url, options) => {
    const body = JSON.parse(options.body);
    requests.push(body);
    return {
      ok: true,
      status: 201,
      text: async () => JSON.stringify({ url: body.url }),
    };
  };
  let stdout = "";
  const out = { write: (chunk) => { stdout += chunk; } };
  const err = { write() {} };

  let code = await main(
    ["task", "artifact", "add", "--url", "https://staging.example.com", "--name", "Staging", "--key", "staging"],
    env,
    fetchImpl,
    out,
    err,
  );
  assert.equal(code, 0);
  assert.equal(stdout, "https://staging.example.com\n");

  code = await main(
    ["task", "artifact", "add", "--url", "https://preview.example.com", "--title", "Preview"],
    env,
    fetchImpl,
    out,
    err,
  );
  assert.equal(code, 0);
  assert.deepEqual(requests, [
    { url: "https://staging.example.com", title: "Staging", key: "staging" },
    { url: "https://preview.example.com", title: "Preview", key: "" },
  ]);
});

test("superplane task artifact add requires a url", () => {
  const result = spawnSync(process.execPath, [path.join(__dirname, "task_cli.js"), "task", "artifact", "add"], {
    encoding: "utf8",
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /url is required/);
});

test("other task commands are recognized and do no work", async () => {
  for (const argv of placeholders) {
    let stderr = "";
    const code = await main(
      argv,
      env,
      async () => {
        throw new Error("fetch must not run");
      },
      { write() {} },
      { write: (chunk) => { stderr += chunk; } },
    );
    const flagAt = argv.findIndex((arg) => arg.startsWith("--"));
    const command = argv.slice(0, flagAt === -1 ? argv.length : flagAt).join(" ");
    assert.equal(code, 1, argv.join(" "));
    assert.match(stderr, new RegExp(`superplane ${command} is not available yet`));
  }
});

test("a placeholder with a missing flag does not claim to run", async () => {
  let stderr = "";
  const code = await main(
    ["task", "check", "add"],
    env,
    async () => {
      throw new Error("fetch must not run");
    },
    { write() {} },
    { write: (chunk) => { stderr += chunk; } },
  );
  assert.equal(code, 1);
  assert.match(stderr, /usage: superplane task check add --name NAME/);
  assert.doesNotMatch(stderr, /not available yet/);
});
