"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

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

test("remove_artifact counts an emoji as one character", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls));
  const title = `${"a".repeat(1023)}😀`;

  await registry.findTool("remove_artifact").handler({ title }, client);

  assert.equal(calls[0].body.title, title);
  await assert.rejects(
    () => registry.findTool("remove_artifact").handler({ title: `${title}b` }, client),
    /title must be at most 1024 characters/,
  );
});

test("remove_artifact requires a key or a title", async () => {
  const client = createClient(env, recordingFetch([]));
  await assert.rejects(
    () => registry.findTool("remove_artifact").handler({}, client),
    /key or title is required/,
  );
});

const PNG = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);

function call(name, input, client) {
  return registry.findTool(name).handler(input, client);
}

test("add_artifact limits the title", async () => {
  const calls = [];
  const client = createClient(env, recordingFetch(calls));

  await assert.rejects(
    () => call("add_artifact", { type: "link", url: "https://example.com", title: "a".repeat(1025) }, client),
    /title must be at most 1024 characters/,
  );
  assert.equal(calls.length, 0);
});

function taskClient() {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "global-artifacts-"));
  fs.mkdirSync(path.join(taskDir, "evidence"));
  fs.writeFileSync(path.join(taskDir, "evidence", "screen.png"), PNG);
  fs.writeFileSync(path.join(taskDir, "notes.txt"), "notes");
  const requests = [];
  const uploads = [];
  const client = {
    env: { SUPERPLANE_TASK_DIR: taskDir },
    request: async (method, requestPath, body) => {
      requests.push({ method, path: requestPath, body });
      return { status: "ok" };
    },
    upload: async (requestPath, headers, stream) => {
      uploads.push({ path: requestPath, headers });
      stream.destroy();
      return { file_id: "file-1", public_url: "https://app.example/screen.png" };
    },
  };
  return { taskDir, client, requests, uploads };
}

test("list_artifacts filters by type", async () => {
  const value = taskClient();

  await call("list_artifacts", {}, value.client);
  await call("list_artifacts", { type: " Link " }, value.client);

  assert.deepEqual(value.requests.map((request) => request.path), [
    "/api/v1/runner/task/artifacts",
    "/api/v1/runner/task/artifacts?type=link",
  ]);
  await assert.rejects(call("list_artifacts", { type: "video" }, value.client), /type must be markdown, link, or file/);
});

test("get_artifact reads an artifact by id or key", async () => {
  const value = taskClient();

  await call("get_artifact", { id: "a1" }, value.client);
  await call("get_artifact", { key: "plan notes" }, value.client);

  assert.deepEqual(value.requests.map((request) => request.path), [
    "/api/v1/runner/task/artifact?id=a1",
    "/api/v1/runner/task/artifact?key=plan+notes",
  ]);
});

test("get_artifact needs exactly one of id, key, or path", async () => {
  const value = taskClient();

  await assert.rejects(call("get_artifact", {}, value.client), /give one of id, key, or path/);
  await assert.rejects(call("get_artifact", { id: "a1", key: "k" }, value.client), /give one of id, key, or path/);
  assert.deepEqual(value.requests, []);
});

test("get_artifact returns an image for a task file and metadata for other files", async () => {
  const value = taskClient();

  const image = await call("get_artifact", { path: "evidence/screen.png" }, value.client);
  const other = await call("get_artifact", { path: "$SUPERPLANE_TASK_DIR/notes.txt" }, value.client);

  assert.equal(image.content[1].type, "image");
  assert.equal(image.content[1].mimeType, "image/png");
  assert.equal(image.structuredContent.path, path.join("evidence", "screen.png"));
  assert.equal(image.structuredContent.sizeBytes, PNG.length);
  assert.equal(other.filename, "notes.txt");
  assert.equal(other.mimeType, "application/octet-stream");
  assert.match(other.sha256, /^[0-9a-f]{64}$/);
});

test("get_artifact rejects a file outside the task directory or a symbolic link", async () => {
  const value = taskClient();
  const outside = path.join(fs.mkdtempSync(path.join(os.tmpdir(), "outside-")), "secret.png");
  fs.writeFileSync(outside, PNG);
  fs.symlinkSync(outside, path.join(value.taskDir, "link.png"));

  await assert.rejects(call("get_artifact", { path: outside }, value.client), /inside the task directory/);
  await assert.rejects(call("get_artifact", { path: "link.png" }, value.client), /regular file/);
});

test("add_artifact uploads a screenshot only after get_artifact reads the final file", async () => {
  const value = taskClient();
  const input = { type: "file", path: "evidence/screen.png", title: "Checkout", key: "checkout" };

  await assert.rejects(call("add_artifact", input, value.client), /call get_artifact before add_artifact/);
  await call("get_artifact", { path: "evidence/screen.png" }, value.client);
  await call("add_artifact", input, value.client);
  fs.appendFileSync(path.join(value.taskDir, "evidence", "screen.png"), "changed");
  await assert.rejects(call("add_artifact", input, value.client), /screenshot changed after get_artifact/);

  assert.equal(value.uploads.length, 1);
  assert.equal(value.uploads[0].path, "/api/v1/runner/task/artifacts");
  assert.equal(value.uploads[0].headers["Content-Type"], "image/png");
  assert.equal(value.uploads[0].headers["X-SuperPlane-Artifact-Title"], "Checkout");
  assert.equal(value.uploads[0].headers["X-SuperPlane-Artifact-Key"], "checkout");
});

test("add_artifact posts a markdown note and a link", async () => {
  const value = taskClient();

  await call("add_artifact", { type: "markdown", body: " # Notes ", title: "Notes", key: "notes" }, value.client);
  await call("add_artifact", { type: "link", url: "https://preview.example", key: "preview" }, value.client);

  assert.deepEqual(value.requests, [
    { method: "POST", path: "/api/v1/runner/task/markdown", body: { body: "# Notes", title: "Notes", key: "notes" } },
    { method: "POST", path: "/api/v1/runner/task/links", body: { url: "https://preview.example", title: "", key: "preview" } },
  ]);
});

test("add_artifact rejects invalid input before it calls the backend", async () => {
  const value = taskClient();

  await assert.rejects(call("add_artifact", { body: "x" }, value.client), /type must be markdown, link, or file/);
  await assert.rejects(call("add_artifact", { type: "markdown" }, value.client), /body is required/);
  await assert.rejects(call("add_artifact", { type: "link", url: "file:///etc/passwd" }, value.client), /url must be http or https/);
  await assert.rejects(call("add_artifact", { type: "file", path: "notes.txt" }, value.client), /file type is not supported/);
  await assert.rejects(
    call("add_artifact", { type: "link", url: "https://a.example", key: "k".repeat(476) }, value.client),
    /key must be at most 475 bytes/,
  );
  assert.deepEqual(value.requests, []);
  assert.deepEqual(value.uploads, []);
});
