#!/usr/bin/env node
"use strict";

const fs = require("fs");
const { isCompactStatusText } = require("./analysis_protocol");

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

async function proposeSpec(input) {
  const body = String((input && input.body) || "").trim();
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

// Splits one task off the draft under refinement. SuperPlane creates the
// draft, links it to this session, and shows it in the chat.
async function createTask(input) {
  const title = String((input && input.title) || "").trim();
  if (!title) {
    throw new Error("title is required");
  }
  const description = String((input && input.description) || "").trim();
  if (!description) {
    throw new Error("description is required");
  }
  return requestJSON("POST", "/api/v1/runner/planning-sessions/tasks", {
    title,
    description,
    activity_id: currentActivityID(),
  });
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

const TOOLS = [
  {
    name: "propose_spec",
    description: "Publish the specification markdown for the open task. Call this before you stop whenever you write or update a specification this turn. Do not leave a written plan unpublished. Pass the full markdown body.",
    inputSchema: {
      type: "object",
      properties: {
        body: { type: "string" },
      },
      required: ["body"],
    },
  },
  {
    name: "propose_clarity",
    description:
      "Publish the 1 through 5 Clarity score: how well the task is defined. Call this every turn. Write the summary the way the task prompt asks. You may call this without propose_spec when only the score changes.",
    inputSchema: {
      type: "object",
      properties: {
        score: {
          type: "number",
          description: "Clarity from 1 through 5.",
        },
        summary: {
          type: "string",
          description: "Short Clarity summary for the user. Follow the task prompt for length and shape.",
        },
      },
      required: ["score", "summary"],
    },
  },
  {
    name: "propose_confidence",
    description:
      "Publish the 1 through 5 Confidence score: how likely a coding agent completes this task in one run without steering. Call this every turn. Write the summary the way the task prompt asks. You may call this without propose_spec when only the score changes.",
    inputSchema: {
      type: "object",
      properties: {
        score: {
          type: "number",
          description: "Confidence from 1 through 5.",
        },
        summary: {
          type: "string",
          description: "Short Confidence summary for the user. Follow the task prompt for length and shape.",
        },
      },
      required: ["score", "summary"],
    },
  },
  {
    name: "survey",
    description:
      "Ask one multiple-choice question. Call this only when the task prompt says to ask. Use 2 to 4 short everyday options. Then stop and wait. Do not ask the same question in chat.",
    inputSchema: {
      type: "object",
      properties: {
        questions: {
          type: "array",
          description:
            "A JSON array of question objects. Do not pass XML or a JSON-encoded string.",
          items: {
            type: "object",
            properties: {
              prompt: { type: "string", description: "One plain question." },
              options: {
                type: "array",
                items: { type: "string" },
                description: "Short everyday options. Under 12 words each.",
              },
            },
            required: ["prompt", "options"],
          },
        },
      },
      required: ["questions"],
    },
  },
  {
    name: "create_task",
    description:
      "Split one part of this task into a new draft task in the same backlog. Call this only after the user confirms the split in chat or in a survey answer. One call per task. Do not create a task that this session already created. After you create the tasks, narrow this task to the part that stays, then call propose_spec, propose_clarity, and propose_confidence again.",
    inputSchema: {
      type: "object",
      properties: {
        title: {
          type: "string",
          description: "Short imperative title for the new task. Under 12 words.",
        },
        description: {
          type: "string",
          description:
            "Markdown description of the new task. Self-contained: a reader who has not seen this chat must understand the goal, the scope, and what done looks like. Do not refer to this conversation.",
        },
      },
      required: ["title", "description"],
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
    sendResult(id, { tools: TOOLS });
    return;
  }
  if (method === "tools/call") {
    const name = params && params.name;
    const args = (params && params.arguments) || {};
    try {
      let result;
      if (name === "propose_spec") {
        result = await proposeSpec(args);
      } else if (name === "propose_clarity") {
        result = await proposeClarity(args);
      } else if (name === "propose_confidence") {
        result = await proposeConfidence(args);
      } else if (name === "survey") {
        result = await proposeSurvey(args);
      } else if (name === "create_task") {
        result = await createTask(args);
      } else {
        sendError(id, -32601, `Unknown tool: ${name}`);
        return;
      }
      sendResult(id, {
        content: [{ type: "text", text: JSON.stringify(result) }],
        structuredContent: result,
      });
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

module.exports = {
  proposeSpec,
  proposeClarity,
  proposeConfidence,
  proposeSurvey,
  createTask,
  recordAgentMessage,
  surveyQuestions,
  TOOLS,
  writeAnalysisOutputs,
  analysisOutputPaths,
  parseFrames,
};
