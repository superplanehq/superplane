#!/usr/bin/env node
"use strict";

/**
 * Drive local Claude Code with the same SuperPlane run.js, turn telemetry,
 * and usage merge as a remote fleet runner.
 *
 * Host-only. Do not embed this in a broker task. Uses the `claude` CLI on
 * PATH and that CLI's login or ANTHROPIC_API_KEY.
 *
 *   node test/runner/claude/run_local.js --model sonnet "List files here"
 *   node test/runner/claude/run_local.js --model sonnet --prompt-file implement.txt --prompt-file pr.txt
 *   node test/runner/claude/run_local.js --task-dir /tmp/sp-claude --keep --model sonnet "Try again"
 */

const fs = require("fs");
const os = require("os");
const path = require("path");
const { spawnSync } = require("child_process");

const REPO_ROOT = path.join(__dirname, "..", "..", "..");
const RUNNER_DIR = path.join(REPO_ROOT, "pkg", "components", "runner");
const CLAUDE_DIR = path.join(RUNNER_DIR, "claude");
const HELPERS = [
  { from: path.join(CLAUDE_DIR, "run.js"), to: "run.js" },
  { from: path.join(RUNNER_DIR, "llm_usage.js"), to: "llm_usage.js" },
  { from: path.join(RUNNER_DIR, "turn_telemetry.js"), to: "turn_telemetry.js" },
];

function parseArgs(argv) {
  const options = {
    model: "",
    workDir: process.cwd(),
    taskDir: "",
    keep: true,
    clean: false,
    help: false,
    promptFiles: [],
    promptText: "",
  };
  const rest = [];
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--help" || arg === "-h") {
      options.help = true;
      continue;
    }
    if (arg === "--keep") {
      options.keep = true;
      options.clean = false;
      continue;
    }
    if (arg === "--clean") {
      options.clean = true;
      options.keep = false;
      continue;
    }
    if (arg === "--model" || arg === "--dir" || arg === "--task-dir" || arg === "--prompt-file") {
      const value = argv[i + 1];
      if (value == null || value.startsWith("--")) {
        throw new Error(`${arg} needs a value`);
      }
      i += 1;
      if (arg === "--model") {
        options.model = value;
      } else if (arg === "--dir") {
        options.workDir = path.resolve(value);
      } else if (arg === "--task-dir") {
        options.taskDir = path.resolve(value);
        options.keep = true;
        options.clean = false;
      } else {
        options.promptFiles.push(path.resolve(value));
      }
      continue;
    }
    if (arg.startsWith("--")) {
      throw new Error(`unknown option ${arg}`);
    }
    rest.push(arg);
  }
  if (rest.length > 0) {
    options.promptText = rest.join(" ");
  }
  return options;
}

function helpText() {
  return `Drive local Claude Code with the SuperPlane runner scripts.

Usage:
  node test/runner/claude/run_local.js [options] [prompt text]
  node test/runner/claude/run_local.js --prompt-file <file> [--prompt-file <file> ...]

Options:
  --model NAME         Claude Code model (same as the remote --model flag)
  --dir PATH           Working directory for Claude (default: current directory)
  --prompt-file PATH   Prompt file. Repeat for more than one prompt in one session
  --task-dir PATH      Reuse a task directory (keeps the directory)
  --keep               Keep the task directory (default)
  --clean              Remove a temporary task directory after the run
  --help               Show this help

Examples:
  node test/runner/claude/run_local.js --model sonnet "List the files in this directory"
  node test/runner/claude/run_local.js --model sonnet --prompt-file implement.txt --prompt-file pr.txt

Requires \`claude\` and \`node\` on PATH. Uses your local Claude Code login.
Does not dispatch a SuperPlane work order.
`;
}

function copyHelpers(taskDir) {
  for (const helper of HELPERS) {
    fs.copyFileSync(helper.from, path.join(taskDir, helper.to));
  }
}

function writePromptFiles(taskDir, options) {
  const promptsDir = path.join(taskDir, "prompts");
  fs.mkdirSync(promptsDir, { recursive: true });
  const files = [];
  let index = 1;
  for (const source of options.promptFiles) {
    const name = `${String(index).padStart(2, "0")}-${path.basename(source)}`;
    const dest = path.join(promptsDir, name);
    fs.copyFileSync(source, dest);
    files.push(dest);
    index += 1;
  }
  if (options.promptText.trim()) {
    const dest = path.join(promptsDir, `${String(index).padStart(2, "0")}-prompt.txt`);
    fs.writeFileSync(dest, `${options.promptText.trim()}\n`);
    files.push(dest);
  }
  return files;
}

function prepareTaskDir(options) {
  const taskDir = options.taskDir || fs.mkdtempSync(path.join(os.tmpdir(), "sp-claude-"));
  fs.mkdirSync(taskDir, { recursive: true });
  copyHelpers(taskDir);
  const promptCountPath = path.join(taskDir, "prompt_count");
  if (!fs.existsSync(promptCountPath)) {
    fs.writeFileSync(promptCountPath, "0\n");
  }
  fs.writeFileSync(path.join(taskDir, "task_cwd"), `${path.resolve(options.workDir)}\n`);
  const promptFiles = writePromptFiles(taskDir, options);
  if (promptFiles.length === 0) {
    throw new Error("give prompt text or --prompt-file");
  }
  return {
    taskDir,
    promptFiles,
    resultFile: path.join(taskDir, "result.json"),
    ephemeral: options.clean && !options.taskDir,
  };
}

function runPrompt(taskDir, resultFile, promptFile, model, workDir) {
  const runJs = path.join(taskDir, "run.js");
  const args = [runJs, promptFile];
  if (model) {
    args.push(model);
  }
  const result = spawnSync(process.execPath, args, {
    cwd: workDir,
    env: {
      ...process.env,
      SUPERPLANE_TASK_DIR: taskDir,
      SUPERPLANE_RESULT_FILE: resultFile,
    },
    stdio: "inherit",
  });
  if (result.error) {
    throw result.error;
  }
  const merge = spawnSync(process.execPath, [path.join(taskDir, "llm_usage.js"), "merge"], {
    cwd: workDir,
    env: {
      ...process.env,
      SUPERPLANE_TASK_DIR: taskDir,
      SUPERPLANE_RESULT_FILE: resultFile,
    },
    stdio: "inherit",
  });
  if (merge.error) {
    throw merge.error;
  }
  return result.status == null ? 1 : result.status;
}

function readJSON(file) {
  if (!fs.existsSync(file)) {
    return null;
  }
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (_err) {
    return null;
  }
}

function summarizeResult(result) {
  if (!result || typeof result !== "object") {
    return "no result.json";
  }
  const usage = result.usage && typeof result.usage === "object" ? result.usage : {};
  const parts = [];
  if (result.num_turns != null) {
    parts.push(`${result.num_turns} turns`);
  }
  if (result.total_cost_usd != null) {
    parts.push(`$${Number(result.total_cost_usd).toFixed(4)}`);
  }
  const input = Number(usage.input_tokens) || 0;
  const output = Number(usage.output_tokens) || 0;
  const cacheRead = Number(usage.cache_read_input_tokens) || 0;
  const cacheWrite = Number(usage.cache_creation_input_tokens) || 0;
  parts.push(`${input} input`);
  parts.push(`${output} output`);
  if (cacheWrite) {
    parts.push(`${cacheWrite} cache write`);
  }
  if (cacheRead) {
    parts.push(`${cacheRead} cache read`);
  }
  return parts.join(" · ");
}

function summarizeTelemetry(taskDir, result) {
  const seriesFile = path.join(taskDir, "turn_telemetry_series.json");
  const series = readJSON(seriesFile);
  const rows = [];
  if (series && Array.isArray(series.series)) {
    for (const item of series.series) {
      const telemetry = (item && item.telemetry) || item;
      const name = (item && item.name) || "prompt";
      const turns = telemetry && Array.isArray(telemetry.turns) ? telemetry.turns.length : 0;
      rows.push(`${name}: ${turns} recorded turns`);
    }
  } else if (result && result.telemetry && Array.isArray(result.telemetry.turns)) {
    rows.push(`telemetry: ${result.telemetry.turns.length} recorded turns`);
  }
  return rows;
}

function printSummary(taskDir, resultFile) {
  const result = readJSON(resultFile);
  const lines = [
    "",
    `task dir: ${taskDir}`,
    `result:   ${resultFile}`,
    `claude:   ${summarizeResult(result)}`,
    ...summarizeTelemetry(taskDir, result).map((line) => `           ${line}`),
  ];
  process.stderr.write(`${lines.join("\n")}\n`);
}

function requireOnPath(name) {
  const result = spawnSync("sh", ["-c", `command -v ${name}`], { encoding: "utf8" });
  if (result.status !== 0) {
    throw new Error(`${name} is not on PATH`);
  }
}

async function main(argv = process.argv.slice(2)) {
  const options = parseArgs(argv);
  if (options.help) {
    process.stdout.write(helpText());
    return 0;
  }
  requireOnPath("node");
  requireOnPath("claude");
  const prepared = prepareTaskDir(options);
  let code = 0;
  try {
    for (const promptFile of prepared.promptFiles) {
      const next = runPrompt(prepared.taskDir, prepared.resultFile, promptFile, options.model, options.workDir);
      if (next !== 0) {
        code = next;
        break;
      }
    }
  } finally {
    printSummary(prepared.taskDir, prepared.resultFile);
    if (prepared.ephemeral) {
      fs.rmSync(prepared.taskDir, { recursive: true, force: true });
    }
  }
  return code;
}

main()
  .then((code) => process.exit(code))
  .catch((err) => {
    console.error(err && err.message ? err.message : err);
    process.exit(1);
  });
