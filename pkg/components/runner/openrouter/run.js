#!/usr/bin/env node
"use strict";

/**
 * SuperPlane OpenRouter agent: bash, read, edit, write tools.
 *
 *   node run.js <prompt-file> [model] [max-turns]
 */

const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

const DEFAULT_MAX_TURNS = 128;
const MAX_TURNS_LIMIT = 256;
const WRAP_UP_PROMPT =
  "You have no remaining tool turns. Do not call tools. Write a plain-text summary of what you completed and what remains.";
const TOOL_NUDGE =
  "Use the bash, read, edit, or write tools to do the work. Do not only describe the changes.";
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

  const prompt = fs.readFileSync(promptFile, "utf8");
  const baseURL = (process.env.OPENROUTER_BASE_URL || "https://openrouter.ai/api/v1").replace(/\/$/, "");
  const messages = [
    {
      role: "system",
      content:
        "You are a coding agent on a SuperPlane fleet runner. Use bash, read, edit, and write tools. Write assistant messages as plain terminal text.",
    },
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
  const turnLimit = resolveMaxTurns(maxTurns);

  try {
    for (let turn = 0; turn < turnLimit; turn += 1) {
      const response = await chat(baseURL, apiKey, model, messages, true);
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
          messages.push({ role: "user", content: TOOL_NUDGE });
          continue;
        }
        break;
      }
      for (const call of toolCalls) {
        const name = call.function && call.function.name;
        const args = parseArgs(call.function && call.function.arguments);
        const output = runTool(name, args);
        process.stdout.write(`[${name}] ${summarize(output)}\n`);
        messages.push({
          role: "tool",
          tool_call_id: call.id,
          content: output,
        });
      }
    }

    if (pendingToolCalls) {
      const wrapUp = await requestWrapUp(baseURL, apiKey, model, messages, usage, lastText, turnLimit);
      lastText = wrapUp.text;
      costMicros += wrapUp.costMicros;
    }
    return 0;
  } catch (err) {
    console.error(err && err.message ? err.message : err);
    return 1;
  } finally {
    writeResult(resultFile, model, usage, costMicros, lastText);
  }
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

async function chat(baseURL, apiKey, model, messages, withTools) {
  const payload = {
    model,
    messages,
    usage: { include: true },
  };
  if (withTools) {
    payload.tools = TOOLS;
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
        return output || `command failed with exit ${result.status}`;
      }
      return output || "(no output)";
    }
    if (name === "read") {
      return fs.readFileSync(args.path, "utf8");
    }
    if (name === "write") {
      fs.mkdirSync(path.dirname(args.path), { recursive: true });
      fs.writeFileSync(args.path, args.content ?? "", "utf8");
      return `wrote ${args.path}`;
    }
    if (name === "edit") {
      const current = fs.readFileSync(args.path, "utf8");
      if (!current.includes(args.old_text)) {
        return "old_text not found";
      }
      fs.writeFileSync(args.path, current.replace(args.old_text, args.new_text), "utf8");
      return `edited ${args.path}`;
    }
    return `unknown tool: ${name}`;
  } catch (err) {
    return err && err.message ? err.message : String(err);
  }
}

function summarize(text) {
  const line = String(text || "").split(/\r?\n/)[0];
  return line.length > 120 ? `${line.slice(0, 117)}...` : line;
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

module.exports = { runPrompt, DEFAULT_MAX_TURNS, MAX_TURNS_LIMIT };
