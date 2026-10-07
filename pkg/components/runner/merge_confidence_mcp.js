#!/usr/bin/env node
"use strict";

/**
 * Stdio MCP server for merge confidence.
 * Talks to SuperPlane with SUPERPLANE_BASE_URL + SUPERPLANE_MERGE_CONFIDENCE_TOKEN.
 */

function readEnv(name) {
  const value = String(process.env[name] || "").trim();
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

async function requestJSON(method, requestPath, body) {
  const baseURL = readEnv("SUPERPLANE_BASE_URL").replace(/\/$/, "");
  const token = readEnv("SUPERPLANE_MERGE_CONFIDENCE_TOKEN");
  const headers = {
    Authorization: `Bearer ${token}`,
    Accept: "application/json",
    "Content-Type": "application/json",
  };
  const response = await fetch(`${baseURL}${requestPath}`, {
    method,
    headers,
    body: JSON.stringify(body),
  });
  const text = await response.text();
  let parsed = {};
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = { message: text.trim() };
    }
  }
  if (!response.ok) {
    const message = parsed.message || parsed.error || text.trim() || `HTTP ${response.status}`;
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }
  return parsed;
}

async function reportMergeCheck(input) {
  const check = String((input && input.check) || "").trim().toLowerCase();
  if (!check) {
    throw new Error("check is required");
  }
  const score = Number(input && input.score);
  if (!Number.isFinite(score)) {
    throw new Error("score is required");
  }
  const summary = String((input && input.summary) || "").trim();
  if (!summary) {
    throw new Error("summary is required");
  }
  return requestJSON("POST", "/api/v1/runner/merge-confidence/checks", {
    check,
    score,
    summary,
  });
}

const TOOLS = [
  {
    name: "report_merge_check",
    description:
      "Publish one merge confidence check for this pull request. Call this once for the check in the prompt. check is the id from the Merge check line. score is an integer from 1 through 5. summary is one sentence.",
    inputSchema: {
      type: "object",
      properties: {
        check: {
          type: "string",
          description: "The id from the Merge check line, such as risk or api-latency.",
        },
        score: {
          type: "number",
          description: "Integer from 1 through 5.",
        },
        summary: {
          type: "string",
          description: "One sentence. Do not repeat the score.",
        },
      },
      required: ["check", "score", "summary"],
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
      if (name !== "report_merge_check") {
        sendError(id, -32601, `Unknown tool: ${name}`);
        return;
      }
      const result = await reportMergeCheck(args);
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
  let rest = buffer;
  while (rest.length > 0) {
    const skip = rest.findIndex((byte) => byte !== 9 && byte !== 10 && byte !== 13 && byte !== 32);
    if (skip < 0) {
      rest = rest.subarray(rest.length);
      break;
    }
    rest = rest.subarray(skip);
    const headerPeek = rest.toString("utf8", 0, Math.min(rest.length, 16));
    if (/^content-length:/i.test(headerPeek)) {
      const headerEnd = rest.indexOf("\r\n\r\n");
      if (headerEnd < 0) {
        break;
      }
      const match = rest.subarray(0, headerEnd).toString("utf8").match(/Content-Length:\s*(\d+)/i);
      if (!match) {
        rest = rest.subarray(headerEnd + 4);
        continue;
      }
      const length = Number(match[1]);
      if (rest.length < headerEnd + 4 + length) {
        break;
      }
      replyFormat = "lsp";
      try {
        messages.push(JSON.parse(rest.subarray(headerEnd + 4, headerEnd + 4 + length).toString("utf8")));
      } catch (_err) {
        // Ignore a malformed frame.
      }
      rest = rest.subarray(headerEnd + 4 + length);
      continue;
    }
    const newline = rest.indexOf("\n");
    if (newline < 0) {
      break;
    }
    const line = rest.subarray(0, newline).toString("utf8").trim();
    rest = rest.subarray(newline + 1);
    if (!line) {
      continue;
    }
    replyFormat = "ndjson";
    try {
      messages.push(JSON.parse(line));
    } catch (_err) {
      // Ignore a malformed frame.
    }
  }
  return { messages, rest };
}

function start() {
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
  start();
}

module.exports = {
  TOOLS,
  reportMergeCheck,
  parseFrames,
};
