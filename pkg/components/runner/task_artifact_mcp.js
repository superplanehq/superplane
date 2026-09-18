#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const path = require("node:path");

const MAX_ARTIFACT_BYTES = 100 * 1024 * 1024;
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
  fs.mkdirSync(paths.taskDir, { recursive: true });
  const temporary = `${paths.manifest}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(manifest, null, 2)}\n`, {
    mode: 0o600,
  });
  fs.renameSync(temporary, paths.manifest);
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

async function uploadArtifact(input, env = process.env, fetchImpl = fetch) {
  const file = resolveArtifactFile(input && input.path, env);
  const baseURL = requiredEnv("SUPERPLANE_BASE_URL", env).replace(/\/$/, "");
  const token = requiredEnv("SUPERPLANE_ARTIFACT_TOKEN", env);
  const title = String((input && input.title) || "").trim();
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

  const manifest = readManifest(env);
  const artifacts = manifest.artifacts.filter(
    (item) => item.file_id !== result.file_id,
  );
  artifacts.push(result);
  writeManifest({ status: "captured", reason: "", artifacts }, env);
  return result;
}

function reportVisualEvidenceUnavailable(input, env = process.env) {
  const reason = String((input && input.reason) || "").trim();
  if (!reason) throw new Error("reason is required");
  const attempts = normalizeEvidenceAttempts(input && input.attempts);
  const manifest = readManifest(env);
  const result = {
    status: "unavailable",
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
    name: "upload_artifact",
    description:
      "Upload a screenshot or video from the task evidence directory and attach it to the work order.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" }, title: { type: "string" } },
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
    if (name === "upload_artifact") result = await uploadArtifact(args);
    else if (name === "report_visual_evidence_unavailable")
      result = reportVisualEvidenceUnavailable(args);
    else throw new Error(`Unknown tool: ${name}`);
    writeMessage({
      jsonrpc: "2.0",
      id,
      result: {
        content: [{ type: "text", text: JSON.stringify(result) }],
        structuredContent: result,
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
  artifactPaths,
  readManifest,
  reportVisualEvidenceUnavailable,
  resolveArtifactFile,
  uploadArtifact,
  writeManifest,
};
