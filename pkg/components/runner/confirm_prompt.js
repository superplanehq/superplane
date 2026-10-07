#!/usr/bin/env node
"use strict";

const fs = require("fs");
const { spawn } = require("child_process");

const CONFIRM_PROMPT_RULE =
  "Do not run a command that waits for a person. When the target file already exists, decide from the task whether to keep it or replace it. If you replace it, pass the overwrite flag. If you keep it, do not run the installer in a way that asks. If a command prints a confirm prompt, stop that command and continue the task. Do not end the run. Do not wait for a person to answer.";

const CONFIRM_PROMPT_BLOCK_MS = 800;
const CPU_SLACK_TICKS = 5;

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
  /\bok to proceed\?/i,
  /\?\s*\(y\)\s*$/i,
];

const TIMER_WAIT = /^(?:hrtimer_nanosleep|do_nanosleep)$/;
const TTY_WAIT = /^(?:n_tty_read|tty_read)$/;

function stripAnsi(text) {
  return String(text || "").replace(/\u001b\[[0-9;]*[A-Za-z]/g, "");
}

function createLineTracker() {
  let pending = "";
  let lastLine = "";
  const publish = () => {
    const current = pending.trim();
    return (current || lastLine).slice(-400);
  };
  return {
    push(chunk) {
      pending += chunk.toString("utf8");
      if (pending.length > 2048) {
        pending = pending.slice(-2048);
      }
      const parts = stripAnsi(pending).replace(/\r/g, "\n").split("\n");
      pending = parts.pop() || "";
      if (pending.length > 500) {
        pending = pending.slice(-500);
      }
      for (const part of parts) {
        const trimmed = part.trim();
        if (trimmed) {
          lastLine = trimmed.slice(-400);
        }
      }
      return publish();
    },
    current: publish,
  };
}

function isBlockedConfirmLine(text) {
  const line = String(text || "").trim();
  if (!line || line.length > 400) {
    return false;
  }
  return CONFIRM_PROMPT_PATTERNS.some((pattern) => pattern.test(line));
}

function readProc(pid, name) {
  try {
    return fs.readFileSync(`/proc/${pid}/${name}`, "utf8");
  } catch (_err) {
    return "";
  }
}

function processSnapshot(pid) {
  const stat = readProc(pid, "stat");
  const close = stat.lastIndexOf(")");
  if (close < 0) {
    return null;
  }
  const fields = stat.slice(close + 2).trim().split(/\s+/);
  const ioText = readProc(pid, "io");
  let io = 0;
  for (const row of ioText.split("\n")) {
    if (row.startsWith("read_bytes:") || row.startsWith("write_bytes:")) {
      io += Number(row.split(/\s+/)[1] || 0);
    }
  }
  const syscall = readProc(pid, "syscall").trim().split(/\s+/);
  return {
    state: fields[0] || "",
    cpu: Number(fields[11] || 0) + Number(fields[12] || 0),
    io,
    wchan: readProc(pid, "wchan").trim(),
    stdinRead: syscall[0] === "0" && syscall[1] === "0x0",
  };
}

function descendantPids(root) {
  const pids = [];
  const seen = new Set();
  const walk = (pid) => {
    const id = String(pid || "");
    if (!id || seen.has(id)) {
      return;
    }
    seen.add(id);
    pids.push(id);
    for (const child of readProc(id, `task/${id}/children`).trim().split(/\s+/)) {
      if (child) {
        walk(child);
      }
    }
  };
  walk(root);
  return pids;
}

function treeActivity(pid) {
  const activity = {
    seen: false,
    cpu: 0,
    io: 0,
    running: false,
    timerWait: false,
    inputWait: false,
  };
  for (const id of descendantPids(pid)) {
    const snap = processSnapshot(id);
    if (!snap) {
      continue;
    }
    activity.seen = true;
    activity.cpu += snap.cpu;
    activity.io += snap.io;
    if (snap.state === "R" || snap.state === "D") {
      activity.running = true;
    }
    if (TIMER_WAIT.test(snap.wchan)) {
      activity.timerWait = true;
    }
    if (snap.stdinRead || TTY_WAIT.test(snap.wchan)) {
      activity.inputWait = true;
    }
  }
  return activity;
}

function commandBlocked(pid, line) {
  const activity = treeActivity(pid);
  if (!activity.seen || activity.running) {
    return null;
  }
  const prompt = isBlockedConfirmLine(line);
  if (activity.timerWait && !activity.inputWait) {
    return null;
  }
  if (!prompt && !activity.inputWait) {
    return null;
  }
  return activity;
}

function collectTargets(roots) {
  const pids = [];
  const seen = new Set();
  for (const root of roots) {
    for (const id of descendantPids(root)) {
      if (seen.has(id)) {
        continue;
      }
      seen.add(id);
      pids.push(id);
    }
  }
  return pids;
}

function signalPids(pids, signal) {
  for (let i = pids.length - 1; i >= 0; i -= 1) {
    try {
      process.kill(Number(pids[i]), signal);
    } catch (_err) {
      continue;
    }
  }
}

function stopChild(child, stopped, signal = "SIGTERM") {
  if (!child || !child.pid) {
    return;
  }
  const pid = child.pid;
  stopped.targets = collectTargets([pid, ...(stopped.targets || [])]);
  if (signal === "SIGKILL") {
    if (stopped.timer) {
      clearTimeout(stopped.timer);
      stopped.timer = null;
    }
    signalPids(stopped.targets, "SIGKILL");
    stopped.current = true;
    return;
  }
  if (stopped.current) {
    return;
  }
  stopped.current = true;
  signalPids(stopped.targets, signal);
  stopped.timer = setTimeout(() => {
    stopped.targets = collectTargets([pid, ...(stopped.targets || [])]);
    signalPids(stopped.targets, "SIGKILL");
  }, 200);
  if (stopped.timer.unref) {
    stopped.timer.unref();
  }
}

function runConfirmPromptGuard(script, io = process) {
  return new Promise((resolve) => {
    const command = String(script || "");
    if (!command.trim()) {
      resolve(2);
      return;
    }
    const child = spawn("bash", ["-c", command], {
      stdio: ["inherit", "pipe", "pipe"],
    });
    const tracker = createLineTracker();
    const stopped = { current: false };
    let watch = null;
    const halt = (signal) => stopChild(child, stopped, signal);
    const onSignal = () => {
      halt("SIGKILL");
      process.exit(1);
    };
    const onExit = () => halt("SIGKILL");
    process.on("SIGTERM", onSignal);
    process.on("SIGINT", onSignal);
    process.on("SIGHUP", onSignal);
    process.on("exit", onExit);
    child.stdout.on("data", (chunk) => {
      io.stdout.write(chunk);
      tracker.push(chunk);
    });
    child.stderr.on("data", (chunk) => {
      io.stderr.write(chunk);
      tracker.push(chunk);
    });
    const timer = setInterval(() => {
      if (!child.pid || stopped.current) {
        return;
      }
      const activity = commandBlocked(child.pid, tracker.current());
      if (!activity) {
        watch = null;
        return;
      }
      if (
        !watch ||
        activity.cpu > watch.cpu + CPU_SLACK_TICKS ||
        activity.io > watch.io
      ) {
        watch = { since: Date.now(), cpu: activity.cpu, io: activity.io };
        return;
      }
      if (Date.now() - watch.since < CONFIRM_PROMPT_BLOCK_MS) {
        return;
      }
      clearInterval(timer);
      halt("SIGTERM");
    }, 100);
    const finish = (code) => {
      clearInterval(timer);
      if (stopped.current) {
        halt("SIGKILL");
      }
      process.removeListener("SIGTERM", onSignal);
      process.removeListener("SIGINT", onSignal);
      process.removeListener("SIGHUP", onSignal);
      process.removeListener("exit", onExit);
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
