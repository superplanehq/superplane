"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const registry = require("./registry");
const { readManifest } = require("./modules/evidence");

function call(name, input, client) {
  return registry.resolve(name, registry.allTools()).handler(input, client);
}

function taskFixture(upload = async () => ({ file_id: "file-1" })) {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "global-flow-"));
  fs.mkdirSync(path.join(taskDir, "evidence"));
  fs.mkdirSync(path.join(taskDir, "attachments"));
  const screenshot = path.join(taskDir, "evidence", "screen.png");
  fs.writeFileSync(screenshot, "png");
  const video = path.join(taskDir, "evidence", "demo.webm");
  fs.writeFileSync(video, "video");
  const attachment = path.join(taskDir, "attachments", "bug.png");
  fs.writeFileSync(attachment, Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]));
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
      return upload();
    },
  };
  return { taskDir, screenshot, video, attachment, client, requests, uploads };
}

const ATTEMPTS = [
  { type: "preview", command: "npm run storybook", outcome: "The command exited with status 1." },
  { type: "playwright", command: "playwright screenshot http://localhost:6006", outcome: "No connection." },
];

test("upload_artifact needs an inspection of the final screenshot", async () => {
  const value = taskFixture();

  await assert.rejects(call("upload_artifact", { path: value.screenshot }, value.client), /call get_artifact before add_artifact/);
  call("inspect_screenshot", { path: value.screenshot }, value.client);
  fs.appendFileSync(value.screenshot, "changed");
  await assert.rejects(call("upload_artifact", { path: value.screenshot }, value.client), /screenshot changed after get_artifact/);
});

test("upload_artifact uploads an inspected screenshot without a key", async () => {
  const value = taskFixture();
  call("inspect_screenshot", { path: value.screenshot }, value.client);

  await call("upload_artifact", { path: value.screenshot, title: "Checkout", key: "ignored" }, value.client);

  assert.equal(value.uploads.length, 1);
  assert.equal(value.uploads[0].headers["X-SuperPlane-Artifact-Title"], "Checkout");
  assert.equal(Object.hasOwn(value.uploads[0].headers, "X-SuperPlane-Artifact-Key"), false);
});

test("upload_artifact keeps the poster when the video upload fails", async () => {
  let uploads = 0;
  const value = taskFixture(async () => {
    uploads += 1;
    if (uploads === 1) return { file_id: "poster-1", public_url: "https://app.example/poster.png" };
    throw new Error("storage unavailable");
  });
  call("inspect_screenshot", { path: value.screenshot }, value.client);

  await assert.rejects(
    call("upload_artifact", { path: value.video, posterPath: value.screenshot, title: "Checkout" }, value.client),
    /storage unavailable/,
  );
  assert.deepEqual(readManifest(value.client.env), {
    status: "partial",
    reason: "Video upload failed: storage unavailable",
    artifacts: [{ file_id: "poster-1", public_url: "https://app.example/poster.png" }],
  });
});

test("add_artifact keeps only the latest file of a keyed artifact in the evidence manifest", async () => {
  let uploads = 0;
  const value = taskFixture(async () => {
    uploads += 1;
    return { artifact_id: "artifact-1", file_id: `file-${uploads}` };
  });
  call("get_artifact", { path: value.screenshot }, value.client);

  await call("add_artifact", { type: "file", path: value.screenshot, key: "checkout" }, value.client);
  await call("add_artifact", { type: "file", path: value.screenshot, key: "checkout" }, value.client);

  assert.deepEqual(readManifest(value.client.env).artifacts, [{ artifact_id: "artifact-1", file_id: "file-2" }]);
});

test("add_artifact rejects a video key with no space for the poster key", async () => {
  const value = taskFixture();
  call("get_artifact", { path: value.screenshot }, value.client);
  const input = { type: "file", path: value.video, posterPath: value.screenshot };

  await assert.rejects(
    call("add_artifact", { ...input, key: "k".repeat(469) }, value.client),
    /key must be at most 468 bytes for a video with a poster/,
  );
  assert.equal(value.uploads.length, 0);

  await call("add_artifact", { ...input, key: "k".repeat(468) }, value.client);
  assert.equal(value.uploads[0].headers["X-SuperPlane-Artifact-Key"], `${"k".repeat(468)}%3Aposter`);
});

test("upload_artifact reads only the task evidence directory", async () => {
  const value = taskFixture();

  await assert.rejects(
    call("upload_artifact", { path: value.attachment }, value.client),
    /inside the task evidence directory/,
  );
  assert.throws(
    () => call("inspect_screenshot", { path: value.attachment }, value.client),
    /inside the task evidence directory/,
  );
});

test("report_visual_evidence_unavailable needs a preview and a Playwright attempt", () => {
  const value = taskFixture();

  assert.throws(
    () => call("report_visual_evidence_unavailable", { reason: "No preview.", attempts: [ATTEMPTS[0], ATTEMPTS[0]] }, value.client),
    /one preview and one playwright attempt/,
  );
  const result = call("report_visual_evidence_unavailable", { reason: "No preview.", attempts: ATTEMPTS }, value.client);
  assert.equal(result.status, "unavailable");
  assert.deepEqual(readManifest(value.client.env).attempts, ATTEMPTS);
});

test("inspect_attachment reads only the task attachments directory", () => {
  const value = taskFixture();

  assert.throws(
    () => call("inspect_attachment", { path: value.screenshot }, value.client),
    /inside the task attachments directory/,
  );
  const result = call("inspect_attachment", { path: "bug.png" }, value.client);
  assert.equal(result.structuredContent.path, "bug.png");
  assert.equal(result.content[1].mimeType, "image/png");
});

test("inspect_attachment rejects a file that is not an image", () => {
  const value = taskFixture();
  fs.writeFileSync(path.join(value.taskDir, "attachments", "notes.png"), "not an image");

  assert.throws(
    () => call("inspect_attachment", { path: "notes.png" }, value.client),
    /attachment must be a PNG, JPEG, GIF, or WebP file/,
  );
});

test("publish_link sends the same request as add_artifact with type link", async () => {
  const value = taskFixture();
  const input = { url: "https://preview.example", title: "Preview", key: "preview" };

  await call("publish_link", input, value.client);
  await call("add_artifact", { ...input, type: "link" }, value.client);

  assert.deepEqual(value.requests[0], {
    method: "POST",
    path: "/api/v1/runner/task/links",
    body: { url: "https://preview.example", title: "Preview", key: "preview" },
  });
  assert.deepEqual(value.requests[1], value.requests[0]);
});

test("propose_update sends the same request as propose_plan", async () => {
  const value = taskFixture();
  value.client.env.SUPERPLANE_ANALYSIS_SPEC_FILE = path.join(value.taskDir, "spec.md");
  const input = { spec: "# Plan" };

  await call("propose_update", input, value.client);
  await call("propose_plan", input, value.client);

  assert.equal(value.requests.length, 2);
  assert.deepEqual(value.requests[1], value.requests[0]);
});

test("flow tools reject invalid input before they call the backend", async () => {
  const value = taskFixture();

  for (const name of ["propose_plan", "propose_update"]) {
    await assert.rejects(call(name, {}, value.client), /scores, spec, or survey is required/);
    await assert.rejects(call(name, { scores: { clarity: { score: 3 } } }, value.client), /clarity summary is required/);
  }
  await assert.rejects(call("publish_link", { url: "file:///etc/passwd" }, value.client), /url must be http or https/);
  await assert.rejects(call("report_merge_check", { check: "risk", score: 2 }, value.client), /summary is required/);
  assert.deepEqual(value.requests, []);
});
