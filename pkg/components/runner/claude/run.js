#!/usr/bin/env node
"use strict";

/**
 * Run Claude Code and format stream-json into readable live logs.
 *
 *   node run.js <prompt-file> [model]
 *   node run.js --format   # format NDJSON from stdin (tests / debug)
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

function main() {
  const args = process.argv.slice(2);
  if (args[0] === "--format") {
    formatStdin();
    return;
  }
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

function formatStdin() {
  const formatter = createFormatter();
  const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
  rl.on("line", (raw) => formatter.handleLine(raw));
  rl.on("close", () => formatter.flush());
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

  const workdirFile = path.join(sp, "workdir");
  if (fs.existsSync(workdirFile)) {
    const dir = fs.readFileSync(workdirFile, "utf8").trim();
    if (dir) {
      process.chdir(dir);
    }
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
    "acceptEdits",
    "--append-system-prompt",
    SYSTEM_PROMPT,
  ];
  if (model) {
    claudeArgs.push("--model", model);
  }
  claudeArgs.push("--allowedTools", "Bash,Read,Edit,Write");
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

  const streamPath = path.join(sp, "stream.jsonl");
  const streamFd = fs.openSync(streamPath, "a");
  const formatter = createFormatter();

  try {
    const child = spawn(command, args, {
      stdio: ["ignore", "pipe", "pipe"],
    });
    child.stderr.pipe(process.stderr);

    const rl = readline.createInterface({ input: child.stdout, crlfDelay: Infinity });
    rl.on("line", (raw) => {
      fs.writeSync(streamFd, `${raw}\n`);
      formatter.handleLine(raw);
    });

    const exitCode = await Promise.all([
      new Promise((resolve, reject) => {
        child.on("error", reject);
        child.on("close", (code) => resolve(code == null ? 1 : code));
      }),
      new Promise((resolve) => rl.on("close", resolve)),
    ]).then(([code]) => code);

    formatter.flush();
    fs.writeFileSync(resultFile, `${formatter.resultJSON()}\n`);
    fs.writeFileSync(promptCountPath, `${promptCount + 1}\n`);
    return exitCode;
  } finally {
    fs.closeSync(streamFd);
  }
}

function commandExists(name) {
  const result = spawnSync("sh", ["-c", `command -v ${name}`], { encoding: "utf8" });
  return result.status === 0;
}

function createFormatter() {
  let streamedText = false;
  let inText = false;
  let textBuf = "";
  let lastLine = "";
  let resultLine = "";

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
          formatAssistant(event, streamedText);
          streamedText = false;
          break;
        }
        case "user": {
          const ended = endTextStream(inText, textBuf);
          inText = ended.inText;
          textBuf = ended.textBuf;
          formatUser(event);
          break;
        }
        case "result": {
          const ended = endTextStream(inText, textBuf);
          inText = ended.inText;
          textBuf = ended.textBuf;
          resultLine = line;
          formatResult(event);
          break;
        }
        case "rate_limit_event":
          println("Rate limit notice — waiting to continue…");
          break;
      }
    },
    flush() {
      const ended = endTextStream(inText, textBuf);
      inText = ended.inText;
      textBuf = ended.textBuf;
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
  };
}

function println(text = "") {
  process.stdout.write(`${text}\n`);
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

function formatAssistant(event, streamedText) {
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
      println(formatToolUse(block));
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

function formatUser(event) {
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
    if (!body.trim()) {
      continue;
    }
    println(indent(truncateText(body.replace(/\s+$/, "")), "     "));
    println();
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

function formatToolUse(block) {
  const name = String(block.name || "tool");
  const detail = toolInputDetail(name, block.input);
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

function indent(text, prefix = "  ") {
  return text
    .split(/\r?\n/)
    .map((line) => (line ? prefix + line : prefix.replace(/\s+$/, "")))
    .join("\n");
}

main();
