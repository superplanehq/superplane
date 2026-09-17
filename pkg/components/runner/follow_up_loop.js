#!/usr/bin/env node
"use strict";

/**
 * Planning-session only. After the hello prompt, wait on SuperPlane and
 * run each user message as the next Claude Code prompt (--continue).
 * Line automations never ship this script.
 */

const fs = require("fs");
const path = require("path");
const { spawn, spawnSync } = require("child_process");

const HOLD_SECONDS = 45;
const WAIT_RETRY_SECONDS = 1;
const FOLLOW_UP_CMD_INDEX_BASE = 1000;
const MAX_UNREACHABLE_WAITS = 8;
const WAIT_FETCH_TIMEOUT_MS = (HOLD_SECONDS + 15) * 1000;

function nextAction(result) {
  const status = result && result.status ? String(result.status) : "";
  if (status === "ended") {
    return { type: "exit", code: 0 };
  }
  if (status === "message") {
    const text = String((result && result.text) || "").trim();
    if (text) {
      return { type: "prompt", text, files: Array.isArray(result.files) ? result.files : [] };
    }
    return { type: "wait" };
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

async function safeWaitRequest(doFetch) {
  let response;
  let text;
  try {
    response = await doFetch();
    text = await response.text();
  } catch {
    return { status: "pending", unreachable: true };
  }
  let parsed = {};
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = { message: text };
    }
  }
  return interpretWaitResponse(response.status, parsed, text);
}

async function requestJSON(method, urlPath) {
  const baseURL = readEnv("SUPERPLANE_BASE_URL").replace(/\/$/, "");
  const token = readEnv("SUPERPLANE_RUN_TOKEN");
  return safeWaitRequest(() =>
    fetch(`${baseURL}${urlPath}`, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: "application/json",
        "ngrok-skip-browser-warning": "1",
      },
      signal: AbortSignal.timeout(WAIT_FETCH_TIMEOUT_MS),
    }),
  );
}

function waitResponseMessage(parsed, text) {
  return String((parsed && (parsed.message || parsed.error)) || text || "");
}

function isPlanningSessionGone(status, parsed, text) {
  if (status !== 404) {
    return false;
  }
  return /planning session not found/i.test(waitResponseMessage(parsed, text));
}

function isTransientWaitFailure(status, parsed) {
  if (status === 401 || status === 400) {
    return false;
  }
  if (status >= 400) {
    return true;
  }
  return Boolean(parsed && (parsed.retryable === true || parsed.cloudflare_error === true));
}

function interpretWaitResponse(status, parsed, text) {
  if (status === 409 || isPlanningSessionGone(status, parsed, text)) {
    return { status: "ended" };
  }
  if (status >= 200 && status < 300) {
    return parsed && typeof parsed === "object" ? parsed : {};
  }
  if (isTransientWaitFailure(status, parsed)) {
    return { status: "pending" };
  }
  throw new Error(waitResponseMessage(parsed, text) || `HTTP ${status}`);
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

function runPromptFile(taskDir, promptFile, model, extraArgs = []) {
  return new Promise((resolve, reject) => {
    const child = spawn(
      process.execPath,
      [path.join(taskDir, "run.js"), promptFile, model || "", ...extraArgs],
      { stdio: "inherit" },
    );
    child.on("error", reject);
    child.on("close", (code) => resolve(code == null ? 1 : code));
  });
}

function defaultSleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

function writeLiveLogRecord(rec) {
  process.stdout.write(`${JSON.stringify(rec)}\n`);
}

function emitFollowUpCommandStart(text, index, startedAt, writeRecord) {
  writeRecord({
    type: "cmd_start",
    index,
    text,
    kind: "prompt",
    preview: text,
    started_at: startedAt,
  });
}

function emitFollowUpCommandEnd(index, code, startedAt, now, writeRecord) {
  writeRecord({
    type: "cmd_end",
    index,
    status: code === 0 ? "passed" : "failed",
    duration_ms: Math.max(0, now - startedAt),
  });
}

async function runFollowUpPrompt(action, helpers, followUpIndex) {
  const text = action.text;
  const writeRecord = helpers.writeLiveLogRecord || writeLiveLogRecord;
  const now = helpers.now || Date.now;
  const index = FOLLOW_UP_CMD_INDEX_BASE + followUpIndex;
  const startedAt = now();
  await maybePrepareAttachments(action.files || [], helpers);
  emitFollowUpCommandStart(text, index, startedAt, writeRecord);
  const code = await helpers.runPrompt(text);
  emitFollowUpCommandEnd(index, code, startedAt, now(), writeRecord);
  return code;
}

function maybePrepareAttachments(files, helpers) {
  if (typeof helpers.prepareAttachments === "function") {
    return helpers.prepareAttachments(files);
  }
  if (!files || files.length === 0) {
    return undefined;
  }
  prepareIncomingAttachments(readEnv("SUPERPLANE_TASK_DIR"), files);
  return undefined;
}

function attachmentKind(file) {
  const type = String((file && file.content_type) || "").split(";")[0].trim().toLowerCase();
  if (type.startsWith("video/")) {
    return "video";
  }
  const name = String((file && (file.filename || file.dest)) || "").toLowerCase();
  return /\.(mp4|webm|mov|ogv|ogg|m4v|mkv)$/.test(name) ? "video" : "file";
}

function sanitizeDestName(name, index) {
  const cleaned = String(name || "file")
    .split(/[/\\]/)
    .pop()
    .replace(/[^a-zA-Z0-9._-]/g, "");
  const base = cleaned && cleaned !== "." ? cleaned : "file";
  return `${String(index).padStart(2, "0")}-${base}`;
}

function loadManifest(taskDir) {
  const manifestPath = path.join(taskDir, "attachments", "manifest.json");
  if (!fs.existsSync(manifestPath)) {
    return {
      version: 1,
      policy: {},
      files: [],
    };
  }
  return JSON.parse(fs.readFileSync(manifestPath, "utf8"));
}

function saveManifest(taskDir, manifest) {
  const dir = path.join(taskDir, "attachments");
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(dir, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);
}

function mergeManifestFiles(taskDir, incoming) {
  const manifest = loadManifest(taskDir);
  const files = Array.isArray(manifest.files) ? manifest.files : [];
  const seen = new Set(files.map((file) => file.id || file.url).filter(Boolean));
  let added = false;
  for (const file of incoming) {
    const key = file.id || file.url;
    if (!key || seen.has(key)) {
      continue;
    }
    seen.add(key);
    files.push({
      id: file.id || "",
      filename: file.filename || "file",
      content_type: file.content_type || "",
      size_bytes: file.size_bytes || 0,
      checksum: file.checksum || "",
      url: file.url || "",
      dest: sanitizeDestName(file.filename || file.url, files.length + 1),
      kind: attachmentKind(file),
      status: "pending",
    });
    added = true;
  }
  manifest.files = files;
  saveManifest(taskDir, manifest);
  return { manifest, added };
}

function runTaskScript(taskDir, name) {
  const script = path.join(taskDir, name);
  if (!fs.existsSync(script)) {
    throw new Error(
      `${name} is missing. Rebuild the runner image with ffmpeg, ffprobe, whisper-cli, and the Whisper model.`,
    );
  }
  const result = spawnSync("bash", [script], { stdio: "inherit", env: process.env });
  if (result.status !== 0) {
    throw new Error(`${name} failed with exit ${result.status == null ? 1 : result.status}`);
  }
}

function prepareIncomingAttachments(taskDir, incoming) {
  if (!incoming || incoming.length === 0) {
    return;
  }
  const { manifest, added } = mergeManifestFiles(taskDir, incoming);
  if (!added) {
    return;
  }
  runTaskScript(taskDir, "fetch_task_attachments.sh");
  if ((manifest.files || []).some((file) => file.kind === "video")) {
    runTaskScript(taskDir, "process_video_attachments.sh");
  }
}

function isUnreachableWait(result) {
  return Boolean(result && result.unreachable);
}

async function runLoop(helpers) {
  const wait = helpers.waitOnce;
  const sleep = helpers.sleep || defaultSleep;
  const log = helpers.log || ((msg) => process.stderr.write(msg));
  const maxUnreachable = helpers.maxUnreachableWaits || MAX_UNREACHABLE_WAITS;
  let followUpIndex = 0;
  let unreachableStreak = 0;
  while (true) {
    const result = await wait();
    if (isUnreachableWait(result)) {
      unreachableStreak += 1;
      if (unreachableStreak >= maxUnreachable) {
        log(`SuperPlane wait failed ${unreachableStreak} times; exiting\n`);
        return 1;
      }
    } else {
      unreachableStreak = 0;
    }
    const action = nextAction(result);
    if (action.type === "exit") {
      return action.code;
    }
    if (action.type === "wait") {
      await sleep(WAIT_RETRY_SECONDS * 1000);
      continue;
    }
    const code = await runFollowUpPrompt(action, helpers, followUpIndex);
    followUpIndex += 1;
    if (code !== 0) {
      log(`follow-up prompt failed with exit ${code}; waiting for the next message\n`);
    }
  }
}

async function main() {
  const taskDir = readEnv("SUPERPLANE_TASK_DIR");
  const model = String(process.argv[2] || "").trim();
  // Forward extra argv so follow-up prompts match the original prompt step.
  const extraArgs = process.argv.slice(3);
  const code = await runLoop({
    waitOnce,
    runPrompt: (text) => runPromptFile(taskDir, writePrompt(taskDir, text), model, extraArgs),
  });
  process.exit(code);
}

module.exports = {
  FOLLOW_UP_CMD_INDEX_BASE,
  MAX_UNREACHABLE_WAITS,
  interpretWaitResponse,
  nextAction,
  prepareIncomingAttachments,
  runLoop,
  safeWaitRequest,
  writeLiveLogRecord,
  writePrompt,
  runPromptFile,
};

if (require.main === module) {
  main().catch((err) => {
    process.stderr.write(`${err && err.message ? err.message : err}\n`);
    process.exit(1);
  });
}
