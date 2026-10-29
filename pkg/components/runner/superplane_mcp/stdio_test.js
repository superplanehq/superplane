"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const path = require("node:path");
const { spawn } = require("node:child_process");

const SERVER = path.join(__dirname, "server.js");

async function fakeRunnerAPI(t) {
  const requests = [];
  const api = http.createServer((req, res) => {
    let body = "";
    req.on("data", (chunk) => {
      body += chunk;
    });
    req.on("end", () => {
      requests.push({ method: req.method, url: req.url, auth: req.headers.authorization, body });
      res.setHeader("Content-Type", "application/json");
      if (req.url === "/api/v1/runner/tools") {
        res.end(JSON.stringify({ tools: ["suggest_owners", "assign_owner"] }));
        return;
      }
      res.end(JSON.stringify({ ok: true, url: req.url }));
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

function startServer(t, baseURL, framing) {
  const child = spawn(process.execPath, [SERVER], {
    env: { ...process.env, SUPERPLANE_BASE_URL: baseURL, SUPERPLANE_RUNNER_TOKEN: "runner-token" },
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
      ["suggest_owners", "assign_owner"],
    );

    const suggest = await request("tools/call", { name: "suggest_owners", arguments: { paths: ["pkg/a.go"] } });
    assert.equal(suggest.result.structuredContent.url, "/api/v1/runner/task/suggest-owners");

    const assign = await request("tools/call", { name: "assign_owner", arguments: { user: "ana", reason: "Owns pkg" } });
    assert.equal(assign.result.structuredContent.url, "/api/v1/runner/task/owner");

    const calls = api.requests.filter((req) => req.url !== "/api/v1/runner/tools");
    assert.deepEqual(
      calls.map((req) => [req.method, req.url, req.auth, JSON.parse(req.body)]),
      [
        ["POST", "/api/v1/runner/task/suggest-owners", "Bearer runner-token", { paths: ["pkg/a.go"] }],
        ["PUT", "/api/v1/runner/task/owner", "Bearer runner-token", { user: "ana", reason: "Owns pkg" }],
      ],
    );
  });
}
