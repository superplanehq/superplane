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

const PLANNING_SYSTEM_PROMPT =
  "This is a SuperPlane planning session. Call the propose_draft tool only when the user asked for a task in this turn. " +
  "Call the survey tool to ask questions. SuperPlane waits after you stop. Do not create work orders yourself. " +
  "When the user creates or skips a draft, acknowledge that in one short sentence and ask what they want to do next. " +
  "Do not call propose_draft unless they ask for a task. When the user starts a refine, read the current task, tell " +
  "them you are ready, and ask what they want to change. Do not call propose_draft until they say what to change. " +
  "Write to the user in plain text. Only explore the repository (read files, search, run read-only commands); do not " +
  "edit or write any files.";

function envFlag(env, name) {
  return Boolean(String((env && env[name]) || "").trim());
}

function planningEnabled(env = process.env) {
  return envFlag(env, "SUPERPLANE_PLANNING_SESSION_ID");
}

// Codex `exec` has no --ask-for-approval flag; approval_policy is set via a
// `-c` config override instead. `--sandbox read-only` still allows Bash/Read
// style exploration (shell commands, file reads) but rejects file writes, so
// planning sessions stay read-only without disabling tool use entirely.
function codexExecArgs(env = process.env, model, mcpScriptPath) {
  const args = ["exec", "--json", "--skip-git-repo-check"];
  if (planningEnabled(env)) {
    args.push("--sandbox", "read-only", "-c", "approval_policy=\"never\"");
    args.push(...mcpConfigOverrides(mcpScriptPath));
  } else {
    args.push("--dangerously-bypass-approvals-and-sandbox");
  }
  if (model) {
    args.push("-m", model);
  }
  return args;
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
  return `"${String(value).replace(/\\/g, "\\\\").replace(/"/g, '\\"')}"`;
}

function tomlStringArray(values) {
  return `[${values.map(tomlString).join(", ")}]`;
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

  let prompt = fs.readFileSync(promptFile, "utf8");
  const promptCountPath = path.join(sp, "prompt_count");
  const promptCount = Number.parseInt(fs.readFileSync(promptCountPath, "utf8").trim(), 10) || 0;

  const startedAt = Date.now();
  const planning = planningEnabled();
  const codexArgs = codexExecArgs(process.env, model, path.join(sp, "planning_session_mcp.js"));
  if (planning) {
    process.stdout.write("Planning session tools enabled\n");
    process.stdout.write("sandbox: read-only\n");
    prompt = `${prompt}\n\n${PLANNING_SYSTEM_PROMPT}`;
  }
  if (promptCount > 0) {
    process.stdout.write("Continuing Codex session in the current directory\n");
  }
  codexArgs.push(prompt);

  const child = spawn("codex", codexArgs, { stdio: ["ignore", "pipe", "pipe"] });
  child.stderr.pipe(process.stderr);

  let lastResult = {};
  const formatter = createCodexFormatter();
  const rl = readline.createInterface({ input: child.stdout, crlfDelay: Infinity });
  rl.on("line", (raw) => {
    const line = raw.trim();
    if (!line) {
      return;
    }
    try {
      const event = JSON.parse(line);
      if (event && typeof event === "object") {
        if (event.usage || (event.item && event.item.usage)) {
          lastResult = event;
        }
        if (event.type === "item.completed" || event.type === "turn.completed" || event.type === "result") {
          if (!lastResult.usage && !(lastResult.item && lastResult.item.usage)) {
            lastResult = event;
          }
        }
        formatter.handleEvent(event);
        return;
      }
    } catch (_err) {
      process.stdout.write(`${line}\n`);
    }
  });

  const exitCode = await Promise.all([
    new Promise((resolve, reject) => {
      child.on("error", reject);
      child.on("close", (code) => resolve(code == null ? 1 : code));
    }),
    new Promise((resolve) => rl.on("close", resolve)),
  ]).then(([code]) => code);

  formatter.flush(exitCode !== 0);
  const usage = extractUsage(lastResult);
  const payload = {
    type: "result",
    result: lastResult.result || lastResult.text || "",
    model: lastResult.model || model,
    usage,
  };
  fs.writeFileSync(resultFile, `${JSON.stringify(payload)}\n`);
  accumulateLLMUsage(payload);
  fs.writeFileSync(promptCountPath, `${promptCount + 1}\n`);
  formatTurnResult({
    is_error: exitCode !== 0,
    num_turns: 1,
    duration_ms: Date.now() - startedAt,
  });
  return exitCode;
}

function extractUsage(event) {
  const source = event.usage || (event.item && event.item.usage) || event;
  return {
    input_tokens: Number(source.input_tokens || source.prompt_tokens || 0),
    output_tokens: Number(source.output_tokens || source.completion_tokens || 0),
    cache_read_input_tokens: Number(source.cached_input_tokens || source.cache_read_input_tokens || 0),
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
  process.stdout.write(`${JSON.stringify(rec)}\n`);
}

function createCodexFormatter() {
  const open = new Map();
  const anonQueue = [];
  let anonSeq = 0;

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
    return id;
  }

  function emitStart(id) {
    const tracked = open.get(id);
    if (!tracked || tracked.emitted) {
      return;
    }
    tracked.emitted = true;
    writeLiveLogRecord({
      type: "tool_start",
      id,
      kind: tracked.kind,
      text: tracked.text,
      started_at: tracked.startedAt,
    });
  }

  function completeTool(item) {
    let id = itemID(item, false);
    if (!open.has(id)) {
      id = rememberTool(item);
    }
    emitStart(id);
    const output = item.aggregated_output || item.output || "";
    if (typeof output === "string" && output.trim()) {
      process.stdout.write(`${output.replace(/\s+$/, "")}\n`);
    }
    const tracked = open.get(id) || { kind: normalizeCodexToolKind(item), startedAt: Date.now() };
    open.delete(id);
    const anonIndex = anonQueue.indexOf(id);
    if (anonIndex >= 0) {
      anonQueue.splice(anonIndex, 1);
    }
    writeLiveLogRecord({
      type: "tool_end",
      id,
      kind: tracked.kind,
      status: toolFailed(item) ? "failed" : "passed",
      duration_ms: Math.max(0, Date.now() - tracked.startedAt),
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
        if (event.type === "item.completed") {
          const text = item.text || item.result || "";
          if (typeof text === "string" && text.trim() && type !== "reasoning") {
            process.stdout.write(`${text.replace(/\s+$/, "")}\n`);
          }
        }
        return;
      }
      if (!isToolItem(type)) {
        return;
      }
      if (event.type === "item.started") {
        rememberTool(item);
        return;
      }
      if (event.type === "item.completed") {
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
        writeLiveLogRecord({
          type: "tool_end",
          id,
          kind: tracked.kind,
          status: failed ? "failed" : "passed",
          duration_ms: Math.max(0, Date.now() - tracked.startedAt),
        });
        open.delete(id);
      }
    },
  };
}

function isMessageItem(type) {
  return type === "agent_message" || type === "assistant_message" || type === "reasoning";
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
  const kinds = changes.map((change) => String((change && change.kind) || "").toLowerCase());
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
    const pathValue = changes.map((change) => change && change.path).find(Boolean);
    return String(pathValue || item.path || "file");
  }
  return String(item.command || item.path || item.query || item.name || type || "tool");
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

function formatTurnResult(event) {
  const isError = Boolean(event && event.is_error);
  const status = isError ? "failed" : "done";
  const parts = [isError ? `✗ ${status}` : `✓ ${status}`];
  if (event && event.num_turns != null) {
    parts.push(`${event.num_turns} turns`);
  }
  if (event && event.total_cost_usd != null) {
    const cost = Number(event.total_cost_usd);
    parts.push(Number.isFinite(cost) ? `$${cost.toFixed(4)}` : `$${event.total_cost_usd}`);
  }
  if (event && event.duration_ms != null) {
    const ms = Number(event.duration_ms);
    if (Number.isFinite(ms)) {
      parts.push(`${(ms / 1000).toFixed(1)}s`);
    }
  }
  process.stdout.write(`${parts.join(" · ")}\n`);
}

function formatCodexJsonLines(rawLines) {
  const formatter = createCodexFormatter();
  for (const line of rawLines) {
    const trimmed = String(line).trim();
    if (!trimmed) {
      continue;
    }
    try {
      formatter.handleEvent(JSON.parse(trimmed));
    } catch (_err) {
      process.stdout.write(`${trimmed}\n`);
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
  planningEnabled,
};
