#!/usr/bin/env node
"use strict";

/**
 * Planning-session only. After the hello prompt, wait on SuperPlane and
 * run each user message as the next Claude Code prompt (--continue).
 * Line automations never ship this script.
 */

const fs = require("fs");
const path = require("path");
const { spawn } = require("child_process");

const HOLD_SECONDS = 45;

function nextAction(result) {
  const status = result && result.status ? String(result.status) : "";
  if (status === "ended") {
    return { type: "exit", code: 0 };
  }
  if (status === "message") {
    const text = String((result && result.text) || "").trim();
    if (text) {
      return { type: "prompt", text };
    }
    return { type: "wait" };
  }
  if (status === "created") {
    const key = String((result && result.work_order_key) || (result && result.work_order_id) || "").trim();
    const label = key ? ` (${key})` : "";
    return {
      type: "prompt",
      text: `The user created the draft task${label}. Acknowledge that in one short friendly sentence. Ask what they want to do next. Do not call propose_draft. Do not start a new draft. Then stop.`,
    };
  }
  if (status === "skipped") {
    return {
      type: "prompt",
      text: "The user skipped that draft. Acknowledge that in one short friendly sentence. Ask what they want to do next. Do not call propose_draft. Do not start a new draft. Then stop.",
    };
  }
  return { type: "wait" };
}

function readEnv(name) {
  const value = String(process.env[name] || "").trim();
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

async function requestJSON(method, urlPath) {
  const baseURL = readEnv("SUPERPLANE_BASE_URL").replace(/\/$/, "");
  const token = readEnv("SUPERPLANE_RUN_TOKEN");
  const response = await fetch(`${baseURL}${urlPath}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
    },
  });
  const text = await response.text();
  let parsed = {};
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = { message: text };
    }
  }
  if (!response.ok) {
    if (response.status === 409) {
      return { status: "ended" };
    }
    throw new Error(parsed.message || parsed.error || text || `HTTP ${response.status}`);
  }
  return parsed;
}

async function waitOnce() {
  return requestJSON("GET", `/api/v1/runner/planning-sessions/wait?hold_seconds=${HOLD_SECONDS}`);
}

function writePrompt(taskDir, text) {
  const dir = path.join(taskDir, "prompts");
  fs.mkdirSync(dir, { recursive: true });
  const file = path.join(dir, `follow-up-${Date.now()}.txt`);
  fs.writeFileSync(file, `${text}\n`);
  return file;
}

function runPromptFile(taskDir, promptFile, model) {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [path.join(taskDir, "run.js"), promptFile, model || ""], {
      stdio: "inherit",
    });
    child.on("error", reject);
    child.on("close", (code) => resolve(code == null ? 1 : code));
  });
}

async function runLoop(helpers) {
  const wait = helpers.waitOnce;
  const runPrompt = helpers.runPrompt;
  while (true) {
    const result = await wait();
    const action = nextAction(result);
    if (action.type === "exit") {
      return action.code;
    }
    if (action.type === "wait") {
      continue;
    }
    const code = await runPrompt(action.text);
    if (code !== 0) {
      process.stderr.write(`follow-up prompt failed with exit ${code}; waiting for the next message\n`);
    }
  }
}

async function main() {
  const taskDir = readEnv("SUPERPLANE_TASK_DIR");
  const model = String(process.argv[2] || "").trim();
  const code = await runLoop({
    waitOnce,
    runPrompt: (text) => runPromptFile(taskDir, writePrompt(taskDir, text), model),
  });
  process.exit(code);
}

module.exports = { nextAction, runLoop, writePrompt };

if (require.main === module) {
  main().catch((err) => {
    process.stderr.write(`${err && err.message ? err.message : err}\n`);
    process.exit(1);
  });
}
