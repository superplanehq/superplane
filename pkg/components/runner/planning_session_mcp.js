#!/usr/bin/env node
"use strict";

const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
const {
  isCompactStatusText,
} = require("./analysis_protocol");
const { MAX_ATTACHMENT_BYTES } = require("./attachment_limit");

const MAX_INSPECTABLE_ATTACHMENT_BYTES = MAX_ATTACHMENT_BYTES;

/**
 * Stdio MCP server for task refinement.
 * Talks to SuperPlane with SUPERPLANE_BASE_URL + SUPERPLANE_RUN_TOKEN.
 */

function readEnv(name) {
  const value = String(process.env[name] || "").trim();
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

async function requestJSON(method, path, body) {
  const baseURL = readEnv("SUPERPLANE_BASE_URL").replace(/\/$/, "");
  const token = readEnv("SUPERPLANE_RUN_TOKEN");
  const headers = {
    Authorization: `Bearer ${token}`,
    Accept: "application/json",
  };
  const options = { method, headers };
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    options.body = JSON.stringify(body);
  }
  const response = await fetch(`${baseURL}${path}`, options);
  const text = await response.text();
  let parsed = {};
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = { message: text };
    }
  }
  if (!response.ok) {
    const message =
      parsed.message || parsed.error || text || `HTTP ${response.status}`;
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }
  return parsed;
}

function surveyQuestions(input) {
  const raw = input && Array.isArray(input.questions) ? input.questions : [];
  return raw.map((question) => ({
    prompt: String((question && question.prompt) || "").trim(),
    options: Array.isArray(question && question.options)
      ? question.options
          .map((option) => String(option || "").trim())
          .filter(Boolean)
      : [],
  }));
}

async function proposeSurvey(input) {
  return requestJSON("POST", "/api/v1/runner/planning-sessions/surveys", {
    questions: surveyQuestions(input),
  });
}

function analysisOutputPaths(env = process.env) {
  return {
    spec: String(env.SUPERPLANE_ANALYSIS_SPEC_FILE || "/tmp/spec.md"),
    score: String(
      env.SUPERPLANE_ANALYSIS_SCORE_FILE || "/tmp/intake-analysis.json",
    ),
  };
}

function writeAnalysisOutputs({ spec, score, summary }, env = process.env) {
  const paths = analysisOutputPaths(env);
  try {
    if (spec != null) {
      fs.writeFileSync(paths.spec, spec);
    }
    if (score != null && Number.isFinite(Number(score))) {
      let existing = {};
      try {
        existing = JSON.parse(fs.readFileSync(paths.score, "utf8"));
      } catch (_err) {
        existing = {};
      }
      const reasons = Array.isArray(existing.reasons) ? existing.reasons : [];
      fs.writeFileSync(
        paths.score,
        `${JSON.stringify({
          score: Math.round(Number(score) * 20),
          summary:
            summary != null ? String(summary) : String(existing.summary || ""),
          reasons,
        })}\n`,
      );
    }
  } catch (_err) {
    // Publish already succeeded. The exit graph reads these files when it can.
  }
}

function scoreField(input, name) {
  const raw = input && input[name];
  if (!raw || typeof raw !== "object") {
    throw new Error(`${name} is required`);
  }
  const score = Number(raw.score);
  if (!Number.isFinite(score)) {
    throw new Error(`${name} score is required`);
  }
  const summary = String(raw.summary || "").trim();
  if (!summary) {
    throw new Error(`${name} summary is required`);
  }
  return { score, summary };
}

async function proposeUpdate(input) {
  const body = {};
  if (input && input.scores) {
    body.scores = {
      clarity: scoreField(input.scores, "clarity"),
      complexity: scoreField(input.scores, "complexity"),
      verifiability: scoreField(input.scores, "verifiability"),
    };
  }
  const spec = unwrapMarkdown(input && input.spec);
  if (spec) {
    body.spec = spec;
  }
  if (input && input.survey) {
    body.survey = { questions: surveyQuestions(input.survey) };
  }
  if (!body.scores && !body.spec && !body.survey) {
    throw new Error("scores, spec, or survey is required");
  }
  const result = await requestJSON("POST", "/api/v1/runner/planning-sessions/updates", body);
  if (body.spec) {
    writeAnalysisOutputs({ spec: body.spec });
  }
  return result;
}

function unwrapMarkdown(value) {
  const text = String(value || "").trim();
  if (!text.startsWith('"')) {
    return text;
  }
  try {
    const decoded = JSON.parse(text);
    if (typeof decoded === "string" && decoded.trim()) {
      return decoded;
    }
  } catch (_err) {
    // The value is not a JSON string.
  }
  return text;
}

async function proposeSpec(input) {
  const body = unwrapMarkdown(input && input.body);
  if (!body) {
    throw new Error("body is required");
  }
  const result = await requestJSON(
    "POST",
    "/api/v1/runner/planning-sessions/specs",
    { body },
  );
  writeAnalysisOutputs({ spec: body });
  return result;
}

function scoreInput(input) {
  const score = Number(input && input.score);
  if (!Number.isFinite(score)) {
    throw new Error("score is required");
  }
  const summary = String((input && input.summary) || "").trim();
  return { score, summary };
}

// Clarity: how well the task is defined. Publishes the clarity check only.
async function proposeClarity(input) {
  const { score, summary } = scoreInput(input);
  return requestJSON("POST", "/api/v1/runner/planning-sessions/clarity", {
    score,
    summary,
  });
}

// Confidence: how likely a coding agent completes the task in one run. The
// exit graph reads the score file as agent fit, so only Confidence writes it.
async function proposeConfidence(input) {
  const { score, summary } = scoreInput(input);
  const result = await requestJSON(
    "POST",
    "/api/v1/runner/planning-sessions/confidence",
    {
      score,
      summary,
    },
  );
  writeAnalysisOutputs({ score, summary });
  return result;
}

function currentActivityID() {
  return String(process.env.SUPERPLANE_ACTIVITY_ID || "").trim() || undefined;
}

function requiredEnv(name, env = process.env) {
  const value = String(env[name] || "").trim();
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

function sniffImageMime(bytes) {
  if (
    bytes.length >= 8 &&
    bytes[0] === 0x89 &&
    bytes[1] === 0x50 &&
    bytes[2] === 0x4e &&
    bytes[3] === 0x47
  ) {
    return "image/png";
  }
  if (bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) {
    return "image/jpeg";
  }
  if (bytes.length >= 6) {
    const header = bytes.toString("ascii", 0, 6);
    if (header === "GIF87a" || header === "GIF89a") {
      return "image/gif";
    }
  }
  if (
    bytes.length >= 12 &&
    bytes.toString("ascii", 0, 4) === "RIFF" &&
    bytes.toString("ascii", 8, 12) === "WEBP"
  ) {
    return "image/webp";
  }
  return "";
}

function resolveAttachmentFile(inputPath, env = process.env) {
  const candidate = String(inputPath || "").trim();
  if (!candidate) {
    throw new Error("path is required");
  }
  const taskDir = path.resolve(requiredEnv("SUPERPLANE_TASK_DIR", env));
  const attachmentsDir = path.join(taskDir, "attachments");
  fs.mkdirSync(attachmentsDir, { recursive: true });
  const attachmentsInfo = fs.lstatSync(attachmentsDir);
  if (attachmentsInfo.isSymbolicLink() || !attachmentsInfo.isDirectory()) {
    throw new Error("task attachments directory must be a regular directory");
  }
  const attachmentsRoot = fs.realpathSync(attachmentsDir);
  const expanded = candidate
    .replace(/\$\{SUPERPLANE_TASK_DIR\}/g, taskDir)
    .replace(/\$SUPERPLANE_TASK_DIR/g, taskDir);
  const absolute = path.isAbsolute(expanded)
    ? path.resolve(expanded)
    : path.resolve(attachmentsDir, expanded);
  const fileInfo = fs.lstatSync(absolute);
  if (fileInfo.isSymbolicLink() || !fileInfo.isFile()) {
    throw new Error("attachment path must be a regular file");
  }
  const resolved = fs.realpathSync(absolute);
  if (
    resolved !== attachmentsRoot &&
    !resolved.startsWith(`${attachmentsRoot}${path.sep}`)
  ) {
    throw new Error("attachment path must be inside the task attachments directory");
  }
  if (fileInfo.size <= 0) {
    throw new Error("attachment file is empty");
  }
  if (fileInfo.size > MAX_INSPECTABLE_ATTACHMENT_BYTES) {
    throw new Error(`attachment exceeds ${MAX_INSPECTABLE_ATTACHMENT_BYTES} bytes`);
  }
  return {
    absolute: resolved,
    filename: path.basename(resolved),
    sizeBytes: fileInfo.size,
    root: attachmentsRoot,
  };
}

function inspectAttachment(input, env = process.env) {
  const file = resolveAttachmentFile(input && input.path, env);
  const bytes = fs.readFileSync(file.absolute);
  const mimeType = sniffImageMime(bytes);
  if (!mimeType) {
    throw new Error("attachment must be a PNG, JPEG, GIF, or WebP file");
  }
  const metadata = {
    path: path.relative(file.root, file.absolute),
    filename: file.filename,
    mimeType,
    sizeBytes: bytes.length,
    sha256: crypto.createHash("sha256").update(bytes).digest("hex"),
  };
  return {
    content: [
      { type: "text", text: JSON.stringify(metadata) },
      { type: "image", data: bytes.toString("base64"), mimeType },
    ],
    structuredContent: metadata,
  };
}

function toolCallResult(result) {
  if (result && Array.isArray(result.content)) {
    return {
      content: result.content,
      structuredContent: result.structuredContent,
    };
  }
  return {
    content: [{ type: "text", text: JSON.stringify(result) }],
    structuredContent: result,
  };
}

async function recordAgentMessage(text) {
  const body = String(text || "").trim();
  if (!body || isCompactStatusText(body)) {
    return { status: "ignored" };
  }
  return requestJSON(
    "POST",
    "/api/v1/runner/planning-sessions/agent-messages",
    {
      text: body,
      activity_id: currentActivityID(),
    },
  );
}

const UPDATE_TOOL = {
  name: "propose_update",
  description:
    "Publish scores, the specification, and an optional survey in one call. Pass scores as clarity, complexity, and verifiability. Each score is an integer from 1 through 3 with a one-sentence summary. Pass spec as the full markdown body. Pass survey only when you ask a question. The first plan turn must include scores.",
  inputSchema: {
    type: "object",
    properties: {
      scores: {
        type: "object",
        properties: {
          clarity: {
            type: "object",
            properties: {
              score: { type: "number", description: "Integer from 1 through 3." },
              summary: { type: "string", description: "One sentence." },
            },
            required: ["score", "summary"],
          },
          complexity: {
            type: "object",
            properties: {
              score: { type: "number", description: "Integer from 1 through 3." },
              summary: { type: "string", description: "One sentence." },
            },
            required: ["score", "summary"],
          },
          verifiability: {
            type: "object",
            properties: {
              score: { type: "number", description: "Integer from 1 through 3." },
              summary: { type: "string", description: "One sentence." },
            },
            required: ["score", "summary"],
          },
        },
        required: ["clarity", "complexity", "verifiability"],
      },
      spec: { type: "string", description: "Full specification markdown." },
      survey: {
        type: "object",
        properties: {
          questions: {
            type: "array",
            items: {
              type: "object",
              properties: {
                prompt: { type: "string" },
                options: { type: "array", items: { type: "string" } },
              },
              required: ["prompt", "options"],
            },
          },
        },
      },
    },
  },
};

const TOOLS = [
  {
    name: "inspect_attachment",
    description:
      "Inspect a user image from the task attachments directory. Returns the image so you can see it. Call this for every PNG, JPEG, GIF, or WebP user image. Do not use OCR or the file command.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" } },
      required: ["path"],
    },
  },
];

let replyFormat = "ndjson";

function writeMessage(message) {
  const encoded = JSON.stringify(message);
  if (replyFormat === "lsp") {
    const payload = Buffer.from(encoded, "utf8");
    process.stdout.write(`Content-Length: ${payload.length}\r\n\r\n`);
    process.stdout.write(payload);
    return;
  }
  process.stdout.write(`${encoded}\n`);
}

function sendResult(id, result) {
  writeMessage({ jsonrpc: "2.0", id, result });
}

function sendError(id, code, message) {
  writeMessage({ jsonrpc: "2.0", id, error: { code, message } });
}

async function handleRequest(message) {
  const { id, method, params } = message;
  if (method === "initialize") {
    sendResult(id, {
      protocolVersion: "2024-11-05",
      capabilities: { tools: {} },
      serverInfo: { name: "superplane", version: "1.0.0" },
    });
    return;
  }
  if (
    method === "notifications/initialized" ||
    method === "notifications/cancelled"
  ) {
    return;
  }
  if (method === "ping") {
    sendResult(id, {});
    return;
  }
  if (method === "tools/list") {
    sendResult(id, { tools: planningTools() });
    return;
  }
  if (method === "tools/call") {
    const name = params && params.name;
    const args = (params && params.arguments) || {};
    try {
      let result;
      if (name === "propose_update") {
        result = await proposeUpdate(args);
      } else if (name === "inspect_attachment") {
        result = inspectAttachment(args);
      } else {
        sendError(id, -32601, `Unknown tool: ${name}`);
        return;
      }
      sendResult(id, toolCallResult(result));
    } catch (err) {
      sendResult(id, {
        content: [
          {
            type: "text",
            text: err && err.message ? err.message : String(err),
          },
        ],
        isError: true,
      });
    }
    return;
  }
  if (id != null) {
    sendError(id, -32601, `Unknown method: ${method}`);
  }
}

const CONTENT_LENGTH_HEADER = "content-length:";

function isContentLengthPrefix(buffer) {
  const peek = buffer
    .toString(
      "utf8",
      0,
      Math.min(buffer.length, CONTENT_LENGTH_HEADER.length),
    )
    .toLowerCase();
  return (
    CONTENT_LENGTH_HEADER.startsWith(peek) ||
    peek.startsWith(CONTENT_LENGTH_HEADER)
  );
}

function parseFrames(buffer) {
  const messages = [];
  let rest = skipASCIIWhitespace(buffer);
  while (rest.length > 0) {
    if (isContentLengthPrefix(rest)) {
      if (rest.length < CONTENT_LENGTH_HEADER.length) {
        break;
      }
      const parsed = parseContentLengthFrame(rest);
      if (!parsed) {
        break;
      }
      replyFormat = "lsp";
      if (parsed.message) {
        messages.push(parsed.message);
      }
      rest = skipASCIIWhitespace(parsed.rest);
      continue;
    }
    if (rest[0] === 0x7b) {
      const parsed = parseNDJSONFrame(rest);
      if (!parsed) {
        break;
      }
      replyFormat = "ndjson";
      if (parsed.message) {
        messages.push(parsed.message);
      }
      rest = skipASCIIWhitespace(parsed.rest);
      continue;
    }
    rest = rest.slice(1);
  }
  return { messages, rest };
}

function skipASCIIWhitespace(buffer) {
  let index = 0;
  while (
    index < buffer.length &&
    (buffer[index] === 0x09 ||
      buffer[index] === 0x0a ||
      buffer[index] === 0x0d ||
      buffer[index] === 0x20)
  ) {
    index += 1;
  }
  return index === 0 ? buffer : buffer.slice(index);
}

function parseContentLengthFrame(buffer) {
  const headerEnd = buffer.indexOf("\r\n\r\n");
  if (headerEnd < 0) {
    return null;
  }
  const header = buffer.slice(0, headerEnd).toString("utf8");
  const match = header.match(/Content-Length:\s*(\d+)/i);
  if (!match) {
    return { message: null, rest: buffer.slice(headerEnd + 4) };
  }
  const length = Number(match[1]);
  const bodyStart = headerEnd + 4;
  if (buffer.length < bodyStart + length) {
    return null;
  }
  const body = buffer.slice(bodyStart, bodyStart + length).toString("utf8");
  let message = null;
  try {
    message = JSON.parse(body);
  } catch {
    // Ignore malformed frames.
  }
  return { message, rest: buffer.slice(bodyStart + length) };
}

function parseNDJSONFrame(buffer) {
  const newline = buffer.indexOf(0x0a);
  if (newline < 0) {
    return null;
  }
  const line = buffer.slice(0, newline).toString("utf8").trim();
  let message = null;
  if (line) {
    try {
      message = JSON.parse(line);
    } catch {
      // Ignore malformed frames.
    }
  }
  return { message, rest: buffer.slice(newline + 1) };
}

async function main() {
  let buffer = Buffer.alloc(0);
  process.stdin.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    const parsed = parseFrames(buffer);
    buffer = parsed.rest;
    for (const message of parsed.messages) {
      Promise.resolve(handleRequest(message)).catch((err) => {
        if (message && message.id != null) {
          sendError(
            message.id,
            -32603,
            err && err.message ? err.message : String(err),
          );
        }
      });
    }
  });
}

if (require.main === module) {
  main();
}

function planningTools() {
  return [UPDATE_TOOL, TOOLS.find((tool) => tool.name === "inspect_attachment")];
}

module.exports = {
  proposeUpdate,
  proposeSpec,
  proposeClarity,
  proposeConfidence,
  proposeSurvey,
  inspectAttachment,
  recordAgentMessage,
  surveyQuestions,
  TOOLS,
  planningTools,
  writeAnalysisOutputs,
  analysisOutputPaths,
  parseFrames,
  MAX_INSPECTABLE_ATTACHMENT_BYTES,
};
