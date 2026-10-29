"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const { createHandler } = require("./server");

function fakeClient(allowed, calls = []) {
  return {
    request: async (method, path, body) => {
      calls.push({ method, path, body });
      if (path === "/api/v1/runner/tools") {
        if (allowed instanceof Error) throw allowed;
        return { tools: allowed };
      }
      return { status: "ok", path };
    },
  };
}

function recorder() {
  const replies = [];
  return {
    replies,
    sendResult: (id, result) => replies.push({ id, result }),
    sendError: (id, code, message) => replies.push({ id, error: { code, message } }),
  };
}

test("tools/list shows only the tools that the backend allows", async () => {
  const transport = recorder();
  const handle = createHandler({ client: fakeClient(["suggest_owners"]) });

  await handle({ id: 1, method: "tools/list" }, transport);

  assert.deepEqual(
    transport.replies[0].result.tools.map((tool) => tool.name),
    ["suggest_owners"],
  );
});

test("tools/list fails on a failed lookup and retries on the next request", async () => {
  const lookups = [new Error("Service Unavailable"), ["suggest_owners"]];
  const client = {
    request: async () => {
      const next = lookups.shift();
      if (next instanceof Error) throw next;
      return { tools: next };
    },
  };
  const transport = recorder();
  const handle = createHandler({ client });

  await assert.rejects(handle({ id: 1, method: "tools/list" }, transport), /Service Unavailable/);
  await handle({ id: 2, method: "tools/list" }, transport);

  assert.deepEqual(
    transport.replies[0].result.tools.map((tool) => tool.name),
    ["suggest_owners"],
  );
});

test("tools/call rejects a hidden tool and runs an allowed tool", async () => {
  const calls = [];
  const transport = recorder();
  const handle = createHandler({ client: fakeClient(["suggest_owners"], calls) });

  await handle(
    { id: 1, method: "tools/call", params: { name: "assign_owner", arguments: { user: "u", reason: "r" } } },
    transport,
  );
  await handle({ id: 2, method: "tools/call", params: { name: "suggest_owners", arguments: {} } }, transport);
  await handle({ id: 3, method: "tools/call", params: { name: "drop_database", arguments: {} } }, transport);

  assert.equal(transport.replies[0].error.code, -32601);
  assert.equal(transport.replies[1].result.structuredContent.path, "/api/v1/runner/task/suggest-owners");
  assert.equal(transport.replies[2].error.code, -32601);
  assert.equal(calls.filter((call) => call.path === "/api/v1/runner/task/owner").length, 0);
  assert.equal(calls.filter((call) => call.path === "/api/v1/runner/tools").length, 1);
});

test("tools/call returns a backend error as a tool error", async () => {
  const transport = recorder();
  const client = {
    request: async (_method, path) => {
      if (path === "/api/v1/runner/tools") return { tools: ["assign_owner"] };
      throw new Error("Task is closed or archived");
    },
  };
  const handle = createHandler({ client });

  await handle(
    { id: 1, method: "tools/call", params: { name: "assign_owner", arguments: { user: "u", reason: "r" } } },
    transport,
  );

  assert.equal(transport.replies[0].result.isError, true);
  assert.equal(transport.replies[0].result.content[0].text, "Task is closed or archived");
});

test("initialize names the global server", async () => {
  const transport = recorder();
  await createHandler({ client: fakeClient([]) })({ id: 1, method: "initialize" }, transport);
  assert.equal(transport.replies[0].result.serverInfo.name, "superplane_global");
});
