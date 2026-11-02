"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const { spawn } = require("node:child_process");

const SERVER = path.join(__dirname, "server.js");
const SHARED_TOOLS = [
  "list_members",
  "assign_task_owner",
  "suggest_task_owner",
  "list_tasks",
  "create_task",
  "close_task",
  "list_checks",
  "add_checks",
  "remove_checks",
  "remove_artifact",
];
const PNG = Buffer.concat([Buffer.from("\x89PNG\r\n\x1a\n", "binary"), Buffer.alloc(16)]);

async function fakeRunnerAPI(t, tools = SHARED_TOOLS) {
  const requests = [];
  const api = http.createServer((req, res) => {
    const chunks = [];
    req.on("data", (chunk) => {
      chunks.push(chunk);
    });
    req.on("end", () => {
      const raw = Buffer.concat(chunks);
      requests.push({ method: req.method, url: req.url, auth: req.headers.authorization, body: raw.toString("utf8"), raw });
      res.setHeader("Content-Type", "application/json");
      if (req.url === "/api/v1/runner/tools") {
        res.end(JSON.stringify({ tools }));
        return;
      }
      res.end(JSON.stringify({ ok: true, url: req.url, file_id: "file-1" }));
    });
  });
  await new Promise((resolve) => api.listen(0, "127.0.0.1", resolve));
  t.after(() => api.close());
  return { baseURL: `http://127.0.0.1:${api.address().port}`, requests };
}

const framings = {
  ndjson: {
    encode: (message) => `${JSON.stringify(message)}\n`,
    decode(buffer) {
      const newline = buffer.indexOf("\n");
      if (newline < 0) return null;
      return { message: JSON.parse(buffer.subarray(0, newline).toString("utf8")), rest: buffer.subarray(newline + 1) };
    },
  },
  "content-length": {
    encode(message) {
      const payload = Buffer.from(JSON.stringify(message), "utf8");
      return Buffer.concat([Buffer.from(`Content-Length: ${payload.length}\r\n\r\n`), payload]);
    },
    decode(buffer) {
      const headerEnd = buffer.indexOf("\r\n\r\n");
      if (headerEnd < 0) return null;
      const length = Number(buffer.subarray(0, headerEnd).toString("utf8").match(/Content-Length:\s*(\d+)/i)[1]);
      const start = headerEnd + 4;
      if (buffer.length < start + length) return null;
      return { message: JSON.parse(buffer.subarray(start, start + length).toString("utf8")), rest: buffer.subarray(start + length) };
    },
  },
};

function startServer(t, baseURL, framing, extraEnv = {}) {
  const child = spawn(process.execPath, [SERVER], {
    env: { ...process.env, SUPERPLANE_BASE_URL: baseURL, SUPERPLANE_RUNNER_TOKEN: "runner-token", ...extraEnv },
    stdio: ["pipe", "pipe", "inherit"],
  });
  t.after(() => child.kill());

  let buffer = Buffer.alloc(0);
  const pending = [];
  child.stdout.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    for (let frame = framing.decode(buffer); frame; frame = framing.decode(buffer)) {
      buffer = frame.rest;
      pending.shift()(frame.message);
    }
  });

  let nextID = 1;
  return function request(method, params) {
    const id = nextID++;
    return new Promise((resolve) => {
      pending.push(resolve);
      child.stdin.write(framing.encode({ jsonrpc: "2.0", id, method, params }));
    });
  };
}

for (const [name, framing] of Object.entries(framings)) {
  test(`server.js serves the shared tools over ${name} stdio`, async (t) => {
    const api = await fakeRunnerAPI(t);
    const request = startServer(t, api.baseURL, framing);

    const init = await request("initialize", {});
    assert.equal(init.result.serverInfo.name, "superplane_global");

    const list = await request("tools/list", {});
    assert.deepEqual(list.result.tools.map((tool) => tool.name), SHARED_TOOLS);

    const check = { key: "risk", name: "Risk", score: 2, max_score: 5, level: "caution", reason: "Small change." };
    const toolCalls = [
      ["list_members", {}],
      ["suggest_task_owner", { paths: ["pkg/a.go"] }],
      ["assign_task_owner", { user: "ana", reason: "Owns pkg" }],
      ["list_tasks", { state: "open", limit: 5 }],
      ["create_task", { title: "Ship it" }],
      ["close_task", { result: "rejected" }],
      ["list_checks", { task: "SP-1" }],
      ["add_checks", { checks: [check] }],
      ["remove_checks", { keys: ["risk"] }],
      ["remove_artifact", { title: "PLAN.md" }],
    ];
    for (const [tool, args] of toolCalls) {
      const reply = await request("tools/call", { name: tool, arguments: args });
      assert.equal(reply.result.isError, undefined, `${tool}: ${JSON.stringify(reply.result)}`);
    }

    const calls = api.requests.filter((req) => req.url !== "/api/v1/runner/tools");
    assert.deepEqual(
      calls.map((req) => [req.method, req.url, req.auth, JSON.parse(req.body || "{}")]),
      [
        ["GET", "/api/v1/runner/members", "Bearer runner-token", {}],
        ["POST", "/api/v1/runner/task/suggest-owners", "Bearer runner-token", { paths: ["pkg/a.go"] }],
        ["PUT", "/api/v1/runner/task/owner", "Bearer runner-token", { user: "ana", reason: "Owns pkg" }],
        ["GET", "/api/v1/runner/tasks?state=open&limit=5", "Bearer runner-token", {}],
        ["POST", "/api/v1/runner/tasks", "Bearer runner-token", { title: "Ship it", description: "" }],
        ["POST", "/api/v1/runner/task/close", "Bearer runner-token", { result: "rejected" }],
        ["GET", "/api/v1/runner/checks?task=SP-1", "Bearer runner-token", {}],
        ["POST", "/api/v1/runner/task/checks", "Bearer runner-token", { checks: [check] }],
        ["DELETE", "/api/v1/runner/task/checks", "Bearer runner-token", { keys: ["risk"] }],
        ["DELETE", "/api/v1/runner/task/artifacts", "Bearer runner-token", { title: "PLAN.md" }],
      ],
    );
  });
}

function flowTaskDir() {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "global-mcp-"));
  fs.mkdirSync(path.join(taskDir, "attachments"));
  fs.mkdirSync(path.join(taskDir, "evidence"));
  fs.writeFileSync(path.join(taskDir, "attachments", "mock.png"), PNG);
  fs.writeFileSync(path.join(taskDir, "evidence", "checkout.png"), PNG);
  return { taskDir, specFile: path.join(taskDir, "spec.md") };
}

test("server.js serves the flow tools with the runner token", async (t) => {
  const tools = ["get_artifact", "add_artifact", "propose_plan", "report_visual_evidence_unavailable", "report_merge_check"];
  const api = await fakeRunnerAPI(t, tools);
  const { taskDir, specFile } = flowTaskDir();
  const request = startServer(t, api.baseURL, framings.ndjson, {
    SUPERPLANE_TASK_DIR: taskDir,
    SUPERPLANE_ANALYSIS_SPEC_FILE: specFile,
  });

  const list = await request("tools/list", {});
  assert.deepEqual(list.result.tools.map((tool) => tool.name), tools);
  assert.deepEqual(JSON.parse(fs.readFileSync(path.join(taskDir, "visual-evidence.json"), "utf8")), {
    status: "not_applicable",
    reason: "",
    artifacts: [],
  });

  const attachment = await request("tools/call", { name: "get_artifact", arguments: { path: "attachments/mock.png" } });
  assert.equal(attachment.result.content[1].type, "image");
  assert.equal(attachment.result.content[1].data, PNG.toString("base64"));
  assert.equal(attachment.result.structuredContent.mimeType, "image/png");

  const plan = await request("tools/call", { name: "propose_plan", arguments: { spec: "# Plan" } });
  assert.equal(plan.result.isError, undefined);
  assert.equal(fs.readFileSync(specFile, "utf8"), "# Plan");

  await request("tools/call", { name: "get_artifact", arguments: { path: "evidence/checkout.png" } });
  const upload = await request("tools/call", {
    name: "add_artifact",
    arguments: { type: "file", path: "evidence/checkout.png", title: "Checkout", key: "checkout" },
  });
  assert.equal(upload.result.isError, undefined, JSON.stringify(upload.result));
  assert.equal(JSON.parse(fs.readFileSync(path.join(taskDir, "visual-evidence.json"), "utf8")).status, "captured");

  await request("tools/call", {
    name: "add_artifact",
    arguments: { type: "link", url: "https://staging.example.com", title: "Staging" },
  });
  await request("tools/call", { name: "report_merge_check", arguments: { check: "Risk", score: 2, summary: "Small change." } });

  const calls = api.requests.filter((req) => req.url !== "/api/v1/runner/tools");
  assert.deepEqual(
    calls.map((req) => [req.method, req.url, req.auth]),
    [
      ["PUT", "/api/v1/runner/task/plan", "Bearer runner-token"],
      ["POST", "/api/v1/runner/task/artifacts", "Bearer runner-token"],
      ["POST", "/api/v1/runner/task/links", "Bearer runner-token"],
      ["POST", "/api/v1/runner/task/merge-checks", "Bearer runner-token"],
    ],
  );
  assert.deepEqual(JSON.parse(calls[0].body), { spec: "# Plan" });
  assert.deepEqual(calls[1].raw, PNG);
  assert.deepEqual(JSON.parse(calls[2].body), { url: "https://staging.example.com", title: "Staging", key: "" });
  assert.deepEqual(JSON.parse(calls[3].body), { check: "risk", score: 2, summary: "Small change." });
});
