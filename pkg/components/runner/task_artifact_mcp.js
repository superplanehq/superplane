#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const crypto = require("node:crypto");
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

function requiredEnv(name, env = process.env) {
  const value = String(env[name] || "").trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

function artifactPaths(env = process.env) {
  const taskDir = path.resolve(requiredEnv("SUPERPLANE_TASK_DIR", env));
  return {
    taskDir,
    evidenceDir: path.join(taskDir, "evidence"),
    manifest: path.join(taskDir, "visual-evidence.json"),
    inspections: path.join(taskDir, "visual-evidence-inspections.json"),
  };
}

function readManifest(env = process.env) {
  const { manifest } = artifactPaths(env);
  try {
    const parsed = JSON.parse(fs.readFileSync(manifest, "utf8"));
    const result = {
      status: String(parsed.status || "not_applicable"),
      reason: String(parsed.reason || ""),
      artifacts: Array.isArray(parsed.artifacts) ? parsed.artifacts : [],
    };
    if (Array.isArray(parsed.attempts)) {
      result.attempts = parsed.attempts;
    }
    return result;
  } catch (_error) {
    return { status: "not_applicable", reason: "", artifacts: [] };
  }
}

function writeManifest(manifest, env = process.env) {
  const paths = artifactPaths(env);
  writeJSONAtomically(paths.manifest, manifest, paths.taskDir);
}

function writeJSONAtomically(destination, value, parentDirectory) {
  fs.mkdirSync(parentDirectory, { recursive: true });
  const temporary = `${destination}.${process.pid}.${crypto.randomUUID()}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, {
    mode: 0o600,
  });
  fs.renameSync(temporary, destination);
}

function readInspections(env = process.env) {
  const { inspections } = artifactPaths(env);
  try {
    const parsed = JSON.parse(fs.readFileSync(inspections, "utf8"));
    return { inspections: Array.isArray(parsed.inspections) ? parsed.inspections : [] };
  } catch (_error) {
    return { inspections: [] };
  }
}

function writeInspections(value, env = process.env) {
  const paths = artifactPaths(env);
  writeJSONAtomically(paths.inspections, value, paths.taskDir);
}

function screenshotPath(file, env = process.env) {
  return path.relative(fs.realpathSync(artifactPaths(env).evidenceDir), file.absolute);
}

function fileSHA256(filePath) {
  return crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex");
}

function inspectScreenshot(input, env = process.env) {
  const file = resolveArtifactFile(input && input.path, env);
  if (!SCREENSHOT_CONTENT_TYPES.has(file.contentType)) {
    throw new Error("screenshot must be a PNG, JPEG, or WebP file");
  }
  if (file.sizeBytes > MAX_INSPECTABLE_SCREENSHOT_BYTES) {
    throw new Error(`screenshot exceeds ${MAX_INSPECTABLE_SCREENSHOT_BYTES} bytes; capture a smaller or more focused image`);
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

function requireInspectedScreenshot(file, env = process.env) {
  if (!SCREENSHOT_CONTENT_TYPES.has(file.contentType)) return;

  const relativePath = screenshotPath(file, env);
  const inspection = readInspections(env).inspections.find((entry) => entry.path === relativePath);
  if (!inspection) {
    throw new Error("call inspect_screenshot before upload_artifact for every screenshot");
  }
  if (inspection.sizeBytes !== file.sizeBytes || inspection.sha256 !== fileSHA256(file.absolute)) {
    throw new Error("screenshot changed after inspection; inspect the final file again before upload");
  }
}

function resolveArtifactFile(inputPath, env = process.env) {
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
  if (
    resolved !== evidenceRoot &&
    !resolved.startsWith(`${evidenceRoot}${path.sep}`)
  ) {
    throw new Error("artifact path must be inside the task evidence directory");
  }
  if (fileInfo.size <= 0) throw new Error("artifact file is empty");
  if (fileInfo.size > MAX_ARTIFACT_BYTES) {
    throw new Error(`artifact exceeds ${MAX_ARTIFACT_BYTES} bytes`);
  }
  const contentType = CONTENT_TYPES.get(path.extname(resolved).toLowerCase());
  if (!contentType) throw new Error("artifact file type is not supported");
  return {
    absolute: resolved,
    filename: path.basename(resolved),
    contentType,
    sizeBytes: fileInfo.size,
  };
}

async function uploadFile(file, title, env, fetchImpl) {
  const baseURL = requiredEnv("SUPERPLANE_BASE_URL", env).replace(/\/$/, "");
  const token = requiredEnv("SUPERPLANE_ARTIFACT_TOKEN", env);
  const headers = {
    Authorization: `Bearer ${token}`,
    Accept: "application/json",
    "Content-Type": file.contentType,
    "Content-Length": String(file.sizeBytes),
    "Content-Disposition": `attachment; filename*=UTF-8''${encodeURIComponent(file.filename)}`,
  };
  if (title) headers["X-SuperPlane-Artifact-Title"] = encodeURIComponent(title);

  const response = await fetchImpl(`${baseURL}/api/v1/runner/artifacts`, {
    method: "POST",
    headers,
    body: fs.createReadStream(file.absolute),
    duplex: "half",
  });
  const text = await response.text();
  let result = {};
  try {
    result = text ? JSON.parse(text) : {};
  } catch (_error) {
    result = { message: text };
  }
  if (!response.ok) {
    throw new Error(
      result.message || result.error || text || `HTTP ${response.status}`,
    );
  }

  return result;
}

function recordArtifact(result, env = process.env) {
  const manifest = readManifest(env);
  const artifacts = manifest.artifacts.filter(
    (item) => item.file_id !== result.file_id,
  );
  artifacts.push(result);
  writeManifest({ status: "captured", reason: "", artifacts }, env);
}

function markdownLabel(value) {
  return String(value || "Evidence")
    .replace(/\\/g, "\\\\")
    .replace(/\[/g, "\\[")
    .replace(/\]/g, "\\]");
}

async function uploadVideoWithPoster(file, poster, title, env, fetchImpl) {
  if (!SCREENSHOT_CONTENT_TYPES.has(poster.contentType)) {
    throw new Error("poster must be a PNG, JPEG, or WebP file");
  }
  requireInspectedScreenshot(poster, env);

  const posterTitle = title ? `${title} poster` : "Video poster";
  const posterResult = await uploadFile(poster, posterTitle, env, fetchImpl);
  recordArtifact(posterResult, env);

  const videoResult = await uploadFile(file, title, env, fetchImpl);
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

async function uploadArtifact(input, env = process.env, fetchImpl = fetch) {
  const file = resolveArtifactFile(input && input.path, env);
  const title = String((input && input.title) || "").trim();
  const posterPath = String((input && input.posterPath) || "").trim();
  const isVideo = file.contentType.startsWith("video/");
  if (posterPath && !isVideo) {
    throw new Error("posterPath is only supported for video artifacts");
  }
  requireInspectedScreenshot(file, env);
  if (posterPath) {
    const poster = resolveArtifactFile(posterPath, env);
    return uploadVideoWithPoster(file, poster, title, env, fetchImpl);
  }

  const result = await uploadFile(file, title, env, fetchImpl);
  recordArtifact(result, env);
  return result;
}

function reportVisualEvidenceUnavailable(input, env = process.env) {
  const reason = String((input && input.reason) || "").trim();
  if (!reason) throw new Error("reason is required");
  const attempts = normalizeEvidenceAttempts(input && input.attempts);
  const manifest = readManifest(env);
  const result = {
    status: manifest.artifacts.length > 0 ? "captured" : "unavailable",
    reason,
    attempts,
    artifacts: manifest.artifacts,
  };
  writeManifest(result, env);
  return result;
}

function normalizeEvidenceAttempts(value) {
  if (!Array.isArray(value) || value.length < 2) {
    throw new Error(
      "attempts is required and must include one preview and one playwright attempt",
    );
  }

  const attempts = value.map((attempt, index) => {
    if (!attempt || typeof attempt !== "object" || Array.isArray(attempt)) {
      throw new Error(`attempts[${index}] must be an object`);
    }
    const type = String(attempt.type || "").trim();
    const command = String(attempt.command || "").trim();
    const outcome = String(attempt.outcome || "").trim();
    if (!ATTEMPT_TYPES.has(type)) {
      throw new Error(
        `attempts[${index}].type must be preview, playwright, or upload`,
      );
    }
    if (!command) throw new Error(`attempts[${index}].command is required`);
    if (!outcome) throw new Error(`attempts[${index}].outcome is required`);
    return { type, command, outcome };
  });

  const attemptTypes = new Set(attempts.map((attempt) => attempt.type));
  if (REQUIRED_ATTEMPT_TYPES.some((type) => !attemptTypes.has(type))) {
    throw new Error(
      "attempts must include one preview and one playwright attempt",
    );
  }
  return attempts;
}

const TOOLS = [
  {
    name: "inspect_screenshot",
    description: "Inspect a screenshot before upload. Returns the image with its filename, MIME type, size, and SHA-256 hash.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" } },
      required: ["path"],
    },
  },
  {
    name: "upload_artifact",
    description:
      "Upload a screenshot or video from the task evidence directory and attach it to the work order.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string" },
        title: { type: "string" },
        posterPath: {
          type: "string",
          description:
            "Inspected PNG, JPEG, or WebP poster to show inline for a video artifact.",
        },
      },
      required: ["path"],
    },
  },
  {
    name: "report_visual_evidence_unavailable",
    description:
      "Record why required visual evidence could not be produced after trying an isolated preview and a Playwright capture.",
    inputSchema: {
      type: "object",
      properties: {
        reason: { type: "string" },
        attempts: {
          type: "array",
          description:
            "Concrete preview and Playwright attempts. Add an upload attempt when upload fails.",
          items: {
            type: "object",
            properties: {
              type: {
                type: "string",
                enum: ["preview", "playwright", "upload"],
              },
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
];

let replyFormat = "ndjson";

function writeMessage(message) {
  const encoded = JSON.stringify(message);
  if (replyFormat === "lsp") {
    const payload = Buffer.from(encoded, "utf8");
    process.stdout.write(`Content-Length: ${payload.length}\r\n\r\n`);
    return process.stdout.write(payload);
  }
  process.stdout.write(`${encoded}\n`);
}

async function handleRequest(message) {
  const { id, method, params } = message;
  if (method === "initialize") {
    writeMessage({
      jsonrpc: "2.0",
      id,
      result: {
        protocolVersion: "2024-11-05",
        capabilities: { tools: {} },
        serverInfo: { name: "superplane", version: "1.0.0" },
      },
    });
    return;
  }
  if (
    method === "notifications/initialized" ||
    method === "notifications/cancelled"
  )
    return;
  if (method === "ping")
    return writeMessage({ jsonrpc: "2.0", id, result: {} });
  if (method === "tools/list")
    return writeMessage({ jsonrpc: "2.0", id, result: { tools: TOOLS } });
  if (method !== "tools/call") {
    if (id != null)
      writeMessage({
        jsonrpc: "2.0",
        id,
        error: { code: -32601, message: `Unknown method: ${method}` },
      });
    return;
  }
  try {
    const name = params && params.name;
    const args = (params && params.arguments) || {};
    let result;
    if (name === "inspect_screenshot") result = inspectScreenshot(args);
    else if (name === "upload_artifact") result = await uploadArtifact(args);
    else if (name === "report_visual_evidence_unavailable")
      result = reportVisualEvidenceUnavailable(args);
    else throw new Error(`Unknown tool: ${name}`);
    const content = result.content || [{ type: "text", text: JSON.stringify(result) }];
    const structuredContent = result.structuredContent || result;
    writeMessage({
      jsonrpc: "2.0",
      id,
      result: {
        content,
        structuredContent,
      },
    });
  } catch (error) {
    writeMessage({
      jsonrpc: "2.0",
      id,
      result: {
        content: [
          {
            type: "text",
            text: error && error.message ? error.message : String(error),
          },
        ],
        isError: true,
      },
    });
  }
}

function parseFrames(buffer) {
  const messages = [];
  let rest = buffer;
  while (rest.length > 0) {
    rest = rest.subarray(
      rest.findIndex((byte) => ![9, 10, 13, 32].includes(byte)) < 0
        ? rest.length
        : rest.findIndex((byte) => ![9, 10, 13, 32].includes(byte)),
    );
    if (!rest.length) break;
    const headerEnd = rest.indexOf("\r\n\r\n");
    if (
      /^content-length:/i.test(
        rest.toString("utf8", 0, Math.min(rest.length, 16)),
      )
    ) {
      if (headerEnd < 0) break;
      const match = rest
        .subarray(0, headerEnd)
        .toString("utf8")
        .match(/Content-Length:\s*(\d+)/i);
      if (!match) {
        rest = rest.subarray(headerEnd + 4);
        continue;
      }
      const length = Number(match[1]);
      if (rest.length < headerEnd + 4 + length) break;
      replyFormat = "lsp";
      messages.push(
        JSON.parse(
          rest.subarray(headerEnd + 4, headerEnd + 4 + length).toString("utf8"),
        ),
      );
      rest = rest.subarray(headerEnd + 4 + length);
      continue;
    }
    const newline = rest.indexOf("\n");
    if (newline < 0) break;
    const line = rest.subarray(0, newline).toString("utf8").trim();
    rest = rest.subarray(newline + 1);
    if (line) messages.push(JSON.parse(line));
  }
  return { messages, rest };
}

function start() {
  writeManifest(readManifest());
  let buffer = Buffer.alloc(0);
  process.stdin.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    let parsed;
    try {
      parsed = parseFrames(buffer);
    } catch (error) {
      process.stderr.write(`${error.message}\n`);
      return;
    }
    buffer = parsed.rest;
    for (const message of parsed.messages) void handleRequest(message);
  });
}

if (require.main === module) start();

module.exports = {
  MAX_ARTIFACT_BYTES,
  MAX_INSPECTABLE_SCREENSHOT_BYTES,
  artifactPaths,
  inspectScreenshot,
  readManifest,
  reportVisualEvidenceUnavailable,
  resolveArtifactFile,
  uploadArtifact,
  writeManifest,
};
