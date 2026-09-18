"use strict";

const { spawn } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const assert = require("node:assert/strict");

const { analysisProtocol, withoutEmbeddedAnalysisProtocol, withAnalysisContinuation } = require("./analysis_protocol");
const {
  proposeSpec,
  proposeClarity,
  proposeConfidence,
  recordAgentMessage,
  writeAnalysisOutputs,
} = require("./planning_session_mcp");

test("analysis protocol covers publish tools and hides chat dumps", () => {
  const pack = analysisProtocol();
  assert.match(pack, /propose_spec/);
  assert.match(pack, /propose_clarity/);
  assert.match(pack, /propose_confidence/);
  assert.match(pack, /A question can raise Clarity or Confidence/);
  assert.doesNotMatch(pack, /Do not ask a question to raise Confidence/);
  assert.doesNotMatch(pack, /propose_plan/);
  assert.match(pack, /The task prompt owns the judgment/);
  assert.match(pack, /Use only the analysis tools/);
  assert.match(pack, /Do not paste the specification/);
  assert.match(pack, /call survey with 2 to 4 options/);
  assert.match(
    pack,
    /\{"questions":\[\{"prompt":"Your question","options":\["First option","Second option"\]\}\]\}/,
  );
  assert.match(pack, /Do not use XML tags/);
  assert.match(pack, /If the survey tool is unavailable or fails, do not put the questions in chat/);
  assert.match(pack, /You may update the score without rewriting the specification/);
  assert.match(pack, /this is a continuation/);
  assert.match(pack, /does not publish the specification or the score/);
  assert.match(pack, /only after those calls/);
  assert.match(pack, /Do not leave a written plan unpublished/);
  assert.doesNotMatch(pack, /Call propose_spec when the task prompt says/);
  assert.match(pack, /Do not name files/);
  assert.match(pack, /Answer the questions in this session/);
  assert.match(pack, /Do not describe agent fit/);
  assert.match(pack, /Do not write a test or an acceptance check/);
  assert.match(pack, /Do not add an Open questions section/);
  assert.doesNotMatch(pack, /Talk like a colleague/);
  assert.doesNotMatch(pack, /## Proposed outcome/);
  assert.doesNotMatch(pack, /## 1\. Research/);
  assert.doesNotMatch(pack, /Why not start/);
  assert.doesNotMatch(pack, /you must ask/);
  assert.doesNotMatch(pack, /## Executive summary/);
  assert.doesNotMatch(pack, /check copy/);
  assert.doesNotMatch(pack, /\/tmp\/spec\.md/);
});

test("analysis user prompt covers tone, score rules, and plan shape", () => {
  const pack = fs.readFileSync(path.join(__dirname, "analysis_user_prompt.md"), "utf8");
  assert.match(pack, /Talk like a colleague/);
  assert.match(pack, /## 1\. Research/);
  assert.match(pack, /## 2\. Decide or ask/);
  assert.match(pack, /## 3\. Score Clarity/);
  assert.match(pack, /## 4\. Score Confidence/);
  assert.match(pack, /## 5\. Write the plan/);
  assert.match(pack, /how well the task is defined/);
  assert.doesNotMatch(pack, /how likely implementation is to succeed/);
  assert.match(pack, /why Clarity is not 5/);
  assert.match(pack, /how likely a coding agent finishes this task in one run/);
  assert.match(pack, /### Calibration/);
  assert.match(pack, /Start at 4 for a bounded change that has a pattern in the repository/);
  assert.match(pack, /### Raise Confidence through refinement/);
  assert.match(pack, /Split into two or three tasks that each fit one run/);
  assert.match(pack, /Be direct when the task is too big or too complex for one run/);
  assert.doesNotMatch(pack, /Do not push Confidence to 5/);
  assert.doesNotMatch(pack, /Do not ask a survey question to raise it/);
  assert.match(pack, /Confidence is provisional/);
  assert.strictEqual(pack.match(/Do not repeat the number in the summary/g)?.length, 2);
  assert.doesNotMatch(pack, /Good: Clarity is \d because/);
  assert.doesNotMatch(pack, /Good: Confidence is \d because/);
  assert.match(pack, /Blast radius/);
  assert.match(pack, /Skip Risks only when Clarity is 5 and Confidence is 4 or higher/);
  assert.doesNotMatch(pack, /how suitable the work is for an agent/);
  assert.match(pack, /2 to 4 short sentences/);
  assert.match(pack, /Keep each option under 12 words/);
  assert.match(pack, /If Clarity is 1 or 2/);
  assert.match(pack, /do not have enough Clarity to write a plan/);
  assert.match(pack, /Keep asking until Clarity is 5/);
  assert.match(pack, /Clarity 3 and 4/);
  assert.match(pack, /Review it and start if you are happy/);
  assert.match(pack, /A simple task can reach 5 with no survey/);
  assert.match(pack, /Do not invent a survey to fill a quota/);
  assert.match(pack, /## Proposed outcome/);
  assert.match(pack, /## Constraints/);
  assert.match(pack, /## Scope/);
  assert.match(pack, /Do not repeat the goal/);
  assert.doesNotMatch(pack, /propose_spec/);
  assert.doesNotMatch(pack, /in chat, survey, or the Clarity summary/);
  assert.doesNotMatch(pack, /Do not describe agent fit/);
  assert.doesNotMatch(pack, /Do not add an Open questions section/);
  assert.doesNotMatch(pack, /## Executive summary/);
  assert.doesNotMatch(pack, /## Files and seams/);
  assert.doesNotMatch(pack, /at least five/);
  assert.doesNotMatch(pack, /Key architecture decisions/);
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
    ["propose_spec", "propose_clarity", "propose_confidence", "survey"],
  );
  const [spec, clarity, confidence, survey] = tools;
  assert.deepEqual(spec.inputSchema.required, ["body"]);
  assert.match(spec.description, /Do not leave a written plan unpublished/);
  assert.match(clarity.description, /how well the task is defined/);
  assert.match(clarity.description, /every turn/);
  assert.match(clarity.description, /without propose_spec/);
  assert.match(clarity.inputSchema.properties.summary.description, /Follow the task prompt/);
  assert.match(confidence.description, /how likely a coding agent completes this task in one run/);
  assert.match(confidence.description, /every turn/);
  assert.match(confidence.description, /without propose_spec/);
  assert.deepEqual(confidence.inputSchema.required, ["score", "summary"]);
  assert.match(survey.description, /short everyday options/);
  assert.match(survey.description, /task prompt says to ask/);
  assert.match(survey.inputSchema.properties.questions.description, /JSON array/);
  assert.equal(survey.inputSchema.additionalProperties, undefined);
  assert.equal(survey.inputSchema.properties.questions.items.additionalProperties, undefined);
  assert.equal(
    survey.inputSchema.properties.questions.items.properties.options.minItems,
    undefined,
  );
  assert.doesNotMatch(clarity.description, /check copy/);
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
    ["propose_spec", "propose_clarity", "propose_confidence", "survey"],
  );
});

async function exchangeMCP(format, messages) {
  const child = spawn(
    process.execPath,
    [path.join(__dirname, "planning_session_mcp.js")],
    {
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env },
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
