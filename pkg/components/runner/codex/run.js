#!/usr/bin/env node
"use strict";

/**
 * Run Codex CLI and write usage to SUPERPLANE_RESULT_FILE.
 *
 *   node run.js <prompt-file> [model]
 */

const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { spawn } = require("child_process");

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

  const codexArgs = ["exec", "--json", "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox"];
  if (model) {
    codexArgs.push("-m", model);
  }
  if (promptCount > 0) {
    process.stdout.write("Continuing Codex session in the current directory\n");
  }
  codexArgs.push(prompt);

  const child = spawn("codex", codexArgs, { stdio: ["ignore", "pipe", "pipe"] });
  child.stderr.pipe(process.stderr);

  let lastResult = {};
  const rl = readline.createInterface({ input: child.stdout, crlfDelay: Infinity });
  rl.on("line", (raw) => {
    const line = raw.trim();
    if (!line) {
      return;
    }
    process.stdout.write(`${line}\n`);
    try {
      const event = JSON.parse(line);
      if (event && typeof event === "object") {
        if (event.usage || (event.item && event.item.usage)) {
          lastResult = event;
          return;
        }
        if (event.type === "item.completed" || event.type === "turn.completed" || event.type === "result") {
          if (!lastResult.usage && !(lastResult.item && lastResult.item.usage)) {
            lastResult = event;
          }
        }
      }
    } catch (_err) {
      // Codex may print non-JSON progress lines.
    }
  });

  const exitCode = await Promise.all([
    new Promise((resolve, reject) => {
      child.on("error", reject);
      child.on("close", (code) => resolve(code == null ? 1 : code));
    }),
    new Promise((resolve) => rl.on("close", resolve)),
  ]).then(([code]) => code);

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

main();
