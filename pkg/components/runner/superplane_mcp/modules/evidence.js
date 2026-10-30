"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const MAX_ARTIFACT_BYTES = 100 * 1024 * 1024;
const MAX_INSPECTABLE_SCREENSHOT_BYTES = 5 * 1024 * 1024;
const ATTEMPT_TYPES = new Set(["preview", "playwright", "upload"]);
const REQUIRED_ATTEMPT_TYPES = ["preview", "playwright"];
const CONTENT_TYPES = new Map([
  [".png", "image/png"],
  [".jpg", "image/jpeg"],
  [".jpeg", "image/jpeg"],
  [".webp", "image/webp"],
  [".webm", "video/webm"],
  [".mp4", "video/mp4"],
]);
const SCREENSHOT_CONTENT_TYPES = new Set(["image/png", "image/jpeg", "image/webp"]);

function artifactPaths(env) {
  const taskDir = String(env.SUPERPLANE_TASK_DIR || "").trim();
  if (!taskDir) throw new Error("SUPERPLANE_TASK_DIR is required");
  const root = path.resolve(taskDir);
  return {
    taskDir: root,
    evidenceDir: path.join(root, "evidence"),
    manifest: path.join(root, "visual-evidence.json"),
    inspections: path.join(root, "visual-evidence-inspections.json"),
  };
}

function readManifest(env) {
  try {
    const parsed = JSON.parse(fs.readFileSync(artifactPaths(env).manifest, "utf8"));
    const result = {
      status: String(parsed.status || "not_applicable"),
      reason: String(parsed.reason || ""),
      artifacts: Array.isArray(parsed.artifacts) ? parsed.artifacts : [],
    };
    if (Array.isArray(parsed.attempts)) result.attempts = parsed.attempts;
    return result;
  } catch (_error) {
    return { status: "not_applicable", reason: "", artifacts: [] };
  }
}

function writeManifest(manifest, env) {
  const paths = artifactPaths(env);
  writeJSONAtomically(paths.manifest, manifest, paths.taskDir);
}

function writeJSONAtomically(destination, value, parentDirectory) {
  fs.mkdirSync(parentDirectory, { recursive: true });
  const temporary = `${destination}.${process.pid}.${crypto.randomUUID()}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o600 });
  fs.renameSync(temporary, destination);
}

function readInspections(env) {
  try {
    const parsed = JSON.parse(fs.readFileSync(artifactPaths(env).inspections, "utf8"));
    return { inspections: Array.isArray(parsed.inspections) ? parsed.inspections : [] };
  } catch (_error) {
    return { inspections: [] };
  }
}

function writeInspections(value, env) {
  const paths = artifactPaths(env);
  writeJSONAtomically(paths.inspections, value, paths.taskDir);
}

function screenshotPath(file, env) {
  return path.relative(fs.realpathSync(artifactPaths(env).evidenceDir), file.absolute);
}

function fileSHA256(filePath) {
  return crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex");
}

function resolveArtifactFile(inputPath, env) {
  const candidate = String(inputPath || "").trim();
  if (!candidate) throw new Error("path is required");
  const paths = artifactPaths(env);
  fs.mkdirSync(paths.evidenceDir, { recursive: true });
  const evidenceInfo = fs.lstatSync(paths.evidenceDir);
  if (evidenceInfo.isSymbolicLink() || !evidenceInfo.isDirectory()) {
    throw new Error("task evidence directory must be a regular directory");
  }
  const evidenceRoot = fs.realpathSync(paths.evidenceDir);
  const absolute = path.resolve(candidate);
  const fileInfo = fs.lstatSync(absolute);
  if (fileInfo.isSymbolicLink() || !fileInfo.isFile()) {
    throw new Error("artifact path must be a regular file");
  }
  const resolved = fs.realpathSync(absolute);
  if (resolved !== evidenceRoot && !resolved.startsWith(`${evidenceRoot}${path.sep}`)) {
    throw new Error("artifact path must be inside the task evidence directory");
  }
  if (fileInfo.size <= 0) throw new Error("artifact file is empty");
  if (fileInfo.size > MAX_ARTIFACT_BYTES) throw new Error(`artifact exceeds ${MAX_ARTIFACT_BYTES} bytes`);
  const contentType = CONTENT_TYPES.get(path.extname(resolved).toLowerCase());
  if (!contentType) throw new Error("artifact file type is not supported");
  return { absolute: resolved, filename: path.basename(resolved), contentType, sizeBytes: fileInfo.size };
}

function inspectScreenshot(input, client) {
  const env = client.env;
  const file = resolveArtifactFile(input && input.path, env);
  if (!SCREENSHOT_CONTENT_TYPES.has(file.contentType)) {
    throw new Error("screenshot must be a PNG, JPEG, or WebP file");
  }
  if (file.sizeBytes > MAX_INSPECTABLE_SCREENSHOT_BYTES) {
    throw new Error(
      `screenshot exceeds ${MAX_INSPECTABLE_SCREENSHOT_BYTES} bytes; capture a smaller or more focused image`,
    );
  }
  const bytes = fs.readFileSync(file.absolute);
  const metadata = {
    path: screenshotPath(file, env),
    filename: file.filename,
    mimeType: file.contentType,
    sizeBytes: file.sizeBytes,
    sha256: crypto.createHash("sha256").update(bytes).digest("hex"),
  };
  const inspections = readInspections(env).inspections.filter((inspection) => inspection.path !== metadata.path);
  inspections.push(metadata);
  writeInspections({ inspections }, env);
  return {
    content: [
      { type: "text", text: JSON.stringify(metadata) },
      { type: "image", data: bytes.toString("base64"), mimeType: file.contentType },
    ],
    structuredContent: metadata,
  };
}

function requireInspectedScreenshot(file, env) {
  if (!SCREENSHOT_CONTENT_TYPES.has(file.contentType)) return;
  const relativePath = screenshotPath(file, env);
  const inspection = readInspections(env).inspections.find((entry) => entry.path === relativePath);
  if (!inspection) throw new Error("call inspect_screenshot before upload_artifact for every screenshot");
  if (inspection.sizeBytes !== file.sizeBytes || inspection.sha256 !== fileSHA256(file.absolute)) {
    throw new Error("screenshot changed after inspection; inspect the final file again before upload");
  }
}

function uploadFile(file, title, client) {
  const headers = {
    "Content-Type": file.contentType,
    "Content-Length": String(file.sizeBytes),
    "Content-Disposition": `attachment; filename*=UTF-8''${encodeURIComponent(file.filename)}`,
  };
  if (title) headers["X-SuperPlane-Artifact-Title"] = encodeURIComponent(title);
  return client.upload("/api/v1/runner/task/artifacts", headers, fs.createReadStream(file.absolute));
}

function recordArtifact(result, env) {
  const manifest = readManifest(env);
  const artifacts = manifest.artifacts.filter((item) => item.file_id !== result.file_id);
  artifacts.push(result);
  const partial = manifest.status === "partial";
  writeManifest({ status: partial ? "partial" : "captured", reason: partial ? manifest.reason : "", artifacts }, env);
}

function markdownLabel(value) {
  return String(value || "Evidence")
    .replace(/\\/g, "\\\\")
    .replace(/\[/g, "\\[")
    .replace(/\]/g, "\\]");
}

async function uploadVideoWithPoster(file, poster, title, client) {
  const env = client.env;
  if (!SCREENSHOT_CONTENT_TYPES.has(poster.contentType)) {
    throw new Error("poster must be a PNG, JPEG, or WebP file");
  }
  requireInspectedScreenshot(poster, env);

  const posterResult = await uploadFile(poster, title ? `${title} poster` : "Video poster", client);
  let videoResult;
  try {
    videoResult = await uploadFile(file, title, client);
  } catch (error) {
    const manifest = readManifest(env);
    const artifacts = manifest.artifacts.filter((item) => item.file_id !== posterResult.file_id);
    artifacts.push(posterResult);
    writeManifest({ status: "partial", reason: `Video upload failed: ${error.message}`, artifacts }, env);
    throw error;
  }
  const label = markdownLabel(title || file.filename);
  const linkedPoster = {
    ...posterResult,
    video_file_id: videoResult.file_id,
    markdown: `[![${label}](${posterResult.public_url})](${videoResult.public_url})`,
  };
  const linkedVideo = {
    ...videoResult,
    poster_file_id: posterResult.file_id,
    markdown: `[Watch video: ${label}](${videoResult.public_url})`,
  };
  const manifest = readManifest(env);
  const artifacts = manifest.artifacts.filter(
    (item) => item.file_id !== posterResult.file_id && item.file_id !== videoResult.file_id,
  );
  artifacts.push(linkedPoster, linkedVideo);
  writeManifest({ status: "captured", reason: "", artifacts }, env);
  return linkedVideo;
}

async function uploadArtifact(input, client) {
  const env = client.env;
  const file = resolveArtifactFile(input && input.path, env);
  const title = String((input && input.title) || "").trim();
  const posterPath = String((input && input.posterPath) || "").trim();
  if (posterPath && !file.contentType.startsWith("video/")) {
    throw new Error("posterPath is only supported for video artifacts");
  }
  requireInspectedScreenshot(file, env);
  if (posterPath) {
    return uploadVideoWithPoster(file, resolveArtifactFile(posterPath, env), title, client);
  }
  const result = await uploadFile(file, title, client);
  recordArtifact(result, env);
  return result;
}

function normalizeEvidenceAttempts(value) {
  if (!Array.isArray(value) || value.length < 2) {
    throw new Error("attempts is required and must include one preview and one playwright attempt");
  }
  const attempts = value.map((attempt, index) => {
    if (!attempt || typeof attempt !== "object" || Array.isArray(attempt)) {
      throw new Error(`attempts[${index}] must be an object`);
    }
    const type = String(attempt.type || "").trim();
    const command = String(attempt.command || "").trim();
    const outcome = String(attempt.outcome || "").trim();
    if (!ATTEMPT_TYPES.has(type)) throw new Error(`attempts[${index}].type must be preview, playwright, or upload`);
    if (!command) throw new Error(`attempts[${index}].command is required`);
    if (!outcome) throw new Error(`attempts[${index}].outcome is required`);
    return { type, command, outcome };
  });
  const attemptTypes = new Set(attempts.map((attempt) => attempt.type));
  if (REQUIRED_ATTEMPT_TYPES.some((type) => !attemptTypes.has(type))) {
    throw new Error("attempts must include one preview and one playwright attempt");
  }
  return attempts;
}

function reportVisualEvidenceUnavailable(input, client) {
  const env = client.env;
  const reason = String((input && input.reason) || "").trim();
  if (!reason) throw new Error("reason is required");
  const attempts = normalizeEvidenceAttempts(input && input.attempts);
  const manifest = readManifest(env);
  let status = "unavailable";
  if (manifest.artifacts.length > 0) status = manifest.status === "partial" ? "partial" : "captured";
  const result = { status, reason, attempts, artifacts: manifest.artifacts };
  writeManifest(result, env);
  return result;
}

// prepare writes the evidence manifest at startup, so a run that captures
// nothing still leaves a readable file.
function prepare(client) {
  writeManifest(readManifest(client.env), client.env);
}

const INSPECT_SCREENSHOT = {
  definition: {
    name: "inspect_screenshot",
    description:
      "Inspect a screenshot before upload. Returns the image with its filename, MIME type, size, and SHA-256 hash.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" } },
      required: ["path"],
    },
  },
  handler: inspectScreenshot,
};

const UPLOAD_ARTIFACT = {
  definition: {
    name: "upload_artifact",
    description: "Upload a screenshot or video from the task evidence directory and attach it to the work order.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string" },
        title: { type: "string" },
        posterPath: {
          type: "string",
          description: "Inspected PNG, JPEG, or WebP poster to show inline for a video artifact.",
        },
      },
      required: ["path"],
    },
  },
  handler: uploadArtifact,
};

const REPORT_VISUAL_EVIDENCE_UNAVAILABLE = {
  definition: {
    name: "report_visual_evidence_unavailable",
    description:
      "Record why required visual evidence could not be produced after trying an isolated preview and a Playwright capture.",
    inputSchema: {
      type: "object",
      properties: {
        reason: { type: "string" },
        attempts: {
          type: "array",
          description: "Concrete preview and Playwright attempts. Add an upload attempt when upload fails.",
          items: {
            type: "object",
            properties: {
              type: { type: "string", enum: ["preview", "playwright", "upload"] },
              command: { type: "string", minLength: 1 },
              outcome: { type: "string", minLength: 1 },
            },
            required: ["type", "command", "outcome"],
            additionalProperties: false,
          },
          minItems: 2,
        },
      },
      required: ["reason", "attempts"],
    },
  },
  handler: reportVisualEvidenceUnavailable,
};

module.exports = {
  MAX_INSPECTABLE_SCREENSHOT_BYTES,
  readManifest,
  prepare,
  tools: [INSPECT_SCREENSHOT, UPLOAD_ARTIFACT, REPORT_VISUAL_EVIDENCE_UNAVAILABLE],
};
