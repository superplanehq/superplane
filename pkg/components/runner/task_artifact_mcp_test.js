"use strict";

const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const {
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

  reportVisualEvidenceUnavailable(
    {
      reason: "The preview did not start.",
      attempts: [
        "npm run storybook failed with exit code 1.",
        "playwright screenshot could not connect to localhost:6006.",
      ],
    },
    value.env,
  );
  assert.deepEqual(readManifest(value.env), {
    status: "unavailable",
    reason: "The preview did not start.",
    attempts: [
      "npm run storybook failed with exit code 1.",
      "playwright screenshot could not connect to localhost:6006.",
    ],
    artifacts: [],
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
    ["upload_artifact", "report_visual_evidence_unavailable"],
  );
});
