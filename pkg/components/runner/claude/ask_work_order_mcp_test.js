"use strict";

const { spawn } = require("node:child_process");
const path = require("node:path");
const test = require("node:test");
const assert = require("node:assert/strict");
const { mapWaitStatus, normalizeQuestions, normalizeTimeout } = require("./ask_work_order_mcp.js");

test("maps wait statuses for the tool result", () => {
  assert.equal(mapWaitStatus("answered"), "answered");
  assert.equal(mapWaitStatus("cancelled"), "cancelled");
  assert.equal(mapWaitStatus("timed_out"), "no_answer");
  assert.equal(mapWaitStatus("pending"), "no_answer");
});

test("normalizes questions and treats empty options as free text", () => {
  const questions = normalizeQuestions([
    { id: " scope ", prompt: " Where? ", options: ["A", "", "B"] },
    { id: "notes", prompt: "Anything else?" },
  ]);
  assert.deepEqual(questions, [
    { id: "scope", prompt: "Where?", options: ["A", "B"], allow_free_text: false },
    { id: "notes", prompt: "Anything else?", options: [], allow_free_text: true },
  ]);
});

test("rejects an empty question list", () => {
  assert.throws(() => normalizeQuestions([]), /1–10/);
});

test("clamps timeout to the allowed range", () => {
  assert.equal(normalizeTimeout(undefined), 3600);
  assert.equal(normalizeTimeout(120), 120);
  assert.throws(() => normalizeTimeout(10), /timeout_seconds/);
  assert.throws(() => normalizeTimeout(70000), /timeout_seconds/);
});

test("lists ask_work_order over newline-delimited JSON-RPC", async () => {
  const replies = await exchangeMCP("ndjson", [
    {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test", version: "1" } },
    },
    { jsonrpc: "2.0", id: 2, method: "tools/list", params: {} },
  ]);
  assert.equal(replies[0].id, 1);
  assert.equal(replies[0].result.serverInfo.name, "superplane");
  assert.equal(replies[1].result.tools[0].name, "ask_work_order");
});

test("lists ask_work_order over Content-Length JSON-RPC", async () => {
  const replies = await exchangeMCP("lsp", [
    { jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {} } },
    { jsonrpc: "2.0", id: 2, method: "tools/list", params: {} },
  ]);
  assert.equal(replies[1].result.tools[0].name, "ask_work_order");
});

test("lists planning tools when a planning session id is set", async () => {
  const replies = await exchangeMCP(
    "ndjson",
    [
      {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test", version: "1" } },
      },
      { jsonrpc: "2.0", id: 2, method: "tools/list", params: {} },
    ],
    { SUPERPLANE_PLANNING_SESSION_ID: "session-1" },
  );
  assert.deepEqual(
    replies[1].result.tools.map((tool) => tool.name),
    ["wait_for_user", "ask", "propose_draft", "say"],
  );
});

async function exchangeMCP(format, messages, env = {}) {
  const child = spawn(process.execPath, [path.join(__dirname, "ask_work_order_mcp.js")], {
    stdio: ["pipe", "pipe", "pipe"],
    env: { ...process.env, ...env },
  });
  const replies = [];
  let buf = Buffer.alloc(0);
  child.stdout.on("data", (chunk) => {
    buf = Buffer.concat([buf, chunk]);
    const parsed = drainReplies(buf, format);
    buf = parsed.rest;
    replies.push(...parsed.messages);
  });
  for (const message of messages) {
    child.stdin.write(encodeMCP(format, message));
  }
  const started = Date.now();
  while (replies.length < messages.length && Date.now() - started < 2000) {
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  child.kill();
  assert.equal(replies.length, messages.length, `expected ${messages.length} ${format} replies, got ${replies.length}`);
  return replies;
}

function encodeMCP(format, message) {
  const encoded = JSON.stringify(message);
  if (format === "lsp") {
    return `Content-Length: ${Buffer.byteLength(encoded, "utf8")}\r\n\r\n${encoded}`;
  }
  return `${encoded}\n`;
}

function drainReplies(buffer, format) {
  const messages = [];
  let rest = buffer;
  if (format === "lsp") {
    while (true) {
      const headerEnd = rest.indexOf("\r\n\r\n");
      if (headerEnd < 0) {
        return { messages, rest };
      }
      const header = rest.slice(0, headerEnd).toString("utf8");
      const match = header.match(/Content-Length:\s*(\d+)/i);
      if (!match) {
        return { messages, rest };
      }
      const length = Number(match[1]);
      const bodyStart = headerEnd + 4;
      if (rest.length < bodyStart + length) {
        return { messages, rest };
      }
      messages.push(JSON.parse(rest.slice(bodyStart, bodyStart + length).toString("utf8")));
      rest = rest.slice(bodyStart + length);
    }
  }
  const text = rest.toString("utf8");
  const lines = text.split("\n");
  const leftover = lines.pop() || "";
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.startsWith("{")) {
      messages.push(JSON.parse(trimmed));
    }
  }
  return { messages, rest: Buffer.from(leftover, "utf8") };
}
