"use strict";

const { spawn } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const assert = require("node:assert/strict");

const { analysisProtocol, withAnalysisContinuation } = require("./analysis_protocol");
const { writeAnalysisOutputs } = require("./planning_session_mcp");

test("analysis protocol covers publish tools and hides chat dumps", () => {
  const pack = analysisProtocol();
  assert.match(pack, /propose_spec/);
  assert.match(pack, /propose_confidence/);
  assert.match(pack, /how suitable the work is for an agent/);
  assert.match(pack, /Do not write a test or an acceptance check/);
  assert.match(pack, /Do not call propose_draft/);
  assert.match(pack, /Do not paste the specification/);
  assert.match(pack, /call survey with 2 to 4 options/);
  assert.match(pack, /If the score is 0 through 3/);
  assert.match(pack, /this is a continuation/);
  assert.doesNotMatch(pack, /check copy/);
  assert.doesNotMatch(pack, /\/tmp\/spec\.md/);
});

test("withAnalysisContinuation prepends prior spec on the first prompt", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "analysis-continuation-"));
  fs.writeFileSync(path.join(dir, "analysis_continuation.md"), "Continue this SuperPlane analysis session.\n");
  assert.equal(
    withAnalysisContinuation(dir, 0, "Analyze the task."),
    "Continue this SuperPlane analysis session.\n\nAnalyze the task.",
  );
  assert.equal(withAnalysisContinuation(dir, 1, "Analyze the task."), "Analyze the task.");
  assert.equal(withAnalysisContinuation(path.join(dir, "missing"), 0, "Analyze the task."), "Analyze the task.");
});

test("writeAnalysisOutputs maps a 0-5 score to the exit-graph percentage", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "analysis-outputs-"));
  const spec = path.join(dir, "spec.md");
  const score = path.join(dir, "score.json");
  writeAnalysisOutputs(
    { spec: "# Add breed\n", score: 4, summary: "The CRUD files already exist." },
    { SUPERPLANE_ANALYSIS_SPEC_FILE: spec, SUPERPLANE_ANALYSIS_SCORE_FILE: score },
  );
  assert.equal(fs.readFileSync(spec, "utf8"), "# Add breed\n");
  assert.deepEqual(JSON.parse(fs.readFileSync(score, "utf8")), {
    score: 80,
    summary: "The CRUD files already exist.",
    reasons: [],
  });
});

test("lists planning tools over newline-delimited JSON-RPC", async () => {
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
  assert.deepEqual(
    replies[1].result.tools.map((tool) => tool.name),
    ["propose_spec", "propose_confidence", "propose_draft", "survey"],
  );
  assert.deepEqual(replies[1].result.tools[0].inputSchema.required, ["body"]);
  assert.match(replies[1].result.tools[1].description, /how suitable the work is for an agent/);
  assert.match(replies[1].result.tools[3].description, /two valid readings exist/);
  assert.match(replies[1].result.tools[1].inputSchema.properties.summary.description, /explains the score/);
  assert.doesNotMatch(replies[1].result.tools[1].description, /check copy/);
  assert.deepEqual(replies[1].result.tools[2].inputSchema.required, ["title", "description"]);
});

test("lists planning tools over Content-Length JSON-RPC", async () => {
  const replies = await exchangeMCP("lsp", [
    { jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {} } },
    { jsonrpc: "2.0", id: 2, method: "tools/list", params: {} },
  ]);
  assert.deepEqual(
    replies[1].result.tools.map((tool) => tool.name),
    ["propose_spec", "propose_confidence", "propose_draft", "survey"],
  );
});

async function exchangeMCP(format, messages) {
  const child = spawn(process.execPath, [path.join(__dirname, "planning_session_mcp.js")], {
    stdio: ["pipe", "pipe", "pipe"],
    env: { ...process.env },
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
