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
  const screenshot = path.join(taskDir, "evidence", "screen.png");
  fs.writeFileSync(screenshot, "png");
  const video = path.join(taskDir, "evidence", "demo.webm");
  fs.writeFileSync(video, "video");
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
  return { taskDir, screenshot, video, client, requests, uploads };
}

const ATTEMPTS = [
  { type: "preview", command: "npm run storybook", outcome: "The command exited with status 1." },
  { type: "playwright", command: "playwright screenshot http://localhost:6006", outcome: "No connection." },
];

test("add_artifact keeps the poster when the video upload fails", async () => {
  let uploads = 0;
  const value = taskFixture(async () => {
    uploads += 1;
    if (uploads === 1) return { file_id: "poster-1", public_url: "https://app.example/poster.png" };
    throw new Error("storage unavailable");
  });
  call("get_artifact", { path: value.screenshot }, value.client);

  await assert.rejects(
    call("add_artifact", { type: "file", path: value.video, posterPath: value.screenshot, title: "Checkout" }, value.client),
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

test("flow tools reject invalid input before they call the backend", async () => {
  const value = taskFixture();

  await assert.rejects(call("propose_plan", {}, value.client), /scores, spec, or survey is required/);
  await assert.rejects(call("propose_plan", { scores: { clarity: { score: 3 } } }, value.client), /clarity summary is required/);
  await assert.rejects(call("report_merge_check", { check: "risk", score: 2 }, value.client), /summary is required/);
  assert.deepEqual(value.requests, []);
});
