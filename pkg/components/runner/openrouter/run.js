#!/usr/bin/env node
"use strict";

/**
 * SuperPlane OpenRouter agent: bash, read, edit, write tools.
 * Planning sessions (SUPERPLANE_PLANNING_SESSION_ID set) drop edit/write and
 * add the propose_draft/survey planning tools instead.
 *
 *   node run.js <prompt-file> [model] [max-turns]
 */

const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

const DEFAULT_MAX_TURNS = 128;
const MAX_TURNS_LIMIT = 256;
const BASE_SYSTEM_PROMPT =
  "You are a coding agent on a SuperPlane fleet runner. Use bash, read, edit, and write tools. Write assistant messages as plain terminal text.";
const PLANNING_SYSTEM_PROMPT =
  "This is a SuperPlane planning session. Use the bash and read tools only to explore the repository for context; " +
  "do not edit or write any files. Call propose_draft only when the user asked for a task in this turn. Call survey " +
  "to ask questions. SuperPlane waits after you stop. Do not create work orders yourself. When the user creates or " +
  "skips a draft, acknowledge that in one short sentence and ask what they want to do next. Do not call propose_draft " +
  "unless they ask for a task. When the user starts a refine, read the current task, tell them you are ready, and " +
  "ask what they want to change. Do not call propose_draft until they say what to change. Write to the user in plain " +
  "text.";
const WRAP_UP_PROMPT =
  "You have no remaining tool turns. Do not call tools. Write a plain-text summary of what you completed and what remains.";
const TOOL_NUDGE =
  "Use the bash, read, edit, or write tools to do the work. Do not only describe the changes.";
const PLANNING_TOOL_NUDGE =
  "Use the bash or read tools to gather context, or call propose_draft/survey. Do not only describe the changes.";
const TOOLS = [
  {
    type: "function",
    function: {
      name: "bash",
      description: "Run a shell command in the current working directory.",
      parameters: {
        type: "object",
        properties: { command: { type: "string" } },
        required: ["command"],
      },
    },
  },
  {
    type: "function",
    function: {
      name: "read",
      description: "Read a UTF-8 text file.",
      parameters: {
        type: "object",
        properties: { path: { type: "string" } },
        required: ["path"],
      },
    },
  },
  {
    type: "function",
    function: {
      name: "write",
      description: "Write a UTF-8 text file, creating parent directories.",
      parameters: {
        type: "object",
        properties: { path: { type: "string" }, content: { type: "string" } },
        required: ["path", "content"],
      },
    },
  },
  {
    type: "function",
    function: {
      name: "edit",
      description: "Replace the first exact occurrence of old_text in a file with new_text.",
      parameters: {
        type: "object",
        properties: {
          path: { type: "string" },
          old_text: { type: "string" },
          new_text: { type: "string" },
        },
        required: ["path", "old_text", "new_text"],
      },
    },
  },
];
// Planning sessions may only explore the repo (bash is documented read-only
// in its tool description and the system prompt; edit/write are dropped
// entirely so the model has no structured way to change files).
const PLANNING_TOOLS = [
  {
    type: "function",
    function: {
      name: "bash",
      description: "Run a read-only shell command (inspect files, search, run tests). Do not modify the repository.",
      parameters: {
        type: "object",
        properties: { command: { type: "string" } },
        required: ["command"],
      },
    },
  },
  TOOLS[1], // read
];

function envFlag(env, name) {
  return Boolean(String((env && env[name]) || "").trim());
}

function planningEnabled(env = process.env) {
  return envFlag(env, "SUPERPLANE_PLANNING_SESSION_ID");
}

// The planning MCP server module is shipped alongside run.js only for
// planning sessions (see runner.PlanningSessionMCPScriptFile). Reuse its
// proposeDraft/proposeSurvey HTTP calls here instead of duplicating the
// request contract.
function loadPlanningHelpers(env = process.env) {
  const taskDir = env.SUPERPLANE_TASK_DIR;
  if (!taskDir) {
    return null;
  }
  const script = path.join(taskDir, "planning_session_mcp.js");
  if (!fs.existsSync(script)) {
    return null;
  }
  return require(script);
}

function toFunctionTool(def) {
  return {
    type: "function",
    function: { name: def.name, description: def.description, parameters: def.inputSchema },
  };
}

function planningToolDefs(helpers) {
  const defs = (helpers && Array.isArray(helpers.TOOLS) && helpers.TOOLS) || [];
  return [...PLANNING_TOOLS, ...defs.map(toFunctionTool)];
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    console.error("usage: node run.js <prompt-file> [model] [max-turns]");
    process.exit(2);
  }
  runPrompt(args[0], args[1] || "", args[2])
    .then((code) => process.exit(code))
    .catch((err) => {
      console.error(err && err.message ? err.message : err);
      process.exit(1);
    });
}

async function runPrompt(promptFile, model, maxTurns = DEFAULT_MAX_TURNS) {
  const resultFile = process.env.SUPERPLANE_RESULT_FILE;
  if (!resultFile) {
    throw new Error("SUPERPLANE_RESULT_FILE is required");
  }
  const apiKey = process.env.OPENROUTER_API_KEY;
  if (!apiKey) {
    throw new Error("OPENROUTER_API_KEY is required");
  }
  if (!model) {
    throw new Error("model is required");
  }

  const planning = planningEnabled(process.env);
  const planningHelpers = planning ? loadPlanningHelpers(process.env) : null;
  const tools = planning ? planningToolDefs(planningHelpers) : TOOLS;
  const toolNudge = planning ? PLANNING_TOOL_NUDGE : TOOL_NUDGE;
  if (planning) {
    process.stdout.write("Planning session tools enabled\n");
    process.stdout.write(`allowed tools: ${tools.map((tool) => tool.function.name).join(", ")}\n`);
  }

  const prompt = fs.readFileSync(promptFile, "utf8");
  const baseURL = (process.env.OPENROUTER_BASE_URL || "https://openrouter.ai/api/v1").replace(/\/$/, "");
  const messages = [
    { role: "system", content: planning ? PLANNING_SYSTEM_PROMPT : BASE_SYSTEM_PROMPT },
    { role: "user", content: prompt },
  ];

  const usage = {
    input_tokens: 0,
    output_tokens: 0,
    cache_read_input_tokens: 0,
    reasoning_tokens: 0,
  };
  let lastText = "";
  let costMicros = 0;
  let pendingToolCalls = false;
  let nudgedForTools = false;
  let numTurns = 0;
  let exitCode = 1;
  const startedAt = Date.now();
  const turnLimit = resolveMaxTurns(maxTurns);

  try {
    for (let turn = 0; turn < turnLimit; turn += 1) {
      numTurns += 1;
      const response = await chat(baseURL, apiKey, model, messages, true, tools);
      addUsage(usage, response.usage);
      costMicros += usageCostMicros(response.usage);

      const message = (response.choices && response.choices[0] && response.choices[0].message) || {};
      messages.push(message);
      const text = assistantText(message);
      if (text) {
        lastText = text;
        process.stdout.write(`${lastText}\n`);
      }

      const toolCalls = extractToolCalls(message);
      pendingToolCalls = toolCalls.length > 0;
      if (!pendingToolCalls) {
        if (!nudgedForTools) {
          nudgedForTools = true;
          process.stderr.write("OpenRouter agent returned no tool calls; asking it to use tools\n");
          messages.push({ role: "user", content: toolNudge });
          continue;
        }
        break;
      }
      for (const call of toolCalls) {
        const name = call.function && call.function.name;
        const args = parseArgs(call.function && call.function.arguments);
        const kind = String(name || "tool").toLowerCase();
        const startedAt = Date.now();
        writeLiveLogRecord({
          type: "tool_start",
          id: call.id,
          kind,
          text: toolPreview(kind, args),
          started_at: startedAt,
        });
        const result = await dispatchTool(name, args, { planning, planningHelpers });
        if (result.output) {
          process.stdout.write(`${result.output}\n`);
        }
        writeLiveLogRecord({
          type: "tool_end",
          id: call.id,
          kind,
          status: result.failed ? "failed" : "passed",
          duration_ms: Math.max(0, Date.now() - startedAt),
        });
        messages.push({
          role: "tool",
          tool_call_id: call.id,
          content: result.output,
        });
      }
    }

    if (pendingToolCalls) {
      const wrapUp = await requestWrapUp(baseURL, apiKey, model, messages, usage, lastText, turnLimit);
      lastText = wrapUp.text;
      costMicros += wrapUp.costMicros;
      numTurns += 1;
    }
    exitCode = 0;
    return 0;
  } catch (err) {
    console.error(err && err.message ? err.message : err);
    exitCode = 1;
    return 1;
  } finally {
    writeResult(resultFile, model, usage, costMicros, lastText);
    formatTurnResult({
      is_error: exitCode !== 0,
      num_turns: numTurns || 1,
      total_cost_usd: costMicros > 0 ? costMicros / 1000000 : undefined,
      duration_ms: Date.now() - startedAt,
    });
  }
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

function writeResult(resultFile, model, usage, costMicros, lastText) {
  const payload = {
    type: "result",
    result: lastText,
    model,
    usage,
  };
  if (costMicros > 0) {
    payload.total_cost_usd = costMicros / 1000000;
  }
  fs.writeFileSync(resultFile, `${JSON.stringify(payload)}\n`);
  accumulateLLMUsage(payload);
}

function resolveMaxTurns(maxTurns) {
  const n = Number(maxTurns);
  if (!(n > 0)) {
    return DEFAULT_MAX_TURNS;
  }
  return Math.min(Math.floor(n), MAX_TURNS_LIMIT);
}

function extractToolCalls(message) {
  if (!message) {
    return [];
  }
  if (Array.isArray(message.tool_calls) && message.tool_calls.length > 0) {
    return message.tool_calls;
  }
  if (message.function_call && message.function_call.name) {
    return [
      {
        id: String(message.function_call.name) + "-legacy",
        type: "function",
        function: message.function_call,
      },
    ];
  }
  if (!Array.isArray(message.content)) {
    return [];
  }
  const calls = [];
  for (const part of message.content) {
    const fn = (part && (part.functionCall || part.function_call)) || null;
    if (!fn || !fn.name) {
      continue;
    }
    const rawArgs = fn.arguments || fn.args || {};
    calls.push({
      id: String(fn.id || fn.name),
      type: "function",
      function: {
        name: fn.name,
        arguments: typeof rawArgs === "string" ? rawArgs : JSON.stringify(rawArgs),
      },
    });
  }
  return calls;
}

function assistantText(message) {
  if (!message || !message.content) {
    return "";
  }
  if (Array.isArray(message.content)) {
    return message.content.map((part) => part.text || "").join("");
  }
  return String(message.content);
}

async function requestWrapUp(baseURL, apiKey, model, messages, usage, lastText, turnLimit) {
  process.stderr.write(
    `OpenRouter agent reached ${turnLimit} turns; requesting a final response without tools\n`,
  );
  messages.push({ role: "user", content: WRAP_UP_PROMPT });
  const response = await chat(baseURL, apiKey, model, messages, false);
  addUsage(usage, response.usage);
  const message = (response.choices && response.choices[0] && response.choices[0].message) || {};
  const text = assistantText(message);
  if (text) {
    process.stdout.write(`${text}\n`);
  }
  return {
    text: text || lastText,
    costMicros: usageCostMicros(response.usage),
  };
}

async function chat(baseURL, apiKey, model, messages, withTools, tools = TOOLS) {
  const payload = {
    model,
    messages,
    usage: { include: true },
  };
  if (withTools) {
    payload.tools = tools;
    payload.tool_choice = "auto";
    payload.provider = { require_parameters: true };
  }
  const response = await fetch(`${baseURL}/chat/completions`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${apiKey}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  const body = await response.json();
  if (!response.ok) {
    throw new Error(body.error && body.error.message ? body.error.message : `OpenRouter HTTP ${response.status}`);
  }
  return body;
}

function addUsage(total, usage) {
  if (!usage) {
    return;
  }
  total.input_tokens += Number(usage.prompt_tokens || usage.input_tokens || 0);
  total.output_tokens += Number(usage.completion_tokens || usage.output_tokens || 0);
  const details = usage.prompt_tokens_details || {};
  total.cache_read_input_tokens += Number(details.cached_tokens || usage.cache_read_input_tokens || 0);
  const completionDetails = usage.completion_tokens_details || {};
  total.reasoning_tokens += Number(completionDetails.reasoning_tokens || usage.reasoning_tokens || 0);
}

function usageCostMicros(usage) {
  if (!usage || usage.cost == null) {
    return 0;
  }
  return Math.round(Number(usage.cost) * 1000000);
}

function parseArgs(raw) {
  if (!raw) {
    return {};
  }
  if (typeof raw === "object") {
    return raw;
  }
  try {
    return JSON.parse(raw);
  } catch (_err) {
    return {};
  }
}

// dispatchTool routes bash/read/write/edit through the synchronous local
// tool runner and propose_draft/survey through the async planning helpers
// (network calls to SuperPlane). Planning sessions never see write/edit in
// their tool list, but reject them here too in case a model hallucinates a
// call to a tool it was not offered.
async function dispatchTool(name, args, ctx) {
  if (ctx.planning) {
    if (name === "propose_draft" || name === "survey") {
      return runPlanningTool(name, args, ctx.planningHelpers);
    }
    if (name === "write" || name === "edit") {
      return { output: `${name} is not available in a planning session`, failed: true };
    }
  }
  return runTool(name, args);
}

async function runPlanningTool(name, args, helpers) {
  if (!helpers) {
    return { output: "planning tools are not available in this run", failed: true };
  }
  try {
    const result = name === "propose_draft" ? await helpers.proposeDraft(args) : await helpers.proposeSurvey(args);
    return { output: JSON.stringify(result), failed: false };
  } catch (err) {
    return { output: err && err.message ? err.message : String(err), failed: true };
  }
}

function runTool(name, args) {
  try {
    if (name === "bash") {
      const result = spawnSync("bash", ["-lc", String(args.command || "")], {
        encoding: "utf8",
        cwd: process.cwd(),
        env: process.env,
        timeout: 120000,
        maxBuffer: 2 * 1024 * 1024,
      });
      const output = `${result.stdout || ""}${result.stderr || ""}`.trim();
      if (result.status !== 0) {
        return { output: output || `command failed with exit ${result.status}`, failed: true };
      }
      return { output: output || "(no output)", failed: false };
    }
    if (name === "read") {
      return { output: fs.readFileSync(args.path, "utf8"), failed: false };
    }
    if (name === "write") {
      fs.mkdirSync(path.dirname(args.path), { recursive: true });
      fs.writeFileSync(args.path, args.content ?? "", "utf8");
      return { output: `wrote ${args.path}`, failed: false };
    }
    if (name === "edit") {
      const current = fs.readFileSync(args.path, "utf8");
      if (!current.includes(args.old_text)) {
        return { output: "old_text not found", failed: true };
      }
      fs.writeFileSync(args.path, current.replace(args.old_text, args.new_text), "utf8");
      return { output: `edited ${args.path}`, failed: false };
    }
    return { output: `unknown tool: ${name}`, failed: true };
  } catch (err) {
    return { output: err && err.message ? err.message : String(err), failed: true };
  }
}

function writeLiveLogRecord(rec) {
  process.stdout.write(`${JSON.stringify(rec)}\n`);
}

function toolPreview(kind, args) {
  if (!args || typeof args !== "object") {
    return kind;
  }
  if (kind === "bash" && typeof args.command === "string" && args.command.trim()) {
    return args.command.trim();
  }
  if (kind === "propose_draft" && typeof args.title === "string" && args.title.trim()) {
    return args.title.trim();
  }
  if (typeof args.path === "string" && args.path.trim()) {
    return args.path.trim();
  }
  return kind;
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

if (require.main === module) {
  main();
}

module.exports = { runPrompt, formatTurnResult, DEFAULT_MAX_TURNS, MAX_TURNS_LIMIT, planningEnabled };
