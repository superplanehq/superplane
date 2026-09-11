#!/usr/bin/env node
"use strict";

const fs = require("fs");

/**
 * Stdio MCP server for Create with an Agent.
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
    const message = parsed.message || parsed.error || text || `HTTP ${response.status}`;
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }
  return parsed;
}

async function proposeDraft(input) {
  const title = String((input && input.title) || "").trim();
  const description = String((input && input.description) || "").trim();
  if (!title) {
    throw new Error("title is required");
  }
  if (!description) {
    throw new Error("description is required");
  }
  return requestJSON("POST", "/api/v1/runner/planning-sessions/drafts", {
    title,
    description,
  });
}

function surveyQuestions(input) {
  const raw = input && Array.isArray(input.questions) ? input.questions : [];
  return raw.map((question) => ({
    prompt: String((question && question.prompt) || "").trim(),
    options: Array.isArray(question && question.options)
      ? question.options.map((option) => String(option || "").trim()).filter(Boolean)
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
    score: String(env.SUPERPLANE_ANALYSIS_SCORE_FILE || "/tmp/intake-analysis.json"),
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
          summary: summary != null ? String(summary) : String(existing.summary || ""),
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
  const result = await requestJSON("POST", "/api/v1/runner/planning-sessions/specs", { body });
  writeAnalysisOutputs({ spec: body });
  return result;
}

async function proposeConfidence(input) {
  const score = Number(input && input.score);
  if (!Number.isFinite(score)) {
    throw new Error("score is required");
  }
  const summary = String((input && input.summary) || "").trim();
  const result = await requestJSON("POST", "/api/v1/runner/planning-sessions/confidence", {
    score,
    summary,
  });
  writeAnalysisOutputs({ score, summary });
  return result;
}

const TOOLS = [
  {
    name: "propose_spec",
    description: "Publish the full spec.md markdown for the open task. Include the title, Executive summary, and plan. Do not change the original request. Call this after you write the specification.",
    inputSchema: {
      type: "object",
      properties: {
        body: { type: "string" },
      },
      required: ["body"],
    },
  },
  {
    name: "propose_confidence",
    description:
      "Publish the 0 through 5 confidence score and one sentence that explains why that score fits. Say how suitable the work is for an agent. Do not write a test or an acceptance check.",
    inputSchema: {
      type: "object",
      properties: {
        score: {
          type: "number",
          description: "Confidence from 0 through 5.",
        },
        summary: {
          type: "string",
          description:
            "One sentence that explains the score. Example: This is a small bug fix with clear reproduction steps and an example in the repository, so an agent can complete it.",
        },
      },
      required: ["score", "summary"],
    },
  },
  {
    name: "propose_draft",
    description: "Show a draft task on the right only when the user asked for a task in this turn. Title and description are required. Description must include the user's request and constraints. The user confirms or skips. Do not create the task. Do not propose another draft unless the user asks.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string" },
        description: { type: "string" },
      },
      required: ["title", "description"],
    },
  },
  {
    name: "survey",
    description:
      "Ask the person a multiple-choice question when the task is unclear or two valid readings exist. Use 2 to 4 options. Then stop. Do not ask the same question in chat.",
    inputSchema: {
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
      required: ["questions"],
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
  if (method === "notifications/initialized" || method === "notifications/cancelled") {
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
      if (name === "propose_draft") {
        result = await proposeDraft(args);
      } else if (name === "propose_spec") {
        result = await proposeSpec(args);
      } else if (name === "propose_confidence") {
        result = await proposeConfidence(args);
      } else if (name === "survey") {
        result = await proposeSurvey(args);
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
        content: [{ type: "text", text: err && err.message ? err.message : String(err) }],
        isError: true,
      });
    }
    return;
  }
  if (id != null) {
    sendError(id, -32601, `Unknown method: ${method}`);
  }
}

function parseFrames(buffer) {
  const messages = [];
  let rest = skipASCIIWhitespace(buffer);
  while (rest.length > 0) {
    const peek = rest.toString("utf8", 0, Math.min(rest.length, 16));
    if (/^content-length:/i.test(peek)) {
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
  while (index < buffer.length && (buffer[index] === 0x09 || buffer[index] === 0x0a || buffer[index] === 0x0d || buffer[index] === 0x20)) {
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
          sendError(message.id, -32603, err && err.message ? err.message : String(err));
        }
      });
    }
  });
}

if (require.main === module) {
  main();
}

module.exports = {
  proposeDraft,
  proposeSpec,
  proposeConfidence,
  proposeSurvey,
  surveyQuestions,
  TOOLS,
  writeAnalysisOutputs,
  analysisOutputPaths,
};
