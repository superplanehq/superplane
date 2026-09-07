#!/usr/bin/env node
"use strict";

/**
 * Run Claude Code and format stream-json into readable live logs.
 *
 *   node run.js <prompt-file> [model]
 */

const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { spawn, spawnSync } = require("child_process");

const TOOL_RESULT_MAX_CHARS = 800;
const TOOL_RESULT_MAX_LINES = 24;

const SYSTEM_PROMPT =
  "Write all assistant messages as plain terminal text. " +
  "Do not use Markdown: no bold/italic markers, headings, links, tables, or fenced code blocks. " +
  "Prefer plain paths, shell commands, and simple indentation.";

const PLANNING_SYSTEM_PROMPT =
  " This is a SuperPlane planning session. Call mcp__superplane__propose_draft only when the user asked for a task in this turn. Call mcp__superplane__survey to ask questions. SuperPlane waits after you stop. Do not create work orders yourself. When the user creates or skips a draft, acknowledge that in one short sentence and ask what they want to do next. Do not call propose_draft unless they ask for a task. When the user starts a refine, read the current task, tell them you are ready, and ask what they want to change. Do not call propose_draft until they say what to change. Write to the user in plain text.";

const BASE_ALLOWED_TOOLS = "Bash,Read,Edit,Write";
// Planning sessions may only explore the repo (Read/Bash) and use the planning
// MCP tools. Edit/Write are intentionally excluded so the agent cannot make
// changes while drafting a task.
const PLANNING_READONLY_TOOLS = "Read,Bash";
const PLANNING_ALLOWED_TOOLS = ["mcp__superplane__propose_draft", "mcp__superplane__survey"];

function envFlag(env, name) {
  return Boolean(String((env && env[name]) || "").trim());
}

function allowedClaudeTools(env = process.env) {
  if (envFlag(env, "SUPERPLANE_PLANNING_SESSION_ID")) {
    return [PLANNING_READONLY_TOOLS, "mcp__superplane", ...PLANNING_ALLOWED_TOOLS].join(",");
  }
  return BASE_ALLOWED_TOOLS;
}

function claudePermissionMode(env = process.env) {
  if (mcpToolsEnabled(env)) {
    // Planning sessions stay read-only by restricting allowedClaudeTools to
    // Read/Bash plus the planning MCP tools. We intentionally do NOT use
    // "plan" mode here: Claude Code blocks every non-read-only tool call in
    // plan mode, including our MCP tools, which fails propose_draft/survey with
    // "Cannot call mcp__superplane__propose_draft while in plan mode." In
    // headless ("-p") mode "default" treats allowedTools as the allowlist, so
    // Edit/Write are still denied (there is no interactive prompt to grant
    // them) while the planning MCP tools remain callable.
    return "default";
  }
  return "acceptEdits";
}

function mcpToolsEnabled(env = process.env) {
  return envFlag(env, "SUPERPLANE_PLANNING_SESSION_ID");
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    console.error("usage: node run.js <prompt-file> [model]");
    process.exit(2);
  }
  runPrompt(args[0], args[1] || "")
    .then((code) => process.exit(code))
    .catch((err) => {
      console.error(err && err.message ? err.message : err);
      process.exit(1);
    });
}

async function runPrompt(promptFile, model) {
  const sp = process.env.SUPERPLANE_TASK_DIR;
  if (!sp) {
    throw new Error("SUPERPLANE_TASK_DIR is required");
  }
  const resultFile = process.env.SUPERPLANE_RESULT_FILE;
  if (!resultFile) {
    throw new Error("SUPERPLANE_RESULT_FILE is required");
  }

  const prompt = fs.readFileSync(promptFile, "utf8");
  const promptCountPath = path.join(sp, "prompt_count");
  const promptCount = Number.parseInt(fs.readFileSync(promptCountPath, "utf8").trim(), 10) || 0;

  const claudeArgs = [
    "--bare",
    "-p",
    "--output-format",
    "stream-json",
    "--verbose",
    "--include-partial-messages",
    "--permission-mode",
    claudePermissionMode(),
    "--add-dir",
    ".",
    "--append-system-prompt",
    SYSTEM_PROMPT,
  ];
  if (mcpToolsEnabled()) {
    println("Planning session tools enabled");
    println(`permission mode: ${claudePermissionMode()}`);
    claudeArgs[claudeArgs.length - 1] = SYSTEM_PROMPT + PLANNING_SYSTEM_PROMPT;
    const mcpConfigPath = path.join(sp, "mcp.runtime.json");
    fs.writeFileSync(
      mcpConfigPath,
      `${JSON.stringify({
        mcpServers: {
          superplane: {
            command: "node",
            args: [path.join(sp, "planning_session_mcp.js")],
          },
        },
      })}\n`,
    );
    claudeArgs.push("--mcp-config", mcpConfigPath);
    claudeArgs.push("--allowedTools", allowedClaudeTools());
    println(`allowed tools: ${allowedClaudeTools()}`);
  } else {
    claudeArgs.push("--allowedTools", allowedClaudeTools());
  }
  if (model) {
    claudeArgs.push("--model", model);
  }
  if (promptCount > 0) {
    claudeArgs.push("--continue");
  }
  claudeArgs.push("--", prompt);

  let command = "claude";
  let args = claudeArgs;
  if (commandExists("stdbuf")) {
    command = "stdbuf";
    args = ["-oL", "-eL", "claude", ...claudeArgs];
  }

  const formatter = createFormatter();
  const child = spawn(command, args, {
    stdio: ["ignore", "pipe", "pipe"],
  });
  child.stderr.pipe(process.stderr);

  const rl = readline.createInterface({ input: child.stdout, crlfDelay: Infinity });
  rl.on("line", (raw) => formatter.handleLine(raw));

  const exitCode = await Promise.all([
    new Promise((resolve, reject) => {
      child.on("error", reject);
      child.on("close", (code) => resolve(code == null ? 1 : code));
    }),
    new Promise((resolve) => rl.on("close", resolve)),
  ]).then(([code]) => code);

  // A nonzero exit code always means failure. But `claude -p` exits 0 even
  // when its own result event reports is_error: it treated a soft failure
  // (an invalid API key, a rate limit it never recovered from, …) as a
  // completed turn. Trust that verdict too, so the node execution — and
  // everything downstream that reads it — actually fails.
  const failed = exitCode !== 0 || formatter.resultFailed();
  formatter.flush(failed);
  const resultJSON = formatter.resultJSON();
  fs.writeFileSync(resultFile, `${resultJSON}\n`);
  accumulateLLMUsage(resultJSON, model);
  fs.writeFileSync(promptCountPath, `${promptCount + 1}\n`);
  if (failed) {
    return exitCode !== 0 ? exitCode : 1;
  }
  return exitCode;
}

function commandExists(name) {
  const result = spawnSync("sh", ["-c", `command -v ${name}`], { encoding: "utf8" });
  return result.status === 0;
}

function accumulateLLMUsage(raw, fallbackModel) {
  const taskDir = process.env.SUPERPLANE_TASK_DIR;
  if (!taskDir) {
    return;
  }
  const script = path.join(taskDir, "llm_usage.js");
  if (!fs.existsSync(script)) {
    return;
  }
  let parsed = {};
  try {
    parsed = JSON.parse(raw);
  } catch (_err) {
    return;
  }
  require(script).accumulate(taskDir, {
    model: parsed.model || fallbackModel,
    usage: parsed.usage,
    total_cost_usd: parsed.total_cost_usd,
  });
}

function createFormatter() {
  let streamedText = false;
  let inText = false;
  let textBuf = "";
  let lastLine = "";
  let resultLine = "";
  let resultFailed = false;
  const tools = createToolTracker();

  return {
    handleLine(raw) {
      const line = raw.trim();
      if (!line) {
        return;
      }
      lastLine = line;

      let event;
      try {
        event = JSON.parse(line);
      } catch {
        println(line);
        return;
      }
      if (!event || typeof event !== "object" || Array.isArray(event)) {
        return;
      }

      switch (event.type) {
        case "system":
          formatSystem(event);
          break;
        case "stream_event": {
          const next = formatStreamEvent(event, streamedText, inText, textBuf);
          streamedText = next.streamedText;
          inText = next.inText;
          textBuf = next.textBuf;
          break;
        }
        case "assistant": {
          const ended = endTextStream(inText, textBuf);
          inText = ended.inText;
          textBuf = ended.textBuf;
          formatAssistant(event, streamedText, tools);
          streamedText = false;
          break;
        }
        case "user": {
          const ended = endTextStream(inText, textBuf);
          inText = ended.inText;
          textBuf = ended.textBuf;
          formatUser(event, tools);
          break;
        }
        case "result": {
          const ended = endTextStream(inText, textBuf);
          inText = ended.inText;
          textBuf = ended.textBuf;
          resultLine = line;
          resultFailed = Boolean(event.is_error);
          formatResult(event);
          break;
        }
        case "rate_limit_event":
          println("Rate limit notice — waiting to continue…");
          break;
      }
    },
    flush(failed) {
      const ended = endTextStream(inText, textBuf);
      inText = ended.inText;
      textBuf = ended.textBuf;
      tools.flush(Boolean(failed));
    },
    resultJSON() {
      if (resultLine) {
        return resultLine;
      }
      if (lastLine) {
        return lastLine;
      }
      return "{}";
    },
    // Claude Code's own "result" event is the authoritative verdict: headless
    // (-p) mode exits 0 even when the turn ended in an error (e.g. the API
    // key it was given is invalid), because the CLI still produced a result,
    // just one that says it failed. The process exit code alone would hide
    // that failure from the rest of the pipeline.
    resultFailed() {
      return resultFailed;
    },
  };
}

function writeLiveLogRecord(rec) {
  process.stdout.write(`${JSON.stringify(rec)}\n`);
}

function println(text = "") {
  process.stdout.write(`${text}\n`);
}

function createToolTracker() {
  const openTools = new Map();
  const fifo = [];
  let anonSeq = 0;

  function resolveKey(id, creating) {
    const key = id != null && String(id).trim() ? String(id).trim() : "";
    if (key) {
      return key;
    }
    if (creating) {
      const generated = `anon-${anonSeq}`;
      anonSeq += 1;
      fifo.push(generated);
      return generated;
    }
    return fifo.shift() || "";
  }

  return {
    start(kind, text, id) {
      const key = resolveKey(id, true);
      const startedAt = Date.now();
      openTools.set(key, { kind, text: text || kind, startedAt, emitted: false });
    },
    emitStart(id) {
      let key = id != null && String(id).trim() ? String(id).trim() : "";
      if (!key || !openTools.has(key)) {
        key = fifo[0] || key;
      }
      const tool = openTools.get(key);
      if (!tool || tool.emitted) {
        return key;
      }
      tool.emitted = true;
      writeLiveLogRecord({
        type: "tool_start",
        id: key,
        kind: tool.kind,
        text: tool.text,
        started_at: tool.startedAt,
      });
      return key;
    },
    end(failed, id) {
      this.emitStart(id);
      let key = resolveKey(id, false);
      if (!openTools.has(key)) {
        key = fifo.shift() || key;
      }
      const tool = openTools.get(key);
      if (!tool) {
        return;
      }
      openTools.delete(key);
      const fifoIndex = fifo.indexOf(key);
      if (fifoIndex >= 0) {
        fifo.splice(fifoIndex, 1);
      }
      writeLiveLogRecord({
        type: "tool_end",
        id: key,
        kind: tool.kind,
        status: failed ? "failed" : "passed",
        duration_ms: Math.max(0, Date.now() - tool.startedAt),
      });
    },
    flush(failed) {
      for (const key of [...openTools.keys()]) {
        this.end(failed, key);
      }
    },
  };
}

function formatSystem(event) {
  if (event.subtype !== "init") {
    if (event.subtype === "api_retry") {
      const attempt = event.attempt ?? "?";
      const maxRetries = event.max_retries ?? "?";
      const delay = event.retry_delay_ms;
      const delayPart = delay != null ? ` in ${delay}ms` : "";
      println(`Retrying API (${attempt}/${maxRetries})${delayPart}…`);
    }
    return;
  }

  const parts = ["Claude Code started"];
  if (event.model) {
    parts.push(`model=${event.model}`);
  }
  if (event.cwd) {
    parts.push(`cwd=${event.cwd}`);
  }
  println(parts.join(" · "));
  const tools = Array.isArray(event.tools) ? event.tools.map((tool) => String(tool)) : [];
  const planningTools = tools.filter((tool) => tool.includes("mcp__superplane"));
  println(planningTools.length > 0 ? `planning tools: ${planningTools.join(", ")}` : "planning tools: none");
  if (event.mcp_server_errors) {
    println(`mcp errors: ${JSON.stringify(event.mcp_server_errors)}`);
  }
  println();
}

function formatStreamEvent(event, streamedText, inText, textBuf) {
  const payload = event.event;
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
    return { streamedText, inText, textBuf };
  }

  const kind = payload.type;
  if (kind === "content_block_start") {
    const block = payload.content_block;
    if (block && typeof block === "object" && block.type === "text") {
      return { streamedText, inText: true, textBuf };
    }
    return { streamedText, inText, textBuf };
  }

  if (kind === "content_block_delta") {
    const delta = payload.delta;
    if (delta && typeof delta === "object" && delta.type === "text_delta") {
      const text = delta.text;
      if (typeof text === "string" && text) {
        // Buffer until newline or block end so live logs (one CloudWatch
        // event per flush chunk) do not show mid-word line breaks.
        return {
          streamedText: true,
          inText: true,
          textBuf: emitCompleteLines(textBuf + text),
        };
      }
    }
    return { streamedText, inText, textBuf };
  }

  if (kind === "content_block_stop" && inText) {
    const ended = endTextStream(true, textBuf);
    println();
    return { streamedText, inText: false, textBuf: ended.textBuf };
  }

  return { streamedText, inText, textBuf };
}

function emitCompleteLines(buf) {
  while (true) {
    const idx = buf.indexOf("\n");
    if (idx < 0) {
      return buf;
    }
    println(buf.slice(0, idx));
    buf = buf.slice(idx + 1);
  }
}

function endTextStream(inText, textBuf) {
  if (!inText && !textBuf) {
    return { inText: false, textBuf: "" };
  }
  if (textBuf) {
    println(textBuf);
  } else if (inText) {
    println();
  }
  return { inText: false, textBuf: "" };
}

function formatAssistant(event, streamedText, tools) {
  const message = event.message;
  if (!message || typeof message !== "object") {
    return;
  }
  const content = message.content;
  if (!Array.isArray(content)) {
    return;
  }

  for (const block of content) {
    if (!block || typeof block !== "object") {
      continue;
    }
    if (block.type === "text" && !streamedText) {
      const text = block.text;
      if (typeof text === "string" && text.trim()) {
        println(text.replace(/\s+$/, ""));
        println();
      }
    } else if (block.type === "tool_use") {
      formatToolUse(block, tools);
    } else if (block.type === "thinking") {
      const thinking = block.thinking;
      if (typeof thinking === "string" && thinking.trim()) {
        println("Thinking");
        println(truncateText(thinking.trim()));
        println();
      }
    }
  }
}

function formatUser(event, tools) {
  const message = event.message;
  if (!message || typeof message !== "object") {
    return;
  }
  const content = message.content;
  if (!Array.isArray(content)) {
    return;
  }

  for (const block of content) {
    if (!block || typeof block !== "object" || block.type !== "tool_result") {
      continue;
    }
    const body = toolResultText(block.content);
    tools.emitStart(block.tool_use_id);
    if (body.trim()) {
      println(truncateText(body.replace(/\s+$/, "")));
    }
    tools.end(Boolean(block.is_error), block.tool_use_id);
  }
}

function formatResult(event) {
  const isError = Boolean(event.is_error);
  const status = isError ? "failed" : "done";
  const parts = [isError ? `✗ ${status}` : `✓ ${status}`];

  if (event.num_turns != null) {
    parts.push(`${event.num_turns} turns`);
  }
  if (event.total_cost_usd != null) {
    const cost = Number(event.total_cost_usd);
    parts.push(Number.isFinite(cost) ? `$${cost.toFixed(4)}` : `$${event.total_cost_usd}`);
  }
  if (event.duration_ms != null) {
    const ms = Number(event.duration_ms);
    if (Number.isFinite(ms)) {
      parts.push(`${(ms / 1000).toFixed(1)}s`);
    }
  }

  println(parts.join(" · "));

  const result = event.result;
  if (typeof result === "string" && result.trim() && isError) {
    println(result.replace(/\s+$/, ""));
  }
}

function formatToolUse(block, tools) {
  const name = String(block.name || "tool");
  const kind = name.toLowerCase();
  tools.start(kind, toolInputDetail(name, block.input) || kind, block.id);
}

function toolInputDetail(name, rawInput) {
  if (rawInput == null || typeof rawInput !== "object" || Array.isArray(rawInput)) {
    if (rawInput == null) {
      return "";
    }
    return truncateText(String(rawInput));
  }

  const lowered = name.toLowerCase();
  if (lowered === "bash") {
    const command = rawInput.command;
    if (typeof command === "string" && command.trim()) {
      return command
        .trim()
        .split(/\r?\n/)
        .join(" ");
    }
  }
  if (["read", "write", "edit", "notebookedit"].includes(lowered)) {
    for (const key of ["file_path", "path", "notebook_path"]) {
      const value = rawInput[key];
      if (typeof value === "string" && value.trim()) {
        let detail = value.trim();
        if ((lowered === "write" || lowered === "edit") && typeof rawInput.content === "string") {
          detail += ` (${rawInput.content.length} chars)`;
        }
        return detail;
      }
    }
  }
  if (lowered === "grep") {
    const parts = [];
    if (rawInput.pattern) {
      parts.push(`pattern: ${rawInput.pattern}`);
    }
    if (rawInput.path) {
      parts.push(`path: ${rawInput.path}`);
    }
    if (parts.length) {
      return parts.join(" · ");
    }
  }
  if (lowered === "glob" && rawInput.pattern) {
    return String(rawInput.pattern);
  }

  try {
    return truncateText(JSON.stringify(rawInput));
  } catch {
    return truncateText(String(rawInput));
  }
}

function toolResultText(content) {
  if (content == null) {
    return "";
  }
  if (typeof content === "string") {
    return content;
  }
  if (Array.isArray(content)) {
    return content
      .map((item) => {
        if (item && typeof item === "object") {
          if (typeof item.text === "string") {
            return item.text;
          }
          return JSON.stringify(item);
        }
        return String(item);
      })
      .join("\n");
  }
  return String(content);
}

function truncateText(text) {
  let lines = text.split(/\r?\n/);
  if (lines.length > TOOL_RESULT_MAX_LINES) {
    const kept = lines.slice(0, TOOL_RESULT_MAX_LINES);
    const omitted = lines.length - TOOL_RESULT_MAX_LINES;
    text = `${kept.join("\n")}\n… (${omitted} more lines)`;
    lines = text.split(/\r?\n/);
  }
  if (text.length > TOOL_RESULT_MAX_CHARS) {
    text = `${text.slice(0, TOOL_RESULT_MAX_CHARS - 1).replace(/\s+$/, "")}…`;
  }
  return text;
}

function formatStreamJsonLines(rawLines) {
  const formatter = createFormatter();
  for (const line of rawLines) {
    formatter.handleLine(line);
  }
  const failed = formatter.resultFailed();
  formatter.flush(failed);
  return { failed };
}

if (require.main === module) {
  main();
}

module.exports = { allowedClaudeTools, claudePermissionMode, formatStreamJsonLines };
