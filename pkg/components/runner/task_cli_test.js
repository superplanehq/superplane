"use strict";

const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { createClient } = require("./superplane_mcp/client");
const registry = require("./superplane_mcp/registry");
const { main } = require("./task_cli");

async function recordScreenshot(file, taskEnv) {
  await registry.findTool("inspect_screenshot").handler(
    { path: file },
    createClient(taskEnv, async () => {
      throw new Error("fetch must not run");
    }),
  );
}

async function readUpload(options) {
  const chunks = [];
  for await (const chunk of options.body) chunks.push(chunk);
  return Buffer.concat(chunks);
}

const env = {
  SUPERPLANE_BASE_URL: "https://app.example",
  SUPERPLANE_RUNNER_TOKEN: "token",
};

const placeholders = [
  ["task", "check", "add", "--name", "Tests pass"],
  ["task", "check", "remove", "--name", "Tests pass"],
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
  assert.equal(requests[0].url, "https://app.example/api/v1/runner/task/markdown");
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
  const taskEnv = { ...env, SUPERPLANE_TASK_DIR: taskDir };
  let blocked = "";
  const blockedCode = await main(
    ["task", "artifact", "add-file", "--file", file, "--name", "Login"],
    taskEnv,
    async () => {
      throw new Error("fetch must not run");
    },
    { write() {} },
    { write: (chunk) => { blocked += chunk; } },
  );
  assert.equal(blockedCode, 1);
  assert.match(blocked, /inspect_screenshot/);

  let inspected = "";
  const inspectCode = await main(
    ["task", "evidence", "inspect", "--file", file],
    taskEnv,
    async () => {
      throw new Error("fetch must not run");
    },
    { write: (chunk) => { inspected += chunk; } },
    { write() {} },
  );
  assert.equal(inspectCode, 0);
  assert.equal(JSON.parse(inspected).filename, "login.png");
  assert.equal(fs.existsSync(path.join(taskDir, "visual-evidence-inspections.json")), false);

  const stillBlocked = await main(
    ["task", "artifact", "add-file", "--file", file, "--name", "Login"],
    taskEnv,
    async () => {
      throw new Error("fetch must not run");
    },
    { write() {} },
    { write() {} },
  );
  assert.equal(stillBlocked, 1);
  await recordScreenshot(file, taskEnv);

  const requests = [];
  let stdout = "";
  const code = await main(
    ["task", "artifact", "add-file", "--file", file, "--name", "Login"],
    taskEnv,
    async (url, options) => {
      await readUpload(options);
      requests.push({ url, headers: options.headers });
      return {
        ok: true,
        status: 201,
        text: async () => JSON.stringify({ title: "Login", file_id: "file-1" }),
      };
    },
    { write: (chunk) => { stdout += chunk; } },
    { write() {} },
  );
  fs.rmSync(taskDir, { recursive: true, force: true });
  assert.equal(code, 0);
  assert.equal(stdout, "Login\n");
  assert.equal(requests[0].url, "https://app.example/api/v1/runner/task/artifacts");
  assert.equal(requests[0].headers["Content-Type"], "image/png");
  assert.equal(requests[0].headers["X-SuperPlane-Artifact-Title"], "Login");
  assert.equal(requests[0].headers.Authorization, "Bearer token");
  assert.equal(requests[0].headers["X-SuperPlane-Artifact-Key"], undefined);
});

test("superplane task artifact add-file uploads a video poster and the video", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "task-artifact-"));
  const evidence = path.join(taskDir, "evidence");
  fs.mkdirSync(evidence);
  const poster = path.join(evidence, "login.png");
  const video = path.join(evidence, "login.webm");
  fs.writeFileSync(poster, "png-bytes");
  fs.writeFileSync(video, "webm-bytes");
  const taskEnv = { ...env, SUPERPLANE_TASK_DIR: taskDir };
  await recordScreenshot(poster, taskEnv);

  const requests = [];
  let stdout = "";
  const code = await main(
    ["task", "artifact", "add-file", "--file", video, "--poster", poster, "--name", "Login"],
    taskEnv,
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
  assert.equal(requests[0]["X-SuperPlane-Artifact-Title"], "Login%20poster");
  assert.equal(requests[1]["Content-Type"], "video/webm");
  assert.equal(requests[1]["X-SuperPlane-Artifact-Title"], "Login");
  assert.equal(requests[1]["X-SuperPlane-Artifact-Key"], undefined);
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
  assert.match(stderr, /posterPath is only supported for video artifacts/);
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

test("task help lists a command for every global tool", async () => {
  let stdout = "";
  const code = await main(["task", "help"], env, fetch, { write: (chunk) => { stdout += chunk; } }, { write() {} });
  assert.equal(code, 0);
  for (const line of [
    "superplane task owner suggest",
    "superplane task owner assign",
    "superplane task list",
    "superplane task create",
    "superplane task close",
    "superplane task plan",
    "superplane task attachment inspect",
    "superplane task evidence inspect",
    "superplane task evidence unavailable",
    "superplane task artifact add-link",
    "superplane task artifact add-markdown",
    "superplane task artifact add-file",
    "superplane task artifact remove",
    "superplane task check report",
  ]) {
    assert.match(stdout, new RegExp(line));
  }
});

test("each global tool command calls that tool", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "task-cli-"));
  const specFile = path.join(taskDir, "spec.md");
  const scoresFile = path.join(taskDir, "scores.json");
  const attemptsFile = path.join(taskDir, "attempts.json");
  const attachment = path.join(taskDir, "attachments", "shot.png");
  fs.writeFileSync(specFile, "# Spec\n");
  fs.writeFileSync(
    scoresFile,
    JSON.stringify({
      clarity: { score: 2, summary: "Clear enough." },
      complexity: { score: 2, summary: "Small change." },
      verifiability: { score: 2, summary: "A test covers it." },
    }),
  );
  fs.writeFileSync(
    attemptsFile,
    JSON.stringify([
      { type: "preview", command: "open preview", outcome: "no preview" },
      { type: "playwright", command: "npx playwright", outcome: "no browser" },
    ]),
  );
  fs.mkdirSync(path.dirname(attachment), { recursive: true });
  fs.writeFileSync(
    attachment,
    Buffer.from(
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
      "base64",
    ),
  );
  const taskEnv = {
    ...env,
    SUPERPLANE_TASK_DIR: taskDir,
    SUPERPLANE_ANALYSIS_SPEC_FILE: path.join(taskDir, "written-spec.md"),
  };
  const calls = [];
  const fetchImpl = async (url, options) => {
    calls.push({
      url,
      method: options.method,
      authorization: options.headers.Authorization,
      body: options.body ? JSON.parse(options.body) : undefined,
    });
    return { ok: true, status: 200, text: async () => "{}" };
  };
  const cases = [
    {
      argv: ["task", "owner", "suggest", "--path", "pkg/a", "--path", "pkg/b"],
      method: "POST",
      url: "https://app.example/api/v1/runner/task/suggest-owners",
      body: { paths: ["pkg/a", "pkg/b"] },
    },
    {
      argv: ["task", "owner", "assign", "--user", "ada", "--reason", "Owns payments.", "--only-if-unassigned"],
      method: "PUT",
      url: "https://app.example/api/v1/runner/task/owner",
      body: { user: "ada", reason: "Owns payments.", only_if_unassigned: true },
    },
    {
      argv: ["task", "list", "--state", "open", "--state", "draft", "--limit", "5"],
      method: "GET",
      url: "https://app.example/api/v1/runner/tasks?state=open&state=draft&limit=5",
    },
    {
      argv: ["task", "create", "--title", "Slow query", "--description", "The daily report query."],
      method: "POST",
      url: "https://app.example/api/v1/runner/tasks",
      body: { title: "Slow query", description: "The daily report query." },
    },
    {
      argv: ["task", "close", "--result", "rejected"],
      method: "POST",
      url: "https://app.example/api/v1/runner/task/close",
      body: { result: "rejected" },
    },
    {
      argv: ["task", "plan", "--file", specFile, "--scores", scoresFile],
      method: "PUT",
      url: "https://app.example/api/v1/runner/task/plan",
      body: {
        spec: "# Spec",
        scores: {
          clarity: { score: 2, summary: "Clear enough." },
          complexity: { score: 2, summary: "Small change." },
          verifiability: { score: 2, summary: "A test covers it." },
        },
      },
    },
    {
      argv: ["task", "artifact", "remove", "--name", "Staging"],
      method: "DELETE",
      url: "https://app.example/api/v1/runner/task/artifacts",
      body: { title: "Staging" },
    },
    {
      argv: ["task", "check", "report", "--check", "Risk", "--score", "4", "--summary", "The change is small."],
      method: "POST",
      url: "https://app.example/api/v1/runner/task/merge-checks",
      body: { check: "risk", score: 4, summary: "The change is small." },
    },
  ];

  for (const item of cases) {
    calls.length = 0;
    let stderr = "";
    const code = await main(item.argv, taskEnv, fetchImpl, { write() {} }, { write: (chunk) => { stderr += chunk; } });
    assert.equal(code, 0, `${item.argv.join(" ")}\n${stderr}`);
    assert.equal(calls.length, 1, item.argv.join(" "));
    assert.equal(calls[0].method, item.method);
    assert.equal(calls[0].url, item.url);
    assert.equal(calls[0].authorization, "Bearer token");
    assert.deepEqual(calls[0].body, item.body);
  }

  calls.length = 0;
  let local = "";
  const unavailable = await main(
    ["task", "evidence", "unavailable", "--reason", "No browser.", "--attempts", attemptsFile],
    taskEnv,
    fetchImpl,
    { write: (chunk) => { local += chunk; } },
    { write() {} },
  );
  assert.equal(unavailable, 0);
  assert.equal(calls.length, 0);
  assert.equal(JSON.parse(local).status, "unavailable");

  local = "";
  const attachmentCode = await main(
    ["task", "attachment", "inspect", "--path", attachment],
    taskEnv,
    fetchImpl,
    { write: (chunk) => { local += chunk; } },
    { write() {} },
  );
  fs.rmSync(taskDir, { recursive: true, force: true });
  assert.equal(attachmentCode, 0);
  assert.equal(calls.length, 0);
  assert.equal(JSON.parse(local).mimeType, "image/png");
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
