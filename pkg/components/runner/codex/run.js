#!/usr/bin/env node
"use strict";

/**
 * Run Codex CLI and emit typed live-log records.
 *
 *   node run.js <prompt-file> [model]
 */

const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { spawn } = require("child_process");

const SESSION_FILE = "codex_session";

function loadActivityStreamModule() {
  const taskDir = process.env.SUPERPLANE_TASK_DIR || "";
  const candidates = [
    path.join(taskDir, "activity_stream.js"),
    path.join(__dirname, "..", "activity_stream.js"),
  ];
  for (const file of candidates) {
    if (file && fs.existsSync(file)) {
      return require(file);
    }
  }
  return { createActivityStream: () => createDisabledActivityStream() };
}

function createDisabledActivityStream() {
  const noop = () => undefined;
  return {
    enabled: false,
    activityId: "",
    start: noop,
    startContent: noop,
    appendContent: noop,
    endContent: noop,
    startTool: (input) => String((input && input.id) || ""),
    appendToolOutput: noop,
    endTool: noop,
    notice: noop,
    end: noop,
    flush: () => Promise.resolve(),
  };
}

function loadAnalysisProtocolModule() {
  const candidates = [
    path.join(__dirname, "analysis_protocol.js"),
    path.join(__dirname, "..", "analysis_protocol.js"),
  ];
  for (const file of candidates) {
    try {
      return require(file);
    } catch (_err) {
      // try the next path
    }
  }
  return {};
}

function loadAnalysisProtocol() {
  const mod = loadAnalysisProtocolModule();
  return typeof mod.analysisProtocol === "function"
    ? mod.analysisProtocol()
    : "";
}

function withoutEmbeddedAnalysisProtocol(prompt) {
  const mod = loadAnalysisProtocolModule();
  if (typeof mod.withoutEmbeddedAnalysisProtocol === "function") {
    return mod.withoutEmbeddedAnalysisProtocol(prompt);
  }
  return prompt;
}

function applyAnalysisContinuation(taskDir, promptCount, prompt) {
  if (!planningAnalysisEnabled()) {
    return prompt;
  }
  const mod = loadAnalysisProtocolModule();
  if (typeof mod.withAnalysisContinuation !== "function") {
    return prompt;
  }
  return mod.withAnalysisContinuation(taskDir, promptCount, prompt);
}

function envFlag(env, name) {
  return Boolean(String((env && env[name]) || "").trim());
}

function planningEnabled(env = process.env) {
  return (
    planningAnalysisEnabled(env) &&
    envFlag(env, "SUPERPLANE_PLANNING_SESSION_ID")
  );
}

function planningAnalysisEnabled(env = process.env) {
  return env.SUPERPLANE_PLANNING_SESSION_KIND === "work_order_analysis";
}

function artifactEnabled(env = process.env) {
  return envFlag(env, "SUPERPLANE_ARTIFACT_TOKEN");
}

function planningSystemPrompt(env = process.env) {
  return planningAnalysisEnabled(env) ? loadAnalysisProtocol() : "";
}

// Codex `exec` has no --ask-for-approval flag, and `exec resume` has no
// --sandbox flag. Config overrides keep both new and resumed analysis turns
// read-only without disabling shell commands and file reads.
function codexExecArgs(
  env = process.env,
  model,
  mcpScriptPath,
  sessionID = "",
) {
  const args = ["exec"];
  if (sessionID) {
    args.push("resume", sessionID);
  }
  args.push("--json", "--skip-git-repo-check");
  if (planningEnabled(env)) {
    args.push(
      "-c",
      'sandbox_mode="read-only"',
      "-c",
      'approval_policy="never"',
    );
    args.push(...mcpConfigOverrides(mcpScriptPath));
    args.push(
      "-c",
      `developer_instructions=${tomlString(loadAnalysisProtocol())}`,
    );
  } else {
    args.push("--dangerously-bypass-approvals-and-sandbox");
    if (artifactEnabled(env)) args.push(...mcpConfigOverrides(mcpScriptPath));
  }
  if (model) {
    args.push("-m", model);
  }
  return args;
}

function readSessionID(taskDir) {
  const file = path.join(taskDir, SESSION_FILE);
  if (!fs.existsSync(file)) {
    return "";
  }
  return fs.readFileSync(file, "utf8").trim();
}

function writeSessionID(taskDir, sessionID) {
  const id = String(sessionID || "").trim();
  if (id) {
    fs.writeFileSync(path.join(taskDir, SESSION_FILE), `${id}\n`);
  }
}

function codexSessionForPrompt(promptCount, sessionID, env = process.env) {
  if (planningAnalysisEnabled(env) && envFlag(env, "SUPERPLANE_ANALYSIS_REWIND")) {
    return "";
  }
  if (Number(promptCount) < 1) {
    return "";
  }
  const id = String(sessionID || "").trim();
  if (!id) {
    throw new Error("Codex session ID is missing for a follow-up prompt");
  }
  return id;
}

function codexSessionIDFromEvent(event) {
  return String(
    (event && (event.thread_id || (event.thread && event.thread.id))) || "",
  ).trim();
}

function mcpConfigOverrides(mcpScriptPath) {
  return [
    "-c",
    `mcp_servers.superplane.command=${tomlString("node")}`,
    "-c",
    `mcp_servers.superplane.args=${tomlStringArray([mcpScriptPath])}`,
  ];
}

function tomlString(value) {
  return `"${String(value)
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\n/g, "\\n")
    .replace(/\r/g, "\\r")}"`;
}

function tomlStringArray(values) {
  return `[${values.map(tomlString).join(", ")}]`;
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    writeStderr("usage: node run.js <prompt-file> [model]\n");
    process.exit(2);
  }
  runPrompt(args[0], args[1] || "")
    .then((code) => process.exit(code))
    .catch((err) => {
      writeStderr(`${err && err.message ? err.message : err}\n`);
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

  const promptCountPath = path.join(sp, "prompt_count");
  const promptCount =
    Number.parseInt(fs.readFileSync(promptCountPath, "utf8").trim(), 10) || 0;
  const sessionID = codexSessionForPrompt(promptCount, readSessionID(sp));
  let prompt = applyAnalysisContinuation(
    sp,
    sessionID ? promptCount : 0,
    fs.readFileSync(promptFile, "utf8"),
  );
  if (planningAnalysisEnabled()) {
    prompt = withoutEmbeddedAnalysisProtocol(prompt);
  }

  const startedAt = Date.now();
  if (artifactEnabled()) {
    const outputDir = path.join(sp, "evidence");
    fs.mkdirSync(outputDir, { recursive: true });
    process.env.PLAYWRIGHT_MCP_OUTPUT_DIR = outputDir;
    process.env.PLAYWRIGHT_MCP_BROWSER =
      process.env.PLAYWRIGHT_MCP_BROWSER || "chromium";
  }
  const planning = planningEnabled();
  const activity = loadActivityStreamModule().createActivityStream({
    provider: "codex",
    turn: promptCount + 1,
  });
  activity.start();
  const mcpScript = path.join(
    sp,
    planning ? "planning_session_mcp.js" : "task_artifact_mcp.js",
  );
  const codexArgs = codexExecArgs(process.env, model, mcpScript, sessionID);
  if (planning) {
    writeStdout("Planning session tools enabled\n");
    writeStdout("sandbox: read-only\n");
  }
  if (sessionID) {
    writeStdout("Continuing Codex session in the current directory\n");
  }
  codexArgs.push(prompt);

  const child = spawn("codex", codexArgs, {
    stdio: ["ignore", "pipe", "pipe"],
  });
  const stderrDone = pipeRedactedStderr(child.stderr);

  let lastResult = {};
  const telemetry = loadTurnTelemetry();
  const formatter = createCodexFormatter(telemetry, activity);
  const rl = readline.createInterface({
    input: child.stdout,
    crlfDelay: Infinity,
  });
  rl.on("line", (raw) => {
    const line = raw.trim();
    if (!line) {
      return;
    }
    try {
      const event = JSON.parse(line);
      if (event && typeof event === "object") {
        const nextSessionID = codexSessionIDFromEvent(event);
        if (nextSessionID) {
          writeSessionID(sp, nextSessionID);
        }
        if (event.usage || (event.item && event.item.usage)) {
          lastResult = event;
        }
        if (
          event.type === "item.completed" ||
          event.type === "turn.completed" ||
          event.type === "result"
        ) {
          if (
            !lastResult.usage &&
            !(lastResult.item && lastResult.item.usage)
          ) {
            lastResult = event;
          }
        }
        formatter.handleEvent(event);
        return;
      }
    } catch (_err) {
      writeStdout(`${line}\n`);
    }
  });

  const exitCode = await Promise.all([
    new Promise((resolve, reject) => {
      child.on("error", reject);
      child.on("close", (code) => resolve(code == null ? 1 : code));
    }),
    new Promise((resolve) => rl.on("close", resolve)),
    stderrDone,
  ]).then(([code]) => code);

  formatter.flush(exitCode !== 0);
  activity.end(exitCode === 0 ? "passed" : "failed");
  await activity.flush();
  const usage = extractUsage(lastResult);
  if (tokenTotal(usage) > 0) {
    telemetry.updateCurrentUsage(usage);
  }
  const payload = {
    type: "result",
    result: formatter.lastText() || lastResult.result || lastResult.text || "",
    model: lastResult.model || model,
    usage,
  };
  telemetry.attachToResult(payload);
  const activityModule = loadActivityStreamModule();
  const safePayload = activityModule.sanitizeLogValue ? activityModule.sanitizeLogValue(payload) : payload;
  fs.writeFileSync(resultFile, `${JSON.stringify(safePayload)}\n`);
  accumulateLLMUsage(safePayload);
  fs.writeFileSync(promptCountPath, `${promptCount + 1}\n`);
  if (planning) {
    await require(path.join(sp, "planning_session_mcp.js")).recordAgentMessage(
      safePayload.result,
    );
  }
  formatTurnResult({
    is_error: exitCode !== 0,
    num_turns:
      safePayload.telemetry && safePayload.telemetry.num_turns
        ? safePayload.telemetry.num_turns
        : 1,
    duration_ms: Date.now() - startedAt,
  });
  return exitCode;
}

function tokenTotal(usage) {
  if (!usage || typeof usage !== "object") {
    return 0;
  }
  return (
    Number(usage.input_tokens || 0) +
    Number(usage.output_tokens || 0) +
    Number(usage.cache_read_input_tokens || 0) +
    Number(usage.reasoning_tokens || 0)
  );
}

function loadTurnTelemetry() {
  const taskDir = process.env.SUPERPLANE_TASK_DIR;
  const candidates = [];
  if (taskDir) {
    candidates.push(path.join(taskDir, "turn_telemetry.js"));
  }
  candidates.push(path.join(__dirname, "..", "turn_telemetry.js"));
  for (const file of candidates) {
    if (fs.existsSync(file)) {
      return require(file).createTurnTelemetry({
        write: writeLiveLogRecord,
        sanitize: loadActivityStreamModule().sanitizeLogValue,
      });
    }
  }
  return require("../turn_telemetry").createTurnTelemetry({
    write: writeLiveLogRecord,
    sanitize: loadActivityStreamModule().sanitizeLogValue,
  });
}

function extractUsage(event) {
  const source = event.usage || (event.item && event.item.usage) || event;
  return {
    input_tokens: Number(source.input_tokens || source.prompt_tokens || 0),
    output_tokens: Number(
      source.output_tokens || source.completion_tokens || 0,
    ),
    cache_read_input_tokens: Number(
      source.cached_input_tokens || source.cache_read_input_tokens || 0,
    ),
    reasoning_tokens: Number(source.reasoning_tokens || 0),
  };
}

function accumulateLLMUsage(payload) {
  const taskDir = process.env.SUPERPLANE_TASK_DIR;
  if (!taskDir) {
    return;
  }
  const script = path.join(taskDir, "llm_usage.js");
  if (!fs.existsSync(script)) {
    return;
  }
  require(script).accumulate(taskDir, payload);
}

function writeLiveLogRecord(rec) {
  const activity = loadActivityStreamModule();
  const safe = activity.sanitizeLogValue ? activity.sanitizeLogValue(rec) : rec;
  process.stdout.write(`${JSON.stringify(safe)}\n`);
}

function writeStdout(value) {
  const activity = loadActivityStreamModule();
  const safe = activity.sanitizeLogText ? activity.sanitizeLogText(value) : String(value);
  process.stdout.write(safe);
}

function writeStderr(value) {
  const activity = loadActivityStreamModule();
  const safe = activity.sanitizeLogText ? activity.sanitizeLogText(value) : String(value);
  process.stderr.write(safe);
}

function pipeRedactedStderr(stream) {
  const lines = readline.createInterface({ input: stream, crlfDelay: Infinity });
  lines.on("line", (line) => {
    writeStderr(`${line}\n`);
  });
  return new Promise((resolve) => lines.on("close", resolve));
}

function createCodexFormatter(telemetry, activityOverride) {
  const tracker = telemetry || loadTurnTelemetry();
  const activity =
    activityOverride ||
    loadActivityStreamModule().createActivityStream({ provider: "codex" });
  const open = new Map();
  const anonQueue = [];
  const contentText = new Map();
  const anonymousContent = new Map();
  let anonSeq = 0;
  let contentSeq = 0;
  let roundOpen = false;
  let lastText = "";

  function itemType(item) {
    return String((item && (item.type || item.item_type)) || "").toLowerCase();
  }

  function itemID(item, creating) {
    const id = item && item.id != null ? String(item.id).trim() : "";
    if (id) {
      return id;
    }
    if (creating) {
      const generated = `anon-${anonSeq}`;
      anonSeq += 1;
      anonQueue.push(generated);
      return generated;
    }
    return anonQueue.shift() || "";
  }

  function contentID(item, type, creating) {
    const id = item && item.id != null ? String(item.id).trim() : "";
    if (id) {
      return id;
    }
    const queue = anonymousContent.get(type) || [];
    if (creating) {
      const generated = `${type}-${contentSeq}`;
      contentSeq += 1;
      queue.push(generated);
      anonymousContent.set(type, queue);
      return generated;
    }
    return queue.shift() || `${type}-${contentSeq++}`;
  }

  function appendContentDelta(contentKind, id, text) {
    if (typeof text !== "string" || !text) {
      return;
    }
    const previous = contentText.get(id) || "";
    const delta = text.startsWith(previous)
      ? text.slice(previous.length)
      : text;
    activity.appendContent(contentKind, id, delta);
    contentText.set(id, text);
  }

  function rememberTool(item) {
    const id = itemID(item, true);
    if (open.has(id)) {
      return id;
    }
    open.set(id, {
      kind: normalizeCodexToolKind(item),
      text: toolTextForItem(item),
      startedAt: Date.now(),
      emitted: false,
    });
    activity.startTool({
      id,
      kind: normalizeCodexToolKind(item),
      name: String(item.name || item.type || "tool"),
      input: toolTextForItem(item),
    });
    return id;
  }

  function emitStart(id) {
    const tracked = open.get(id);
    if (!tracked || tracked.emitted) {
      return;
    }
    tracked.emitted = true;
    writeLiveLogRecord(
      tracker.stampToolStart({
        type: "tool_start",
        id,
        kind: tracked.kind,
        text: tracked.text,
        started_at: tracked.startedAt,
      }),
    );
  }

  function completeTool(item) {
    let id = itemID(item, false);
    if (!open.has(id)) {
      id = rememberTool(item);
    }
    emitStart(id);
    const output = item.aggregated_output || item.output || "";
    if (typeof output === "string" && output.trim()) {
      if (activity.enabled) {
        activity.appendToolOutput(
          id,
          output.replace(/\s+$/, ""),
          toolFailed(item) ? "stderr" : "stdout",
        );
      } else {
        writeStdout(`${output.replace(/\s+$/, "")}\n`);
      }
    }
    const tracked = open.get(id) || {
      kind: normalizeCodexToolKind(item),
      startedAt: Date.now(),
    };
    open.delete(id);
    const anonIndex = anonQueue.indexOf(id);
    if (anonIndex >= 0) {
      anonQueue.splice(anonIndex, 1);
    }
    writeLiveLogRecord(
      tracker.stampToolEnd({
        type: "tool_end",
        id,
        kind: tracked.kind,
        status: toolFailed(item) ? "failed" : "passed",
        duration_ms: Math.max(0, Date.now() - tracked.startedAt),
      }),
    );
    activity.endTool(id, {
      status: codexToolStatus(item),
      exitCode: item.exit_code,
      signal: item.signal,
    });
  }

  return {
    handleEvent(event) {
      const item = event.item;
      if (!item || typeof item !== "object") {
        return;
      }
      const type = itemType(item);
      if (isMessageItem(type)) {
        const contentKind = type === "reasoning" ? "reasoning" : "assistant";
        const id = contentID(item, type, event.type === "item.started");
        if (event.type === "item.started") {
          activity.startContent(contentKind, id);
          const initialText = item.text || item.result || "";
          appendContentDelta(contentKind, id, initialText);
          return;
        }
        if (event.type === "item.completed") {
          const text = item.text || item.result || "";
          if (!roundOpen) {
            tracker.beginTurn(item.usage || event.usage, {
              message:
                typeof text === "string" && type !== "reasoning"
                  ? text
                  : undefined,
            });
          }
          roundOpen = false;
          activity.startContent(contentKind, id);
          appendContentDelta(contentKind, id, text);
          activity.endContent(id);
          contentText.delete(id);
          if (typeof text === "string" && text.trim() && type !== "reasoning") {
            lastText = text.replace(/\s+$/, "");
            if (!activity.enabled) {
              writeStdout(`${lastText}\n`);
            }
          }
        }
        return;
      }
      if (!isToolItem(type)) {
        return;
      }
      if (event.type === "item.started") {
        if (!roundOpen) {
          tracker.beginTurn(item.usage || event.usage);
          roundOpen = true;
        }
        const id = rememberTool(item);
        emitStart(id);
        return;
      }
      if (event.type === "item.completed") {
        if (!roundOpen) {
          tracker.beginTurn(item.usage || event.usage);
          roundOpen = true;
        }
        completeTool(item);
      }
    },
    flush(failed) {
      for (const id of [...open.keys()]) {
        emitStart(id);
        const tracked = open.get(id);
        if (!tracked) {
          continue;
        }
        writeLiveLogRecord(
          tracker.stampToolEnd({
            type: "tool_end",
            id,
            kind: tracked.kind,
            status: failed ? "failed" : "passed",
            duration_ms: Math.max(0, Date.now() - tracked.startedAt),
          }),
        );
        activity.endTool(id, { status: failed ? "failed" : "interrupted" });
        open.delete(id);
      }
    },
    lastText() {
      return lastText;
    },
  };
}

function isMessageItem(type) {
  return (
    type === "agent_message" ||
    type === "assistant_message" ||
    type === "reasoning"
  );
}

function isToolItem(type) {
  return (
    type === "command_execution" ||
    type === "file_change" ||
    type === "mcp_tool_call" ||
    type === "web_search" ||
    type === "tool_call" ||
    type === "bash" ||
    type === "read" ||
    type === "edit" ||
    type === "write"
  );
}

function normalizeCodexToolKind(item) {
  const type = String(item.type || item.item_type || "").toLowerCase();
  if (type === "command_execution" || type === "bash") {
    return "bash";
  }
  if (type === "file_change") {
    return fileChangeKind(item);
  }
  if (type === "mcp_tool_call") {
    return String(item.tool || item.name || "mcp").toLowerCase();
  }
  if (type === "web_search") {
    return "web_search";
  }
  if (type === "read" || type === "edit" || type === "write") {
    return type;
  }
  return type || "tool";
}

function fileChangeKind(item) {
  const changes = Array.isArray(item.changes) ? item.changes : [];
  const kinds = changes.map((change) =>
    String((change && change.kind) || "").toLowerCase(),
  );
  if (kinds.includes("add") && !kinds.includes("update")) {
    return "write";
  }
  if (kinds.length > 0) {
    return "edit";
  }
  return "edit";
}

function toolTextForItem(item) {
  const type = String(item.type || item.item_type || "").toLowerCase();
  if (type === "command_execution" || type === "bash") {
    return stripBashLc(String(item.command || item.text || "bash"));
  }
  if (type === "file_change") {
    const changes = Array.isArray(item.changes) ? item.changes : [];
    const paths = changes
      .map((change) => change && change.path)
      .filter(Boolean);
    return paths.length > 0
      ? paths.map(String).join("\n")
      : String(item.path || "file");
  }
  return String(
    item.command || item.path || item.query || item.name || type || "tool",
  );
}

function stripBashLc(command) {
  return command.replace(/^bash\s+-lc\s+/, "").trim() || command;
}

function toolFailed(item) {
  if (item.status === "failed") {
    return true;
  }
  const exit = Number(item.exit_code);
  return Number.isFinite(exit) && exit !== 0;
}

function codexToolStatus(item) {
  const status = String((item && item.status) || "").toLowerCase();
  if (status === "cancelled" || status === "canceled") return "cancelled";
  if (status === "timed_out" || status === "timeout") return "timed_out";
  if (status === "interrupted") return "interrupted";
  return toolFailed(item) ? "failed" : "passed";
}

function formatTurnResult(event) {
  const isError = Boolean(event && event.is_error);
  const status = isError ? "failed" : "done";
  const parts = [isError ? `✗ ${status}` : `✓ ${status}`];
  if (event && event.num_turns != null) {
    parts.push(`${event.num_turns} turns`);
  }
  if (event && event.total_cost_usd != null) {
    const cost = Number(event.total_cost_usd);
    parts.push(
      Number.isFinite(cost)
        ? `$${cost.toFixed(4)}`
        : `$${event.total_cost_usd}`,
    );
  }
  if (event && event.duration_ms != null) {
    const ms = Number(event.duration_ms);
    if (Number.isFinite(ms)) {
      parts.push(`${(ms / 1000).toFixed(1)}s`);
    }
  }
  writeStdout(`${parts.join(" · ")}\n`);
}

function formatCodexJsonLines(rawLines) {
  const formatter = createCodexFormatter(loadTurnTelemetry());
  for (const line of rawLines) {
    const trimmed = String(line).trim();
    if (!trimmed) {
      continue;
    }
    try {
      formatter.handleEvent(JSON.parse(trimmed));
    } catch (_err) {
      writeStdout(`${trimmed}\n`);
    }
  }
  formatter.flush();
}

if (require.main === module) {
  main();
}

module.exports = {
  formatCodexJsonLines,
  formatTurnResult,
  createCodexFormatter,
  normalizeCodexToolKind,
  codexExecArgs,
  codexSessionForPrompt,
  codexSessionIDFromEvent,
  planningEnabled,
  planningSystemPrompt,
  planningAnalysisEnabled,
};
