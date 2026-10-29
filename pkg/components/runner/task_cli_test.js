"use strict";

const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
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

test("superplane task artifact add attaches a markdown file", async () => {
  const file = path.join(os.tmpdir(), `preview-login-${process.pid}.md`);
  fs.writeFileSync(file, "Email: demo@operately.dev\n\nPassword: preview-demo-1\n");
  const requests = [];
  let stdout = "";
  const code = await main(
    ["task", "artifact", "add", "--file", file, "--name", "Preview login", "--key", "preview-login"],
    env,
    async (url, options) => {
      requests.push({ url, body: JSON.parse(options.body) });
      return {
        ok: true,
        status: 201,
        text: async () => JSON.stringify({ title: "Preview login" }),
      };
    },
    { write: (chunk) => { stdout += chunk; } },
    { write() {} },
  );
  fs.unlinkSync(file);
  assert.equal(code, 0);
  assert.equal(stdout, "Preview login\n");
  assert.equal(requests[0].url, "https://app.example/api/v1/runner/artifacts/markdown");
  assert.deepEqual(requests[0].body, {
    body: "Email: demo@operately.dev\n\nPassword: preview-demo-1",
    title: "Preview login",
    key: "preview-login",
  });
});

test("superplane task artifact add rejects a url and a file together", async () => {
  let stderr = "";
  const code = await main(
    ["task", "artifact", "add", "--url", "https://example.com", "--file", "notes.md"],
    env,
    async () => {
      throw new Error("fetch must not run");
    },
    { write() {} },
    { write: (chunk) => { stderr += chunk; } },
  );
  assert.equal(code, 1);
  assert.match(stderr, /--url URL \| --file PATH/);
});

test("superplane task artifact add reports a missing file", async () => {
  let stderr = "";
  const code = await main(
    ["task", "artifact", "add", "--file", "missing-preview-login.md"],
    env,
    async () => {
      throw new Error("fetch must not run");
    },
    { write() {} },
    { write: (chunk) => { stderr += chunk; } },
  );
  assert.equal(code, 1);
  assert.match(stderr, /cannot read missing-preview-login.md/);
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
