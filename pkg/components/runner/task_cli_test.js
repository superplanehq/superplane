"use strict";

const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { main } = require("./task_cli");

async function readUpload(options) {
  const chunks = [];
  for await (const chunk of options.body) chunks.push(chunk);
  return Buffer.concat(chunks);
}

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
  ["task", "plan", "--file", "./PLAN.md"],
  ["task", "move", "--column", "done"],
];

test("superplane task artifact add-link attaches the url and label", async () => {
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
    ["task", "artifact", "add-link", "--url", "https://staging.example.com", "--name", "Staging", "--key", "staging"],
    env,
    fetchImpl,
    out,
    err,
  );
  assert.equal(code, 0);
  assert.equal(stdout, "https://staging.example.com\n");

  code = await main(
    ["task", "artifact", "add-link", "--url", "https://preview.example.com", "--title", "Preview"],
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

test("superplane task artifact add-markdown attaches a markdown file", async () => {
  const file = path.join(os.tmpdir(), `preview-login-${process.pid}.md`);
  fs.writeFileSync(file, "Email: demo@operately.dev\n\nPassword: preview-demo-1\n");
  const requests = [];
  let stdout = "";
  const code = await main(
    ["task", "artifact", "add-markdown", "--file", file, "--name", "Preview login", "--key", "preview-login"],
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

test("superplane task artifact add-markdown reports a missing file", async () => {
  let stderr = "";
  const code = await main(
    ["task", "artifact", "add-markdown", "--file", "missing-preview-login.md"],
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

test("superplane task artifact add-file uploads an image from the evidence directory", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "task-artifact-"));
  const evidence = path.join(taskDir, "evidence");
  fs.mkdirSync(evidence);
  const file = path.join(evidence, "login.png");
  fs.writeFileSync(file, "png-bytes");
  const requests = [];
  let stdout = "";
  const code = await main(
    ["task", "artifact", "add-file", "--file", file, "--name", "Login", "--key", "login"],
    { ...env, SUPERPLANE_TASK_DIR: taskDir },
    async (url, options) => {
      await readUpload(options);
      requests.push({ url, headers: options.headers });
      return {
        ok: true,
        status: 201,
        text: async () => JSON.stringify({ title: "Login" }),
      };
    },
    { write: (chunk) => { stdout += chunk; } },
    { write() {} },
  );
  fs.rmSync(taskDir, { recursive: true, force: true });
  assert.equal(code, 0);
  assert.equal(stdout, "Login\n");
  assert.equal(requests[0].url, "https://app.example/api/v1/runner/artifacts");
  assert.equal(requests[0].headers["Content-Type"], "image/png");
  assert.equal(requests[0].headers["X-SuperPlane-Artifact-Title"], "Login");
  assert.equal(requests[0].headers["X-SuperPlane-Artifact-Key"], "login");
});

test("superplane task artifact add-file uploads a video poster and the video", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "task-artifact-"));
  const evidence = path.join(taskDir, "evidence");
  fs.mkdirSync(evidence);
  const poster = path.join(evidence, "login.png");
  const video = path.join(evidence, "login.webm");
  fs.writeFileSync(poster, "png-bytes");
  fs.writeFileSync(video, "webm-bytes");
  const requests = [];
  let stdout = "";
  const code = await main(
    ["task", "artifact", "add-file", "--file", video, "--poster", poster, "--name", "Login", "--key", "login"],
    { ...env, SUPERPLANE_TASK_DIR: taskDir },
    async (_url, options) => {
      await readUpload(options);
      requests.push(options.headers);
      const title = decodeURIComponent(options.headers["X-SuperPlane-Artifact-Title"]);
      return {
        ok: true,
        status: 201,
        text: async () => JSON.stringify({ title }),
      };
    },
    { write: (chunk) => { stdout += chunk; } },
    { write() {} },
  );
  fs.rmSync(taskDir, { recursive: true, force: true });
  assert.equal(code, 0);
  assert.equal(stdout, "Login\n");
  assert.equal(requests.length, 2);
  assert.equal(requests[0]["Content-Type"], "image/png");
  assert.equal(requests[0]["X-SuperPlane-Artifact-Key"], "login-poster");
  assert.equal(requests[1]["Content-Type"], "video/webm");
  assert.equal(requests[1]["X-SuperPlane-Artifact-Key"], "login");
});

test("superplane task artifact add-file rejects a poster for an image", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "task-artifact-"));
  const evidence = path.join(taskDir, "evidence");
  fs.mkdirSync(evidence);
  const file = path.join(evidence, "login.png");
  const poster = path.join(evidence, "other.png");
  fs.writeFileSync(file, "png-bytes");
  fs.writeFileSync(poster, "png-bytes");
  let stderr = "";
  const code = await main(
    ["task", "artifact", "add-file", "--file", file, "--poster", poster],
    { ...env, SUPERPLANE_TASK_DIR: taskDir },
    async () => {
      throw new Error("fetch must not run");
    },
    { write() {} },
    { write: (chunk) => { stderr += chunk; } },
  );
  fs.rmSync(taskDir, { recursive: true, force: true });
  assert.equal(code, 1);
  assert.match(stderr, /poster is only supported for video/);
});

test("superplane task artifact add-file rejects a file outside the evidence directory", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "task-artifact-"));
  fs.mkdirSync(path.join(taskDir, "evidence"));
  const outside = path.join(taskDir, "login.png");
  fs.writeFileSync(outside, "png-bytes");
  let stderr = "";
  const code = await main(
    ["task", "artifact", "add-file", "--file", outside],
    { ...env, SUPERPLANE_TASK_DIR: taskDir },
    async () => {
      throw new Error("fetch must not run");
    },
    { write() {} },
    { write: (chunk) => { stderr += chunk; } },
  );
  fs.rmSync(taskDir, { recursive: true, force: true });
  assert.equal(code, 1);
  assert.match(stderr, /artifact path must be inside the task evidence directory/);
});

test("superplane task artifact add is not a command", () => {
  const result = spawnSync(process.execPath, [path.join(__dirname, "task_cli.js"), "task", "artifact", "add"], {
    encoding: "utf8",
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /add-link/);
  assert.match(result.stderr, /add-markdown/);
  assert.match(result.stderr, /add-file/);
  assert.doesNotMatch(result.stderr, /task artifact add --url/);
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

test("task help lists the artifact commands", async () => {
  let stdout = "";
  const code = await main(["task", "help"], env, fetch, { write: (chunk) => { stdout += chunk; } }, { write() {} });
  assert.equal(code, 0);
  assert.match(stdout, /superplane task artifact add-link/);
  assert.doesNotMatch(stdout, /suggest-owners/);
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
