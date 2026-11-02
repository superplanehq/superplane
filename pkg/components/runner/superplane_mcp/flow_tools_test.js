"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const registry = require("./registry");
const { readManifest } = require("./modules/evidence");

function call(name, input, client) {
  return registry.findTool(name).handler(input, client);
}

function taskFixture(upload = async () => ({ file_id: "file-1" })) {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "global-flow-"));
  fs.mkdirSync(path.join(taskDir, "evidence"));
  fs.mkdirSync(path.join(taskDir, "attachments"));
  const screenshot = path.join(taskDir, "evidence", "screen.png");
  fs.writeFileSync(screenshot, "png");
  const video = path.join(taskDir, "evidence", "demo.webm");
  fs.writeFileSync(video, "video");
  const requests = [];
  const client = {
    env: { SUPERPLANE_TASK_DIR: taskDir },
    request: async (method, requestPath, body) => {
      requests.push({ method, path: requestPath, body });
      return { status: "ok" };
    },
    upload,
  };
  return { taskDir, screenshot, video, client, requests };
}

const ATTEMPTS = [
  { type: "preview", command: "npm run storybook", outcome: "The command exited with status 1." },
  { type: "playwright", command: "playwright screenshot http://localhost:6006", outcome: "No connection." },
];

test("upload_artifact needs an inspection of the final screenshot", async () => {
  const value = taskFixture();

  await assert.rejects(call("upload_artifact", { path: value.screenshot }, value.client), /inspect_screenshot before upload_artifact/);
  call("inspect_screenshot", { path: value.screenshot }, value.client);
  fs.appendFileSync(value.screenshot, "changed");
  await assert.rejects(call("upload_artifact", { path: value.screenshot }, value.client), /changed after inspection/);
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
});

test("flow tools reject invalid input before they call the backend", async () => {
  const value = taskFixture();

  await assert.rejects(call("propose_update", {}, value.client), /scores, spec, or survey is required/);
  await assert.rejects(
    call("propose_update", { scores: { clarity: { score: 3 } } }, value.client),
    /clarity summary is required/,
  );
  await assert.rejects(call("publish_link", { url: "file:///etc/passwd" }, value.client), /url must be http or https/);
  await assert.rejects(call("report_merge_check", { check: "risk", score: 2 }, value.client), /summary is required/);
  assert.deepEqual(value.requests, []);
});
