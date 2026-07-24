#!/usr/bin/env node
"use strict";

/**
 * Run OpenCode headless and format its JSON event stream into readable
 * live logs.
 *
 *   node run.js <prompt-file> <provider/model>
 *
 * OpenCode emits JSONL on stdout with `--format json`. Event shapes differ
 * from Claude Code's stream-json, so this formatter handles OpenCode's own
 * events (step_start / text / tool_use / step_finish / error) rather than
 * reusing Claude's handler.
 *
 * Session continuity: the first prompt has no session; this script captures
 * the session id emitted by OpenCode into $SUPERPLANE_TASK_DIR/session_id so
 * that later prompt steps resume the exact same session via `--session`.
 */

const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { spawn, spawnSync } = require("child_process");

const TOOL_RESULT_MAX_CHARS = 800;
const TOOL_RESULT_MAX_LINES = 24;

function main() {
  const args = process.argv.slice(2);
  if (args.length < 2) {
    console.error("usage: node run.js <prompt-file> <provider/model>");
    process.exit(2);
  }
  runPrompt(args[0], args[1])
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
  if (!model) {
    throw new Error("model is required (provider/model)");
  }

  const prompt = fs.readFileSync(promptFile, "utf8");
  const sessionPath = path.join(sp, "session_id");
  const previousSession = readSessionID(sessionPath);

  const openCodeArgs = ["run", "--format", "json", "--model", model];
  if (previousSession) {
    // Resume the exact session started by an earlier prompt step. This is
    // safer than `--continue`, which resumes the most recent session in the
    // directory and is unsafe when tasks share a machine/workdir.
    openCodeArgs.push("--session", previousSession);
  }
  openCodeArgs.push(prompt);

  let command = "opencode";
  let args = openCodeArgs;
  if (commandExists("stdbuf")) {
    command = "stdbuf";
    args = ["-oL", "-eL", "opencode", ...openCodeArgs];
  }

  const formatter = createFormatter(model, previousSession);
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

  formatter.flush();

  const sessionID = formatter.sessionID() || previousSession;
  if (sessionID) {
    fs.writeFileSync(sessionPath, `${sessionID}\n`);
  }
  fs.writeFileSync(resultFile, `${formatter.resultJSON()}\n`);

  // An OpenCode `error` event is a failure even if the process exits 0.
  if (exitCode === 0 && formatter.hasError()) {
    return 1;
  }
  return exitCode;
}

function readSessionID(sessionPath) {
  try {
    const raw = fs.readFileSync(sessionPath, "utf8").trim();
    return raw || "";
  } catch {
    return "";
  }
}

function commandExists(name) {
  const result = spawnSync("sh", ["-c", `command -v ${name}`], { encoding: "utf8" });
  return result.status === 0;
}

function createFormatter(model, initialSession) {
  let announced = false;
  let sessionID = initialSession || "";
  let inText = false;
  let textBuf = "";
  let lastText = "";
  let errorMessage = "";
  let lastLine = "";

  function captureSession(event) {
    const candidate =
      event.sessionID ||
      event.session_id ||
      event.sessionId ||
      (event.session && (event.session.id || event.session.sessionID)) ||
      (event.info && (event.info.sessionID || event.info.session_id));
    if (typeof candidate === "string" && candidate.trim()) {
      sessionID = candidate.trim();
    }
  }

  function announce() {
    if (announced) {
      return;
    }
    announced = true;
    const parts = ["OpenCode started"];
    if (model) {
      parts.push(`model=${model}`);
    }
    if (sessionID) {
      parts.push(`session=${sessionID}`);
    }
    println(parts.join(" · "));
    println();
  }

  function endText() {
    const ended = endTextStream(inText, textBuf);
    inText = ended.inText;
    textBuf = ended.textBuf;
  }

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
        // Not JSON — surface it as-is so nothing is silently swallowed.
        println(line);
        return;
      }
      if (!event || typeof event !== "object" || Array.isArray(event)) {
        return;
      }

      captureSession(event);

      switch (normalizeType(event.type)) {
        case "step_start":
          announce();
          break;
        case "text": {
          announce();
          const text = extractText(event);
          if (text) {
            lastText = text;
            inText = true;
            textBuf = emitCompleteLines(textBuf + text);
          }
          break;
        }
        case "tool_use":
        case "tool": {
          announce();
          endText();
          println(formatToolUse(event));
          break;
        }
        case "step_finish":
        case "finish": {
          endText();
          println();
          formatStepFinish(event);
          break;
        }
        case "error": {
          endText();
          errorMessage = extractError(event);
          println(`✗ error${errorMessage ? `: ${errorMessage}` : ""}`);
          break;
        }
        default:
          break;
      }
    },
    flush() {
      endText();
    },
    sessionID() {
      return sessionID;
    },
    hasError() {
      return Boolean(errorMessage);
    },
    resultJSON() {
      const result = {
        type: "result",
        session_id: sessionID,
        result: lastText,
      };
      if (errorMessage) {
        result.is_error = true;
        result.error = errorMessage;
      }
      try {
        return JSON.stringify(result);
      } catch {
        return lastLine || "{}";
      }
    },
  };
}

function normalizeType(type) {
  if (typeof type !== "string") {
    return "";
  }
  return type.trim().toLowerCase().replace(/-/g, "_");
}

function extractText(event) {
  const candidates = [event.text, event.content, event.part && event.part.text, event.delta];
  for (const candidate of candidates) {
    if (typeof candidate === "string" && candidate) {
      return candidate;
    }
  }
  return "";
}

function extractError(event) {
  const candidates = [
    event.message,
    event.error && (typeof event.error === "string" ? event.error : event.error.message),
    event.text,
  ];
  for (const candidate of candidates) {
    if (typeof candidate === "string" && candidate.trim()) {
      return candidate.trim();
    }
  }
  return "";
}

function println(text = "") {
  process.stdout.write(`${text}\n`);
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

function formatStepFinish(event) {
  const reason = event.reason || event.finishReason || event.finish_reason;
  const isError = reason && String(reason).toLowerCase() !== "stop" && String(reason).toLowerCase() !== "end_turn";
  const parts = [isError ? `✗ ${reason}` : "✓ done"];

  const cost = event.cost != null ? event.cost : event.total_cost_usd;
  if (cost != null) {
    const value = Number(cost);
    parts.push(Number.isFinite(value) ? `$${value.toFixed(4)}` : `$${cost}`);
  }

  const tokens = extractTokens(event);
  if (tokens) {
    parts.push(`${tokens} tokens`);
  }

  println(parts.join(" · "));
  println();
}

function extractTokens(event) {
  const tokens = event.tokens || event.usage;
  if (!tokens || typeof tokens !== "object") {
    if (typeof event.tokens === "number") {
      return event.tokens;
    }
    return 0;
  }
  const input = Number(tokens.input || tokens.input_tokens || tokens.prompt_tokens || 0);
  const output = Number(tokens.output || tokens.output_tokens || tokens.completion_tokens || 0);
  const total = input + output;
  return Number.isFinite(total) && total > 0 ? total : 0;
}

function formatToolUse(event) {
  const name = String(event.tool || event.name || (event.part && event.part.tool) || "tool");
  const input = event.input || event.args || (event.part && event.part.state && event.part.state.input) || event.parameters;
  const detail = toolInputDetail(name, input);
  const header = `-> [${name}]`;
  if (!detail) {
    return header;
  }
  if (detail.includes("\n")) {
    return `${header}\n${indent(detail, "     ")}`;
  }
  return `${header} ${detail}`;
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
      return command.trim().split(/\r?\n/).join(" ");
    }
  }
  if (["read", "write", "edit"].includes(lowered)) {
    for (const key of ["filePath", "file_path", "path"]) {
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

function indent(text, prefix = "  ") {
  return text
    .split(/\r?\n/)
    .map((line) => (line ? prefix + line : prefix.replace(/\s+$/, "")))
    .join("\n");
}

main();
