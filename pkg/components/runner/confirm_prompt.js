#!/usr/bin/env node
"use strict";

const { spawn } = require("child_process");

const CONFIRM_PROMPT_RULE =
  "Do not run a command that waits for a person. When the target file already exists, decide from the task whether to keep it or replace it. If you replace it, pass the overwrite flag. If you keep it, do not run the installer in a way that asks. If a command prints a confirm prompt, stop that command and continue the task. Do not end the run. Do not wait for a person to answer.";

const CONFIRM_PROMPT_BLOCK_MS = 800;

const CONFIRM_PROMPT_PATTERNS = [
  /\(y\/n\)/i,
  /\[y\/n\]/i,
  /\(yes\/no\)/i,
  /\[yes\/no\]/i,
  /\b(?:Do you|Would you|Shall I|Are you sure|Ready to)\b.*\?/i,
  /Overwrite\?/i,
  /already exists\b.*\?/i,
  /Press (?:any key|Enter)/i,
  /Continue\?/i,
];

function stripAnsi(text) {
  return String(text || "").replace(/\u001b\[[0-9;]*[A-Za-z]/g, "");
}

function lastOutputLine(text) {
  const lines = stripAnsi(text).replace(/\r/g, "\n").trimEnd().split("\n");
  return (lines[lines.length - 1] || "").trim();
}

function isBlockedConfirmLine(text) {
  const line = lastOutputLine(text);
  if (!line || line.length > 400) {
    return false;
  }
  return CONFIRM_PROMPT_PATTERNS.some((pattern) => pattern.test(line));
}

function stopChild(child, stopped) {
  if (!child || !child.pid || stopped.current) {
    return;
  }
  stopped.current = true;
  try {
    process.kill(-child.pid, "SIGTERM");
  } catch (_err) {
    try {
      child.kill("SIGTERM");
    } catch (_killErr) {
      return;
    }
  }
  setTimeout(() => {
    try {
      process.kill(-child.pid, "SIGKILL");
    } catch (_err) {
      try {
        child.kill("SIGKILL");
      } catch (_killErr) {
        return;
      }
    }
  }, 200).unref();
}

function runConfirmPromptGuard(script, io = process) {
  return new Promise((resolve) => {
    const command = String(script || "");
    if (!command.trim()) {
      resolve(2);
      return;
    }
    const child = spawn("bash", ["-c", command], {
      detached: true,
      stdio: ["inherit", "pipe", "pipe"],
    });
    let output = "";
    let blockedSince = 0;
    const stopped = { current: false };
    const note = (chunk, stream) => {
      stream.write(chunk);
      output += chunk.toString("utf8");
      blockedSince = isBlockedConfirmLine(output) ? Date.now() : 0;
    };
    child.stdout.on("data", (chunk) => note(chunk, io.stdout));
    child.stderr.on("data", (chunk) => note(chunk, io.stderr));
    const timer = setInterval(() => {
      if (!blockedSince || Date.now() - blockedSince < CONFIRM_PROMPT_BLOCK_MS) {
        return;
      }
      clearInterval(timer);
      stopChild(child, stopped);
    }, 100);
    const finish = (code) => {
      clearInterval(timer);
      resolve(code);
    };
    child.on("error", () => finish(1));
    child.on("close", (code, signal) => {
      if (stopped.current || signal) {
        finish(1);
        return;
      }
      finish(code == null ? 1 : code);
    });
  });
}

if (require.main === module) {
  runConfirmPromptGuard(process.argv[2]).then((code) => {
    process.exit(code);
  });
}

module.exports = {
  CONFIRM_PROMPT_RULE,
  isBlockedConfirmLine,
  runConfirmPromptGuard,
};
