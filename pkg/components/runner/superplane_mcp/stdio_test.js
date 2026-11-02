"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const { spawn } = require("node:child_process");

const SERVER = path.join(__dirname, "server.js");
const OWNER_AND_TASK_TOOLS = ["suggest_owners", "assign_owner", "list_tasks", "create_task", "close_task", "remove_artifact"];
const PNG = Buffer.concat([Buffer.from("\x89PNG\r\n\x1a\n", "binary"), Buffer.alloc(16)]);

async function fakeRunnerAPI(t, tools = OWNER_AND_TASK_TOOLS) {
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
  test(`server.js serves the owner tools over ${name} stdio`, async (t) => {
    const api = await fakeRunnerAPI(t);
    const request = startServer(t, api.baseURL, framing);

    const init = await request("initialize", {});
    assert.equal(init.result.serverInfo.name, "superplane_global");

    const list = await request("tools/list", {});
    assert.deepEqual(
      list.result.tools.map((tool) => tool.name),
      ["suggest_owners", "assign_owner", "list_tasks", "create_task", "close_task", "remove_artifact"],
    );

    const suggest = await request("tools/call", { name: "suggest_owners", arguments: { paths: ["pkg/a.go"] } });
    assert.equal(suggest.result.structuredContent.url, "/api/v1/runner/task/suggest-owners");

    const assign = await request("tools/call", { name: "assign_owner", arguments: { user: "ana", reason: "Owns pkg" } });
    assert.equal(assign.result.structuredContent.url, "/api/v1/runner/task/owner");

    const listed = await request("tools/call", { name: "list_tasks", arguments: { state: "open", limit: 5 } });
    assert.equal(listed.result.structuredContent.url, "/api/v1/runner/tasks?state=open&limit=5");

    const created = await request("tools/call", { name: "create_task", arguments: { title: "Ship it" } });
    assert.equal(created.result.structuredContent.url, "/api/v1/runner/tasks");

    const closed = await request("tools/call", { name: "close_task", arguments: { result: "rejected" } });
    assert.equal(closed.result.structuredContent.url, "/api/v1/runner/task/close");

    const removed = await request("tools/call", { name: "remove_artifact", arguments: { title: "PLAN.md" } });
    assert.equal(removed.result.structuredContent.url, "/api/v1/runner/task/artifacts");

    const calls = api.requests.filter((req) => req.url !== "/api/v1/runner/tools");
    assert.deepEqual(
      calls.map((req) => [req.method, req.url, req.auth, JSON.parse(req.body || "{}")]),
      [
        ["POST", "/api/v1/runner/task/suggest-owners", "Bearer runner-token", { paths: ["pkg/a.go"] }],
        ["PUT", "/api/v1/runner/task/owner", "Bearer runner-token", { user: "ana", reason: "Owns pkg" }],
        ["GET", "/api/v1/runner/tasks?state=open&limit=5", "Bearer runner-token", {}],
        ["POST", "/api/v1/runner/tasks", "Bearer runner-token", { title: "Ship it", description: "" }],
        ["POST", "/api/v1/runner/task/close", "Bearer runner-token", { result: "rejected" }],
        ["DELETE", "/api/v1/runner/task/artifacts", "Bearer runner-token", { title: "PLAN.md" }],
      ],
    );
  });
}

test("server.js serves the flow tools with the runner token", async (t) => {
  const flowTools = [
    "propose_update",
    "inspect_attachment",
    "inspect_screenshot",
    "upload_artifact",
    "report_visual_evidence_unavailable",
    "publish_link",
    "report_merge_check",
  ];
  const api = await fakeRunnerAPI(t, flowTools);
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "global-mcp-"));
  fs.mkdirSync(path.join(taskDir, "attachments"));
  fs.mkdirSync(path.join(taskDir, "evidence"));
  fs.writeFileSync(path.join(taskDir, "attachments", "mock.png"), PNG);
  const screenshot = path.join(taskDir, "evidence", "checkout.png");
  fs.writeFileSync(screenshot, PNG);
  const specFile = path.join(taskDir, "spec.md");
  const request = startServer(t, api.baseURL, framings.ndjson, {
    SUPERPLANE_TASK_DIR: taskDir,
    SUPERPLANE_ANALYSIS_SPEC_FILE: specFile,
  });

  const list = await request("tools/list", {});
  assert.deepEqual(list.result.tools.map((tool) => tool.name), flowTools);
  assert.deepEqual(JSON.parse(fs.readFileSync(path.join(taskDir, "visual-evidence.json"), "utf8")), {
    status: "not_applicable",
    reason: "",
    artifacts: [],
  });

  const attachment = await request("tools/call", { name: "inspect_attachment", arguments: { path: "mock.png" } });
  assert.equal(attachment.result.content[1].type, "image");
  assert.equal(attachment.result.content[1].data, PNG.toString("base64"));
  assert.equal(attachment.result.structuredContent.mimeType, "image/png");

  const plan = await request("tools/call", { name: "propose_update", arguments: { spec: "# Plan" } });
  assert.equal(plan.result.isError, undefined);
  assert.equal(fs.readFileSync(specFile, "utf8"), "# Plan");

  await request("tools/call", { name: "inspect_screenshot", arguments: { path: screenshot } });
  const upload = await request("tools/call", { name: "upload_artifact", arguments: { path: screenshot, title: "Checkout" } });
  assert.equal(upload.result.isError, undefined, JSON.stringify(upload.result));
  assert.equal(JSON.parse(fs.readFileSync(path.join(taskDir, "visual-evidence.json"), "utf8")).status, "captured");

  await request("tools/call", { name: "publish_link", arguments: { url: "https://staging.example.com", title: "Staging" } });
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
