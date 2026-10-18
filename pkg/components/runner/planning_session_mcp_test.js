"use strict";

const { spawn } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const assert = require("node:assert/strict");

const { analysisProtocol, withoutEmbeddedAnalysisProtocol, withAnalysisContinuation, isCompactStatusText } = require("./analysis_protocol");
const {
  proposeUpdate,
  proposeSpec,
  proposeClarity,
  proposeConfidence,
  inspectAttachment,
  recordAgentMessage,
  writeAnalysisOutputs,
  parseFrames,
  MAX_INSPECTABLE_ATTACHMENT_BYTES,
} = require("./planning_session_mcp");

const PNG_BYTES = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00]);
const GIF_BYTES = Buffer.from([0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x00]);
const JPEG_BYTES = Buffer.from([0xff, 0xd8, 0xff, 0x00]);

test("analysis protocol always uses the review contract", () => {
  const review = analysisProtocol();
  assert.match(review, /propose_update/);
  assert.doesNotMatch(review, /Call propose_clarity/);
  assert.doesNotMatch(review, /Call propose_confidence/);
  assert.match(review, /Do not call propose_spec, propose_clarity, propose_confidence, or survey/);
  assert.match(review, /integer from 1 through 3/);
  assert.match(review, /Do not use that scale or its thresholds/);

  const ignored = analysisProtocol({
    SUPERPLANE_PLANNING_CLARITY: "false",
    SUPERPLANE_PLANNING_CONFIDENCE: "false",
    SUPERPLANE_PLANNING_REVIEW: "false",
  });
  assert.equal(ignored, review);
});

test("planningTools always exposes propose_update", () => {
  const { planningTools } = require("./planning_session_mcp");
  assert.deepEqual(
    planningTools().map((tool) => tool.name),
    ["propose_update", "inspect_attachment"],
  );
  assert.deepEqual(
    planningTools({
      SUPERPLANE_PLANNING_CLARITY: "false",
      SUPERPLANE_PLANNING_CONFIDENCE: "false",
      SUPERPLANE_PLANNING_REVIEW: "false",
    }).map((tool) => tool.name),
    ["propose_update", "inspect_attachment"],
  );
});

test("analysis protocol covers publish tools and hides chat dumps", () => {
  const pack = analysisProtocol();
  assert.match(pack, /propose_update/);
  assert.match(pack, /weakest sub-parameter/);
  assert.match(pack, /The task prompt owns the judgment|When the two seem to disagree on judgment, the task prompt wins/);
  assert.match(pack, /Use only the analysis tools/);
  assert.match(pack, /Do not paste the published plan/);
  assert.match(pack, /inspect_attachment/);
  assert.match(pack, /Do not use OCR/);
  assert.doesNotMatch(pack, /Call propose_spec when the task prompt says/);
  assert.doesNotMatch(pack, /Talk like a colleague/);
  assert.doesNotMatch(pack, /## Proposed outcome/);
});

test("review user prompt scores three sub-parameters on the 1 through 3 scale", () => {
  const pack = fs.readFileSync(path.join(__dirname, "analysis_user_prompt_review.md"), "utf8");
  assert.match(pack, /Talk like a colleague/);
  assert.match(pack, /## 1\. Research/);
  assert.match(pack, /## 2\. Decide or ask/);
  assert.match(pack, /## 3\. Score Clarity/);
  assert.match(pack, /## 4\. Score Complexity/);
  assert.match(pack, /## 5\. Score Verifiability/);
  assert.match(pack, /## 6\. Write the plan/);
  assert.match(pack, /integer from 1 through 3/);
  assert.doesNotMatch(pack, /1 through 5/);
  assert.doesNotMatch(pack, /## Score Confidence/);
  assert.doesNotMatch(pack, /Confidence summary/);
  assert.match(pack, /derives one Confidence number from the weakest sub-parameter/);
  assert.match(pack, /You never invent the headline number/);
  assert.match(pack, /how well the task is defined/);
  assert.match(pack, /finishes this task in one run/);
  assert.match(pack, /whether the run can prove the change works/);
  assert.match(pack, /### Calibration/);
  assert.match(pack, /Start at 3 for a bounded change that has a pattern in the repository/);
  assert.match(pack, /Finish the research first\. Publish the scores, the plan, and any question together, once, at the end of the turn\./);
  assert.match(pack, /Never publish placeholder or filler text/);
  assert.match(pack, /If Clarity is 1, do not write a specification/);
  assert.match(pack, /Publish the scores alone/);
  assert.match(pack, /If Clarity is 2, write the plan\. Say it is a first pass\./);
  assert.match(pack, /Keep refining until every score is 3/);
  assert.match(pack, /A simple task can reach 3 on every score with no survey/);
  assert.match(pack, /one sentence of 14 words or fewer/);
  assert.match(pack, /about the task, not to the user/);
  assert.match(pack, /name only the weakest sub-parameter/);
  assert.match(pack, /Skip Risks only when every score is 3/);
  assert.match(pack, /Keep this task as one task/);
  assert.match(pack, /ask to update the plan or the scores, or confirm a decision/);
  assert.match(pack, /Do not use a survey to answer a question/);
  assert.match(pack, /Keep each option under 12 words/);
  assert.doesNotMatch(pack, /propose_update/);
  assert.doesNotMatch(pack, /propose_spec/);
});

test("embedded analysis protocol is removed from the task prompt", () => {
  const protocol = analysisProtocol();
  assert.equal(
    withoutEmbeddedAnalysisProtocol(`${protocol}\n\nTask:\nFix retries.`),
    "Task:\nFix retries.",
  );
  assert.equal(
    withoutEmbeddedAnalysisProtocol("Legacy analysis prompt."),
    "Legacy analysis prompt.",
  );
});

test("withAnalysisContinuation prepends prior spec on the first prompt", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "analysis-continuation-"));
  fs.writeFileSync(
    path.join(dir, "analysis_continuation.md"),
    "Continue this SuperPlane analysis session.\n",
  );
  assert.equal(
    withAnalysisContinuation(dir, 0, "Analyze the task."),
    "Continue this SuperPlane analysis session.\n\nAnalyze the task.",
  );
  assert.equal(
    withAnalysisContinuation(dir, 1, "Analyze the task."),
    "Analyze the task.",
  );
  assert.equal(
    withAnalysisContinuation(path.join(dir, "missing"), 0, "Analyze the task."),
    "Analyze the task.",
  );
});

test("writeAnalysisOutputs maps a 0-5 score to the exit-graph percentage", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "analysis-outputs-"));
  const spec = path.join(dir, "spec.md");
  const score = path.join(dir, "score.json");
  writeAnalysisOutputs(
    {
      spec: "# Add breed\n",
      score: 4,
      summary: "The CRUD files already exist.",
    },
    {
      SUPERPLANE_ANALYSIS_SPEC_FILE: spec,
      SUPERPLANE_ANALYSIS_SCORE_FILE: score,
    },
  );
  assert.equal(fs.readFileSync(spec, "utf8"), "# Add breed\n");
  assert.deepEqual(JSON.parse(fs.readFileSync(score, "utf8")), {
    score: 80,
    summary: "The CRUD files already exist.",
    reasons: [],
  });
});

test("proposeUpdate publishes scores spec and survey on one route", async () => {
  const previousBaseURL = process.env.SUPERPLANE_BASE_URL;
  const previousToken = process.env.SUPERPLANE_RUN_TOKEN;
  const previousFetch = global.fetch;
  const calls = [];
  process.env.SUPERPLANE_BASE_URL = "https://superplane.example";
  process.env.SUPERPLANE_RUN_TOKEN = "runner-token";
  global.fetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, text: async () => '{"status":"shown"}' };
  };

  try {
    const result = await proposeUpdate({
      scores: {
        clarity: { score: 4, summary: "Outcome is clear." },
        complexity: { score: 3, summary: "One run can finish." },
        verifiability: { score: 5, summary: "Existing tests cover the change." },
      },
      spec: "# Retry refunds\n",
      survey: { questions: [{ prompt: "Which service?", options: ["Payments", "Billing"] }] },
    });
    assert.deepEqual(result, { status: "shown" });
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/planning-sessions/updates");
    assert.deepEqual(JSON.parse(calls[0].options.body), {
      scores: {
        clarity: { score: 4, summary: "Outcome is clear." },
        complexity: { score: 3, summary: "One run can finish." },
        verifiability: { score: 5, summary: "Existing tests cover the change." },
      },
      spec: "# Retry refunds",
      survey: { questions: [{ prompt: "Which service?", options: ["Payments", "Billing"] }] },
    });
  } finally {
    global.fetch = previousFetch;
    if (previousBaseURL === undefined) delete process.env.SUPERPLANE_BASE_URL;
    else process.env.SUPERPLANE_BASE_URL = previousBaseURL;
    if (previousToken === undefined) delete process.env.SUPERPLANE_RUN_TOKEN;
    else process.env.SUPERPLANE_RUN_TOKEN = previousToken;
  }
});

test("proposeSpec, proposeClarity, and proposeConfidence publish on separate routes", async () => {
  const previousBaseURL = process.env.SUPERPLANE_BASE_URL;
  const previousToken = process.env.SUPERPLANE_RUN_TOKEN;
  const previousFetch = global.fetch;
  const calls = [];
  process.env.SUPERPLANE_BASE_URL = "https://superplane.example";
  process.env.SUPERPLANE_RUN_TOKEN = "runner-token";
  global.fetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, text: async () => '{"status":"shown"}' };
  };

  try {
    const spec = await proposeSpec({ body: "# Retry refunds\n" });
    const clarity = await proposeClarity({
      score: 5,
      summary: "  The plan is ready.  ",
    });
    const confidence = await proposeConfidence({
      score: 4,
      summary: "This issue is a good fit for an agent.",
    });
    assert.deepEqual(spec, { status: "shown" });
    assert.deepEqual(clarity, { status: "shown" });
    assert.deepEqual(confidence, { status: "shown" });
    assert.equal(calls.length, 3);
    assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/planning-sessions/specs");
    assert.deepEqual(JSON.parse(calls[0].options.body), { body: "# Retry refunds" });
    assert.equal(calls[1].url, "https://superplane.example/api/v1/runner/planning-sessions/clarity");
    assert.deepEqual(JSON.parse(calls[1].options.body), {
      score: 5,
      summary: "The plan is ready.",
    });
    assert.equal(calls[2].url, "https://superplane.example/api/v1/runner/planning-sessions/confidence");
    assert.deepEqual(JSON.parse(calls[2].options.body), {
      score: 4,
      summary: "This issue is a good fit for an agent.",
    });
  } finally {
    global.fetch = previousFetch;
    if (previousBaseURL === undefined) delete process.env.SUPERPLANE_BASE_URL;
    else process.env.SUPERPLANE_BASE_URL = previousBaseURL;
    if (previousToken === undefined) delete process.env.SUPERPLANE_RUN_TOKEN;
    else process.env.SUPERPLANE_RUN_TOKEN = previousToken;
  }
});

test("compact status text is not a user-facing reply", () => {
  assert.equal(isCompactStatusText("Compactions remaining: 0"), true);
  assert.equal(isCompactStatusText("  Compactions remaining: 2 \n"), true);
  assert.equal(isCompactStatusText("The plan is ready."), false);
});

test("recordAgentMessage ignores Claude compact status", async () => {
  const previousFetch = global.fetch;
  const calls = [];
  global.fetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, text: async () => '{"status":"shown"}' };
  };
  try {
    const result = await recordAgentMessage("Compactions remaining: 0");
    assert.deepEqual(result, { status: "ignored" });
    assert.equal(calls.length, 0);
  } finally {
    global.fetch = previousFetch;
  }
});

test("recordAgentMessage publishes the final reply outside the MCP tool list", async () => {
  const previousBaseURL = process.env.SUPERPLANE_BASE_URL;
  const previousToken = process.env.SUPERPLANE_RUN_TOKEN;
  const previousActivityID = process.env.SUPERPLANE_ACTIVITY_ID;
  const previousFetch = global.fetch;
  const calls = [];
  process.env.SUPERPLANE_BASE_URL = "https://superplane.example";
  process.env.SUPERPLANE_RUN_TOKEN = "runner-token";
  process.env.SUPERPLANE_ACTIVITY_ID = "activity-1";
  global.fetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, text: async () => '{"status":"shown"}' };
  };

  try {
    const result = await recordAgentMessage("  I found the retry seam.  ");
    assert.deepEqual(result, { status: "shown" });
    assert.equal(calls.length, 1);
    assert.equal(
      calls[0].url,
      "https://superplane.example/api/v1/runner/planning-sessions/agent-messages",
    );
    assert.equal(calls[0].options.method, "POST");
    assert.equal(calls[0].options.headers.Authorization, "Bearer runner-token");
    assert.deepEqual(JSON.parse(calls[0].options.body), {
      text: "I found the retry seam.",
      activity_id: "activity-1",
    });
  } finally {
    global.fetch = previousFetch;
    if (previousBaseURL === undefined) delete process.env.SUPERPLANE_BASE_URL;
    else process.env.SUPERPLANE_BASE_URL = previousBaseURL;
    if (previousToken === undefined) delete process.env.SUPERPLANE_RUN_TOKEN;
    else process.env.SUPERPLANE_RUN_TOKEN = previousToken;
    if (previousActivityID === undefined)
      delete process.env.SUPERPLANE_ACTIVITY_ID;
    else process.env.SUPERPLANE_ACTIVITY_ID = previousActivityID;
  }
});

test("lists planning tools over newline-delimited JSON-RPC", async () => {
  const replies = await exchangeMCP("ndjson", [
    {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        protocolVersion: "2024-11-05",
        capabilities: {},
        clientInfo: { name: "test", version: "1" },
      },
    },
    { jsonrpc: "2.0", id: 2, method: "tools/list", params: {} },
  ]);
  assert.equal(replies[0].id, 1);
  assert.equal(replies[0].result.serverInfo.name, "superplane");
  const tools = replies[1].result.tools;
  assert.deepEqual(
    tools.map((tool) => tool.name),
    ["propose_update", "inspect_attachment"],
  );
  const [updateTool, inspectTool] = tools;
  assert.match(updateTool.description, /clarity, complexity, and verifiability/);
  assert.match(updateTool.description, /1 through 3/);
  assert.deepEqual(inspectTool.inputSchema.required, ["path"]);
  assert.match(inspectTool.description, /user image/);
  assert.match(inspectTool.description, /Do not use OCR/);
});

test("lists planning tools over Content-Length JSON-RPC", async () => {
  const replies = await exchangeMCP("lsp", [
    {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: { protocolVersion: "2024-11-05", capabilities: {} },
    },
    { jsonrpc: "2.0", id: 2, method: "tools/list", params: {} },
  ]);
  assert.deepEqual(
    replies[1].result.tools.map((tool) => tool.name),
    ["propose_update", "inspect_attachment"],
  );
});

test("inspectAttachment returns an image block for a saved PNG", () => {
  const value = attachmentFixture();
  const result = inspectAttachment({ path: value.file }, value.env);

  assert.equal(result.content[0].type, "text");
  assert.deepEqual(result.content[1], {
    type: "image",
    data: PNG_BYTES.toString("base64"),
    mimeType: "image/png",
  });
  assert.equal(result.structuredContent.data, undefined);
  assert.equal(result.structuredContent.filename, "01-shot.png");
  assert.match(result.structuredContent.sha256, /^[a-f0-9]{64}$/);
});

test("inspectAttachment sniffs a PNG without an extension", () => {
  const value = attachmentFixture();
  const file = path.join(value.attachments, "02-aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa");
  fs.writeFileSync(file, PNG_BYTES);
  const result = inspectAttachment({ path: file }, value.env);
  assert.equal(result.content[1].mimeType, "image/png");
});

test("inspectAttachment sniffs GIF and JPEG bytes over the filename", () => {
  const value = attachmentFixture();
  const gif = path.join(value.attachments, "shot.png");
  fs.writeFileSync(gif, GIF_BYTES);
  assert.equal(inspectAttachment({ path: gif }, value.env).content[1].mimeType, "image/gif");

  const jpeg = path.join(value.attachments, "shot.gif");
  fs.writeFileSync(jpeg, JPEG_BYTES);
  assert.equal(inspectAttachment({ path: jpeg }, value.env).content[1].mimeType, "image/jpeg");
});

test("inspectAttachment rejects a PNG filename that is not an image", () => {
  const value = attachmentFixture();
  const file = path.join(value.attachments, "fake.png");
  fs.writeFileSync(file, Buffer.from("png"));
  assert.throws(
    () => inspectAttachment({ path: file }, value.env),
    /PNG, JPEG, GIF, or WebP/,
  );
});

test("inspectAttachment accepts a file at the work-order size limit", () => {
  const value = attachmentFixture();
  const file = path.join(value.attachments, "limit.png");
  const descriptor = fs.openSync(file, "w");
  fs.writeSync(descriptor, PNG_BYTES);
  fs.ftruncateSync(descriptor, MAX_INSPECTABLE_ATTACHMENT_BYTES);
  fs.closeSync(descriptor);
  const result = inspectAttachment({ path: file }, value.env);
  assert.equal(result.structuredContent.mimeType, "image/png");
  assert.equal(result.structuredContent.sizeBytes, MAX_INSPECTABLE_ATTACHMENT_BYTES);
});

test("inspectAttachment rejects files above the work-order size limit", () => {
  const value = attachmentFixture();
  const oversized = path.join(value.attachments, "large.png");
  const descriptor = fs.openSync(oversized, "w");
  fs.ftruncateSync(descriptor, MAX_INSPECTABLE_ATTACHMENT_BYTES + 1);
  fs.closeSync(descriptor);
  assert.throws(
    () => inspectAttachment({ path: oversized }, value.env),
    /attachment exceeds/,
  );
});

test("inspectAttachment accepts a filename relative to attachments", () => {
  const value = attachmentFixture();
  const result = inspectAttachment({ path: "01-shot.png" }, value.env);
  assert.equal(result.content[1].mimeType, "image/png");
});

test("inspectAttachment rejects paths outside the attachments directory", () => {
  const value = attachmentFixture();
  const outside = path.join(value.taskDir, "outside.png");
  fs.writeFileSync(outside, "png");
  assert.throws(
    () => inspectAttachment({ path: outside }, value.env),
    /inside the task attachments directory/,
  );
});

test("inspectAttachment rejects symlinks", () => {
  const value = attachmentFixture();
  const link = path.join(value.attachments, "linked.png");
  fs.symlinkSync(value.file, link);
  assert.throws(() => inspectAttachment({ path: link }, value.env), /regular file/);
});

test("returns inspect_attachment image content over JSON-RPC", async () => {
  const value = attachmentFixture();
  const replies = await exchangeMCP(
    "ndjson",
    [
      {
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: { name: "inspect_attachment", arguments: { path: value.file } },
      },
    ],
    value.env,
  );
  assert.equal(replies[0].result.content[1].type, "image");
  assert.equal(replies[0].result.content[1].data, PNG_BYTES.toString("base64"));
  assert.equal(replies[0].result.structuredContent.data, undefined);
});

test("parseFrames keeps a partial Content-Length header", () => {
  const ping = {
    jsonrpc: "2.0",
    id: 1,
    method: "ping",
  };
  const frame = Buffer.from(encodeMCP("lsp", ping));
  const partial = frame.subarray(0, 8);
  const parsed = parseFrames(partial);
  assert.equal(parsed.messages.length, 0);
  assert.deepEqual(parsed.rest, partial);
});

test("parseFrames reads LSP frames that arrive in small chunks", () => {
  const ping = {
    jsonrpc: "2.0",
    id: 1,
    method: "ping",
  };
  const list = {
    jsonrpc: "2.0",
    id: 2,
    method: "tools/list",
    params: {},
  };
  const frame = Buffer.concat([
    Buffer.from(encodeMCP("lsp", ping)),
    Buffer.from(encodeMCP("lsp", list)),
  ]);
  for (const chunkSize of [1, 8, 15, 16, 64]) {
    const fed = feedParseFrames(frame, chunkSize);
    assert.deepEqual(
      fed.messages.map((message) => message.method),
      ["ping", "tools/list"],
      `chunk size ${chunkSize}`,
    );
    assert.equal(fed.rest.length, 0, `chunk size ${chunkSize}`);
  }
});

test("parseFrames still recovers NDJSON after a stray byte", () => {
  const ping = {
    jsonrpc: "2.0",
    id: 1,
    method: "ping",
  };
  const parsed = parseFrames(Buffer.from(`x${encodeMCP("ndjson", ping)}`));
  assert.equal(parsed.messages.length, 1);
  assert.equal(parsed.messages[0].method, "ping");
});

function feedParseFrames(frame, chunkSize) {
  let buffer = Buffer.alloc(0);
  const messages = [];
  for (let index = 0; index < frame.length; index += chunkSize) {
    buffer = Buffer.concat([
      buffer,
      frame.subarray(index, index + chunkSize),
    ]);
    const parsed = parseFrames(buffer);
    buffer = parsed.rest;
    messages.push(...parsed.messages);
  }
  return { messages, rest: buffer };
}

async function exchangeMCP(format, messages, extraEnv = {}) {
  const child = spawn(
    process.execPath,
    [path.join(__dirname, "planning_session_mcp.js")],
    {
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env, ...extraEnv },
    },
  );
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
  assert.equal(
    replies.length,
    messages.length,
    `expected ${messages.length} ${format} replies, got ${replies.length}`,
  );
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
      messages.push(
        JSON.parse(rest.slice(bodyStart, bodyStart + length).toString("utf8")),
      );
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

function attachmentFixture() {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "planning-attachments-"));
  const attachments = path.join(taskDir, "attachments");
  fs.mkdirSync(attachments);
  const file = path.join(attachments, "01-shot.png");
  fs.writeFileSync(file, PNG_BYTES);
  return {
    taskDir,
    attachments,
    file,
    env: { SUPERPLANE_TASK_DIR: taskDir },
  };
}
