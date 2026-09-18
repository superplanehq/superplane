"use strict";

const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const {
  MAX_INSPECTABLE_SCREENSHOT_BYTES,
  inspectScreenshot,
  readManifest,
  reportVisualEvidenceUnavailable,
  resolveArtifactFile,
  uploadArtifact,
} = require("./task_artifact_mcp");

function fixture() {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "artifact-mcp-"));
  const evidence = path.join(taskDir, "evidence");
  fs.mkdirSync(evidence);
  const file = path.join(evidence, "screen.png");
  fs.writeFileSync(file, Buffer.from("png"));
  return {
    taskDir,
    evidence,
    file,
    env: {
      SUPERPLANE_TASK_DIR: taskDir,
      SUPERPLANE_BASE_URL: "https://app.example",
      SUPERPLANE_ARTIFACT_TOKEN: "token",
    },
  };
}

test("resolveArtifactFile accepts supported regular files under evidence", () => {
  const value = fixture();
  assert.deepEqual(resolveArtifactFile(value.file, value.env), {
    absolute: fs.realpathSync(value.file),
    filename: "screen.png",
    contentType: "image/png",
    sizeBytes: 3,
  });
});

test("resolveArtifactFile rejects files outside evidence and symlinks", () => {
  const value = fixture();
  const outside = path.join(value.taskDir, "outside.png");
  fs.writeFileSync(outside, "png");
  assert.throws(
    () => resolveArtifactFile(outside, value.env),
    /inside the task evidence directory/,
  );
  const link = path.join(value.evidence, "link.png");
  fs.symlinkSync(value.file, link);
  assert.throws(() => resolveArtifactFile(link, value.env), /regular file/);
});

test("resolveArtifactFile rejects a symlinked evidence directory", () => {
  const value = fixture();
  const external = fs.mkdtempSync(path.join(os.tmpdir(), "artifact-external-"));
  fs.rmSync(value.evidence, { recursive: true });
  fs.symlinkSync(external, value.evidence);
  const file = path.join(external, "screen.png");
  fs.writeFileSync(file, "png");

  assert.throws(
    () => resolveArtifactFile(file, value.env),
    /evidence directory must be a regular directory/,
  );
});

test("resolveArtifactFile rejects unsupported extensions", () => {
  const value = fixture();
  const unsupported = path.join(value.evidence, "report.pdf");
  fs.writeFileSync(unsupported, "pdf");
  assert.throws(
    () => resolveArtifactFile(unsupported, value.env),
    /not supported/,
  );
});

test("uploadArtifact streams metadata and records the returned artifact", async () => {
  const value = fixture();
  const inspected = inspectScreenshot({ path: value.file }, value.env);
  assert.equal(inspected.structuredContent.filename, "screen.png");
  const calls = [];
  const title = "Checkout → success";
  const result = await uploadArtifact(
    { path: value.file, title },
    value.env,
    async (url, options) => {
      calls.push({ url, options });
      return {
        ok: true,
        text: async () =>
          JSON.stringify({ file_id: "file-1", markdown: "![Checkout](url)" }),
      };
    },
  );
  assert.equal(result.file_id, "file-1");
  assert.equal(calls[0].url, "https://app.example/api/v1/runner/artifacts");
  assert.equal(calls[0].options.headers.Authorization, "Bearer token");
  assert.equal(calls[0].options.headers["Content-Length"], "3");
  assert.equal(
    decodeURIComponent(
      calls[0].options.headers["X-SuperPlane-Artifact-Title"],
    ),
    title,
  );
  assert.doesNotThrow(() => new Headers(calls[0].options.headers));
  assert.equal(readManifest(value.env).status, "captured");
  assert.equal(readManifest(value.env).artifacts.length, 1);

  await uploadArtifact({ path: value.file }, value.env, async () => ({
    ok: true,
    text: async () =>
      JSON.stringify({ file_id: "file-1", markdown: "![Checkout](url)" }),
  }));
  assert.equal(readManifest(value.env).artifacts.length, 1);
});

test("inspectScreenshot returns an image block and records its hash", () => {
  const value = fixture();
  const result = inspectScreenshot({ path: value.file }, value.env);

  assert.equal(result.content[0].type, "text");
  assert.deepEqual(result.content[1], {
    type: "image",
    data: Buffer.from("png").toString("base64"),
    mimeType: "image/png",
  });
  assert.match(result.structuredContent.sha256, /^[a-f0-9]{64}$/);
  const inspections = JSON.parse(fs.readFileSync(path.join(value.taskDir, "visual-evidence-inspections.json"), "utf8"));
  assert.equal(inspections.inspections[0].path, "screen.png");
  assert.equal(inspections.inspections[0].sha256, result.structuredContent.sha256);
});

test("inspectScreenshot rejects unsupported and oversized files", () => {
  const value = fixture();
  const video = path.join(value.evidence, "demo.webm");
  fs.writeFileSync(video, "video");
  assert.throws(() => inspectScreenshot({ path: video }, value.env), /PNG, JPEG, or WebP/);

  const oversized = path.join(value.evidence, "large.png");
  const descriptor = fs.openSync(oversized, "w");
  fs.ftruncateSync(descriptor, MAX_INSPECTABLE_SCREENSHOT_BYTES + 1);
  fs.closeSync(descriptor);
  assert.throws(() => inspectScreenshot({ path: oversized }, value.env), /smaller or more focused image/);
});

test("inspectScreenshot rejects traversal and symlinks", () => {
  const value = fixture();
  const outside = path.join(value.taskDir, "outside.png");
  fs.writeFileSync(outside, "png");
  assert.throws(() => inspectScreenshot({ path: outside }, value.env), /inside the task evidence directory/);

  const link = path.join(value.evidence, "linked.png");
  fs.symlinkSync(value.file, link);
  assert.throws(() => inspectScreenshot({ path: link }, value.env), /regular file/);
});

test("uploadArtifact rejects uninspected and modified screenshots", async () => {
  const value = fixture();
  const upload = async () => ({ ok: true, text: async () => JSON.stringify({ file_id: "file-1" }) });

  await assert.rejects(uploadArtifact({ path: value.file }, value.env, upload), /inspect_screenshot before upload_artifact/);
  inspectScreenshot({ path: value.file }, value.env);
  fs.appendFileSync(value.file, "changed");
  await assert.rejects(uploadArtifact({ path: value.file }, value.env, upload), /changed after inspection/);
});

test("uploadArtifact uploads videos without screenshot inspection", async () => {
  const value = fixture();
  const video = path.join(value.evidence, "demo.webm");
  fs.writeFileSync(video, "video");
  const result = await uploadArtifact({ path: video }, value.env, async () => ({
    ok: true,
    text: async () => JSON.stringify({ file_id: "video-1" }),
  }));
  assert.equal(result.file_id, "video-1");
});

test("uploadArtifact uploads an inspected poster before a video", async () => {
  const value = fixture();
  const video = path.join(value.evidence, "demo.webm");
  fs.writeFileSync(video, "video");
  inspectScreenshot({ path: value.file }, value.env);
  const calls = [];

  const result = await uploadArtifact(
    { path: video, posterPath: value.file, title: "Checkout flow" },
    value.env,
    async (_url, options) => {
      calls.push(options.headers["Content-Type"]);
      const isPoster = options.headers["Content-Type"] === "image/png";
      return {
        ok: true,
        text: async () =>
          JSON.stringify(
            isPoster
              ? { file_id: "poster-1", public_url: "https://app.example/poster.png", markdown: "![Checkout](poster)" }
              : { file_id: "video-1", public_url: "https://app.example/demo.webm", markdown: "[Checkout](video)" },
          ),
      };
    },
  );

  assert.equal(result.file_id, "video-1");
  assert.deepEqual(calls, ["image/png", "video/webm"]);
  const manifest = readManifest(value.env);
  assert.equal(manifest.status, "captured");
  assert.equal(manifest.artifacts.length, 2);
  assert.equal(manifest.artifacts[0].video_file_id, "video-1");
  assert.equal(manifest.artifacts[1].poster_file_id, "poster-1");
  assert.equal(
    manifest.artifacts[0].markdown,
    "[![Checkout flow](https://app.example/poster.png)](https://app.example/demo.webm)",
  );
  assert.equal(manifest.artifacts[1].markdown, "[Watch video: Checkout flow](https://app.example/demo.webm)");
});

test("uploadArtifact keeps a poster when the video upload fails", async () => {
  const value = fixture();
  const video = path.join(value.evidence, "demo.webm");
  fs.writeFileSync(video, "video");
  inspectScreenshot({ path: value.file }, value.env);
  let request = 0;

  await assert.rejects(
    uploadArtifact(
      { path: video, posterPath: value.file, title: "Checkout flow" },
      value.env,
      async () => {
        request += 1;
        if (request === 1) {
          return {
            ok: true,
            text: async () =>
              JSON.stringify({ file_id: "poster-1", public_url: "https://app.example/poster.png", markdown: "![Checkout](poster)" }),
          };
        }
        return { ok: false, status: 503, text: async () => JSON.stringify({ message: "storage unavailable" }) };
      },
    ),
    /storage unavailable/,
  );

  assert.deepEqual(readManifest(value.env), {
    status: "captured",
    reason: "",
    artifacts: [
      { file_id: "poster-1", public_url: "https://app.example/poster.png", markdown: "![Checkout](poster)" },
    ],
  });
});

test("uploadArtifact validates video posters as inspected screenshots", async () => {
  const value = fixture();
  const video = path.join(value.evidence, "demo.webm");
  const otherVideo = path.join(value.evidence, "poster.webm");
  fs.writeFileSync(video, "video");
  fs.writeFileSync(otherVideo, "video");
  const upload = async () => ({ ok: true, text: async () => JSON.stringify({ file_id: "artifact-1" }) });

  await assert.rejects(
    uploadArtifact({ path: video, posterPath: value.file }, value.env, upload),
    /inspect_screenshot before upload_artifact/,
  );
  await assert.rejects(
    uploadArtifact({ path: video, posterPath: otherVideo }, value.env, upload),
    /poster must be a PNG, JPEG, or WebP file/,
  );
  await assert.rejects(
    uploadArtifact({ path: value.file, posterPath: value.file }, value.env, upload),
    /posterPath is only supported for video artifacts/,
  );
});

test("reportVisualEvidenceUnavailable requires documented attempts", () => {
  const value = fixture();

  assert.throws(
    () =>
      reportVisualEvidenceUnavailable(
        { reason: "The preview did not start." },
        value.env,
      ),
    /attempts is required/,
  );

  assert.throws(
    () =>
      reportVisualEvidenceUnavailable(
        {
          reason: "The preview did not start.",
          attempts: [
            {
              type: "preview",
              command: "npm run storybook",
              outcome: "The command exited with status 1.",
            },
            {
              type: "preview",
              command: "npm run storybook -- --port 6007",
              outcome: "The command exited with status 1.",
            },
          ],
        },
        value.env,
      ),
    /one preview and one playwright attempt/,
  );

  reportVisualEvidenceUnavailable(
    {
      reason: "The preview did not start.",
      attempts: [
        {
          type: "preview",
          command: "npm run storybook",
          outcome: "The command exited with status 1.",
        },
        {
          type: "playwright",
          command: "playwright screenshot http://localhost:6006",
          outcome: "The command could not connect to localhost:6006.",
        },
      ],
    },
    value.env,
  );
  assert.deepEqual(readManifest(value.env), {
    status: "unavailable",
    reason: "The preview did not start.",
    attempts: [
      {
        type: "preview",
        command: "npm run storybook",
        outcome: "The command exited with status 1.",
      },
      {
        type: "playwright",
        command: "playwright screenshot http://localhost:6006",
        outcome: "The command could not connect to localhost:6006.",
      },
    ],
    artifacts: [],
  });
});

test("reportVisualEvidenceUnavailable keeps captured evidence available", () => {
  const value = fixture();
  inspectScreenshot({ path: value.file }, value.env);
  return uploadArtifact({ path: value.file }, value.env, async () => ({
    ok: true,
    text: async () => JSON.stringify({ file_id: "poster-1", markdown: "![Poster](poster)" }),
  })).then(() => {
    const result = reportVisualEvidenceUnavailable(
      {
        reason: "The video upload failed.",
        attempts: [
          { type: "preview", command: "npm run storybook", outcome: "The preview started." },
          { type: "playwright", command: "playwright video-start", outcome: "The video was captured." },
          { type: "upload", command: "upload_artifact", outcome: "The video upload failed." },
        ],
      },
      value.env,
    );

    assert.equal(result.status, "captured");
    assert.equal(result.artifacts.length, 1);
    assert.equal(result.reason, "The video upload failed.");
  });
});

test("lists artifact tools over newline-delimited JSON-RPC", () => {
  const value = fixture();
  const input = [
    JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} }),
    JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} }),
    "",
  ].join("\n");
  const result = spawnSync(
    process.execPath,
    [path.join(__dirname, "task_artifact_mcp.js")],
    {
      input,
      encoding: "utf8",
      env: { ...process.env, ...value.env },
    },
  );
  assert.equal(result.status, 0, result.stderr);
  const replies = result.stdout
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));
  assert.deepEqual(
    replies[1].result.tools.map((tool) => tool.name),
    ["inspect_screenshot", "upload_artifact", "report_visual_evidence_unavailable"],
  );
});

test("returns image content over JSON-RPC without copying base64 into structured content", () => {
  const value = fixture();
  const input = [
    JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      method: "tools/call",
      params: { name: "inspect_screenshot", arguments: { path: value.file } },
    }),
    "",
  ].join("\n");
  const result = spawnSync(process.execPath, [path.join(__dirname, "task_artifact_mcp.js")], {
    input,
    encoding: "utf8",
    env: { ...process.env, ...value.env },
  });
  assert.equal(result.status, 0, result.stderr);
  const reply = JSON.parse(result.stdout.trim());
  assert.equal(reply.result.content[1].type, "image");
  assert.equal(reply.result.content[1].data, Buffer.from("png").toString("base64"));
  assert.equal(reply.result.structuredContent.data, undefined);
});
