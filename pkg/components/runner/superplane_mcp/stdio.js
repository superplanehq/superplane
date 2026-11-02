"use strict";

/**
 * Stdio JSON-RPC transport for MCP. Agents frame messages as newline-delimited
 * JSON or with Content-Length headers. Replies use the framing of the last
 * request.
 */

function createTransport(output = process.stdout) {
  let replyFormat = "ndjson";

  function write(message) {
    const encoded = JSON.stringify(message);
    if (replyFormat === "lsp") {
      const payload = Buffer.from(encoded, "utf8");
      output.write(`Content-Length: ${payload.length}\r\n\r\n`);
      output.write(payload);
      return;
    }
    output.write(`${encoded}\n`);
  }

  function parse(buffer) {
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
        if (headerEnd < 0) break;
        const match = rest.subarray(0, headerEnd).toString("utf8").match(/Content-Length:\s*(\d+)/i);
        if (!match) {
          rest = rest.subarray(headerEnd + 4);
          continue;
        }
        const length = Number(match[1]);
        if (rest.length < headerEnd + 4 + length) break;
        replyFormat = "lsp";
        pushJSON(messages, rest.subarray(headerEnd + 4, headerEnd + 4 + length).toString("utf8"));
        rest = rest.subarray(headerEnd + 4 + length);
        continue;
      }
      const newline = rest.indexOf("\n");
      if (newline < 0) break;
      const line = rest.subarray(0, newline).toString("utf8").trim();
      rest = rest.subarray(newline + 1);
      if (!line) continue;
      replyFormat = "ndjson";
      pushJSON(messages, line);
    }
    return { messages, rest };
  }

  return {
    parse,
    sendResult: (id, result) => write({ jsonrpc: "2.0", id, result }),
    sendError: (id, code, message) => write({ jsonrpc: "2.0", id, error: { code, message } }),
  };
}

function pushJSON(messages, text) {
  try {
    messages.push(JSON.parse(text));
  } catch (_err) {
    // Ignore a malformed frame.
  }
}

function start(handle, input = process.stdin, transport = createTransport()) {
  let buffer = Buffer.alloc(0);
  input.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    const parsed = transport.parse(buffer);
    buffer = parsed.rest;
    for (const message of parsed.messages) {
      Promise.resolve(handle(message, transport)).catch((err) => {
        if (message && message.id != null) {
          transport.sendError(message.id, -32603, errorMessage(err));
        }
      });
    }
  });
}

function errorMessage(err) {
  return err && err.message ? err.message : String(err);
}

module.exports = { createTransport, start, errorMessage };
