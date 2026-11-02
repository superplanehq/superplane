#!/usr/bin/env node
"use strict";

/**
 * Global runner MCP server. It lists only the tools that the run token allows.
 * The backend checks the token scope again on every call.
 */

const { createClient } = require("./client");
const registry = require("./registry");
const stdio = require("./stdio");

const SERVER_NAME = "superplane_global";

function createHandler({
  client = createClient(),
  tools = registry.allTools(),
  prepare = (names) => registry.prepare(names, client),
} = {}) {
  let allowed;

  // A failed lookup is not cached, so the next request can retry it.
  function allowedNames() {
    if (!allowed) {
      allowed = client.request("GET", "/api/v1/runner/tools").then((result) => {
        const names = new Set(Array.isArray(result.tools) ? result.tools : []);
        try {
          prepare(names);
        } catch (err) {
          process.stderr.write(`prepare global runner tools: ${stdio.errorMessage(err)}\n`);
        }
        return names;
      });
      allowed.catch(() => {
        allowed = undefined;
      });
    }
    return allowed;
  }

  async function allowedTools() {
    const names = await allowedNames();
    return tools.filter((tool) => names.has(tool.definition.name));
  }

  async function callTool(id, params, transport) {
    const name = params && params.name;
    const tool = (await allowedTools()).find((candidate) => candidate.definition.name === name);
    if (!tool) {
      transport.sendError(id, -32601, `Unknown tool: ${name}`);
      return;
    }
    try {
      const result = await tool.handler((params && params.arguments) || {}, client);
      transport.sendResult(id, toolResult(result));
    } catch (err) {
      transport.sendResult(id, {
        content: [{ type: "text", text: stdio.errorMessage(err) }],
        isError: true,
      });
    }
  }

  return async function handle(message, transport) {
    const { id, method, params } = message || {};
    switch (method) {
      case "initialize":
        transport.sendResult(id, {
          protocolVersion: "2024-11-05",
          capabilities: { tools: {} },
          serverInfo: { name: SERVER_NAME, version: "1.0.0" },
        });
        return;
      case "notifications/initialized":
      case "notifications/cancelled":
        return;
      case "ping":
        transport.sendResult(id, {});
        return;
      case "tools/list":
        transport.sendResult(id, { tools: (await allowedTools()).map((tool) => tool.definition) });
        return;
      case "tools/call":
        await callTool(id, params, transport);
        return;
      default:
        if (id != null) transport.sendError(id, -32601, `Unknown method: ${method}`);
    }
  };
}

// Image tools return their own content blocks, so the base64 image stays out
// of structuredContent.
function toolResult(result) {
  if (result && Array.isArray(result.content)) {
    return { content: result.content, structuredContent: result.structuredContent };
  }
  return { content: [{ type: "text", text: JSON.stringify(result) }], structuredContent: result };
}

if (require.main === module) {
  stdio.start(createHandler());
}

module.exports = { SERVER_NAME, createHandler };
