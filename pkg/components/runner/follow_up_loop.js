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
const WAIT_RETRY_SECONDS = 1;
const FOLLOW_UP_CMD_INDEX_BASE = 1000;
const MAX_UNREACHABLE_WAITS = 8;
const WAIT_FETCH_TIMEOUT_MS = (HOLD_SECONDS + 15) * 1000;
const NIL_UUID = "00000000-0000-0000-0000-000000000000";
const UUID_PATTERN =
  /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
const IMAGE_CONTENT_TYPES = {
  "image/png": ".png",
  "image/jpeg": ".jpg",
  "image/jpg": ".jpg",
  "image/gif": ".gif",
  "image/webp": ".webp",
};
const MAX_ATTACHMENT_BYTES = 10 * 1024 * 1024;
const ATTACHMENT_DOWNLOAD_TIMEOUT_MS = 15_000;
const MAX_ATTACHMENT_REDIRECTS = 3;

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

function parseUUID(value) {
  const id = String(value || "").trim();
  if (!UUID_PATTERN.test(id) || id.toLowerCase() === NIL_UUID) {
    return "";
  }
  return id;
}

function fileIDFromHMACSignedURL(parsed) {
  const trimmed = parsed.pathname.replace(/^\/+|\/+$/g, "");
  const prefix = "api/v1/public/files/";
  if (!trimmed.startsWith(prefix)) {
    return "";
  }
  return parseUUID(trimmed.slice(prefix.length));
}

function isObjectStorageHost(host) {
  const value = String(host || "").toLowerCase();
  if (value === "storage.googleapis.com" || value.endsWith(".storage.googleapis.com")) {
    return true;
  }
  if (value === "s3.amazonaws.com" || value.endsWith(".s3.amazonaws.com")) {
    return true;
  }
  if (!value.endsWith(".amazonaws.com")) {
    return false;
  }
  return value.startsWith("s3.") || value.includes(".s3.") || value.includes(".s3-");
}

function fileIDFromObjectSignedURL(parsed) {
  if (!isObjectStorageHost(parsed.hostname)) {
    return "";
  }
  return parseUUID(path.posix.basename(parsed.pathname.replace(/\/+$/g, "")));
}

function fileIDFromSignedURL(raw) {
  let parsed;
  try {
    parsed = new URL(String(raw || "").trim());
  } catch {
    return "";
  }
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    return "";
  }
  if (parsed.searchParams.get("sp_file") !== "1") {
    return "";
  }
  return fileIDFromHMACSignedURL(parsed) || fileIDFromObjectSignedURL(parsed);
}

function allowedAppHosts(env = process.env) {
  const hosts = new Set();
  for (const name of ["SUPERPLANE_BASE_URL", "BASE_URL"]) {
    const raw = String((env && env[name]) || "").trim();
    if (!raw) {
      continue;
    }
    try {
      const parsed = new URL(raw);
      if (parsed.hostname) {
        hosts.add(parsed.hostname.toLowerCase());
      }
    } catch {
      // Ignore invalid base URLs.
    }
  }
  return hosts;
}

function isAllowedSignedDownloadURL(raw, env = process.env) {
  if (!fileIDFromSignedURL(raw)) {
    return false;
  }
  let parsed;
  try {
    parsed = new URL(String(raw || "").trim());
  } catch {
    return false;
  }
  if (isObjectStorageHost(parsed.hostname)) {
    return true;
  }
  return allowedAppHosts(env).has(String(parsed.hostname || "").toLowerCase());
}

function signedFileURLs(text) {
  const seen = new Set();
  const urls = [];
  const matches = String(text || "").match(/https?:\/\/[^\s)\]>"']+/g) || [];
  for (const match of matches) {
    if (!fileIDFromSignedURL(match) || seen.has(match)) {
      continue;
    }
    seen.add(match);
    urls.push(match);
  }
  return urls;
}

function attachmentFilename(raw, index, extension) {
  let base = "file";
  try {
    const name = path.posix.basename(new URL(raw).pathname);
    if (name && name !== "." && name !== "/") {
      base = name;
    }
  } catch {
    // Keep the fallback name.
  }
  let cleaned = "";
  for (const char of path.basename(base)) {
    if (/[a-zA-Z0-9._-]/.test(char)) {
      cleaned += char;
    }
  }
  if (!cleaned || cleaned === ".") {
    cleaned = "file";
  }
  const ext = extension && !cleaned.toLowerCase().endsWith(extension) ? extension : "";
  return `${String(index).padStart(2, "0")}-${cleaned}${ext}`;
}

function nextAttachmentIndex(dir) {
  let max = 0;
  let names = [];
  try {
    names = fs.readdirSync(dir);
  } catch {
    return 1;
  }
  for (const name of names) {
    const match = String(name).match(/^(\d+)-/);
    if (match) {
      max = Math.max(max, Number(match[1]));
    }
  }
  return max + 1;
}

function sniffImageExtension(bytes) {
  if (
    bytes.length >= 8 &&
    bytes[0] === 0x89 &&
    bytes[1] === 0x50 &&
    bytes[2] === 0x4e &&
    bytes[3] === 0x47
  ) {
    return ".png";
  }
  if (bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) {
    return ".jpg";
  }
  if (bytes.length >= 6) {
    const header = bytes.toString("ascii", 0, 6);
    if (header === "GIF87a" || header === "GIF89a") {
      return ".gif";
    }
  }
  if (
    bytes.length >= 12 &&
    bytes.toString("ascii", 0, 4) === "RIFF" &&
    bytes.toString("ascii", 8, 12) === "WEBP"
  ) {
    return ".webp";
  }
  return "";
}

function extensionForContentType(value) {
  const type = String(value || "")
    .split(";")[0]
    .trim()
    .toLowerCase();
  return IMAGE_CONTENT_TYPES[type] || "";
}

async function responseBytesLimited(response, maxBytes) {
  const declared = Number(
    response && response.headers && typeof response.headers.get === "function"
      ? response.headers.get("content-length")
      : "",
  );
  if (Number.isFinite(declared) && declared > maxBytes) {
    throw new Error(`attachment exceeds ${maxBytes} bytes`);
  }
  if (response && response.body && typeof response.body.getReader === "function") {
    const reader = response.body.getReader();
    const chunks = [];
    let total = 0;
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      const size = value && value.byteLength ? value.byteLength : 0;
      total += size;
      if (total > maxBytes) {
        try {
          await reader.cancel();
        } catch {
          // The size limit already failed the download.
        }
        throw new Error(`attachment exceeds ${maxBytes} bytes`);
      }
      chunks.push(Buffer.from(value));
    }
    return Buffer.concat(chunks, total);
  }
  if (response && typeof response.arrayBuffer === "function") {
    const bytes = Buffer.from(await response.arrayBuffer());
    if (bytes.length > maxBytes) {
      throw new Error(`attachment exceeds ${maxBytes} bytes`);
    }
    return bytes;
  }
  if (response && Buffer.isBuffer(response.body)) {
    if (response.body.length > maxBytes) {
      throw new Error(`attachment exceeds ${maxBytes} bytes`);
    }
    return response.body;
  }
  throw new Error("attachment download returned no body");
}

function redirectLocation(response, currentURL) {
  const status = Number(response && response.status);
  if (status < 300 || status >= 400) {
    return "";
  }
  const location =
    response.headers && typeof response.headers.get === "function"
      ? response.headers.get("location")
      : "";
  if (!String(location || "").trim()) {
    return "";
  }
  return new URL(location, currentURL).toString();
}

async function fetchAttachment(url, fetchImpl, env) {
  let current = url;
  for (let hop = 0; hop <= MAX_ATTACHMENT_REDIRECTS; hop += 1) {
    if (!isAllowedSignedDownloadURL(current, env)) {
      throw new Error("attachment URL host is not allowed");
    }
    const response = await fetchImpl(current, {
      redirect: "manual",
      signal: AbortSignal.timeout(ATTACHMENT_DOWNLOAD_TIMEOUT_MS),
    });
    const next = redirectLocation(response, current);
    if (next) {
      current = next;
      continue;
    }
    if (!response || !response.ok) {
      throw new Error("download failed for attachment");
    }
    const bytes = await responseBytesLimited(response, MAX_ATTACHMENT_BYTES);
    if (bytes.length <= 0) {
      throw new Error("attachment file is empty");
    }
    const headerType =
      response.headers && typeof response.headers.get === "function"
        ? response.headers.get("content-type")
        : "";
    return { bytes, headerType };
  }
  throw new Error("attachment download redirected too many times");
}

async function downloadAttachment(dir, url, index, fetchImpl, env) {
  const { bytes, headerType } = await fetchAttachment(url, fetchImpl, env);
  const extension = extensionForContentType(headerType) || sniffImageExtension(bytes);
  const filename = attachmentFilename(url, index, extension);
  const destination = path.join(dir, filename);
  fs.writeFileSync(destination, bytes);
  return { path: destination, filename };
}

function replaceURL(text, url, replacement) {
  return text.split(url).join(replacement);
}

async function materializeFollowUpAttachments(taskDir, text, fetchImpl, env = process.env) {
  const urls = signedFileURLs(text);
  if (!taskDir || urls.length === 0) {
    return text;
  }
  const attachmentsDir = path.join(taskDir, "attachments");
  fs.mkdirSync(attachmentsDir, { recursive: true });
  const doFetch = fetchImpl || fetch;
  let nextIndex = nextAttachmentIndex(attachmentsDir);
  let next = text;
  const saved = [];
  let failed = 0;
  for (const url of urls) {
    if (!isAllowedSignedDownloadURL(url, env)) {
      failed += 1;
      next = replaceURL(next, url, "");
      continue;
    }
    try {
      const file = await downloadAttachment(attachmentsDir, url, nextIndex, doFetch, env);
      nextIndex += 1;
      saved.push(file);
      next = replaceURL(next, url, file.path);
    } catch {
      failed += 1;
      next = replaceURL(next, url, "");
    }
  }
  const parts = [next];
  if (failed > 0) {
    parts.push(
      failed === 1
        ? "SuperPlane could not download 1 user image."
        : `SuperPlane could not download ${failed} user images.`,
    );
  }
  if (saved.length > 0) {
    const paths = saved.map((file) => file.path).join(", ");
    parts.push(`Call inspect_attachment on ${paths} and review the returned image.`);
  }
  return parts.join("\n\n");
}

async function prepareFollowUpText(text, helpers) {
  if (!helpers.taskDir) {
    return text;
  }
  const materialize = helpers.materializeAttachments || materializeFollowUpAttachments;
  return materialize(helpers.taskDir, text, helpers.fetch, helpers.env);
}

function persistAnalysisContinuation(taskDir, result) {
  if (!taskDir) {
    return;
  }
  const file = path.join(taskDir, "analysis_continuation.md");
  const text = String((result && result.continuation) || "").trim();
  if (!text) {
    try {
      fs.unlinkSync(file);
    } catch (_err) {
      // No previous rewind file.
    }
    return;
  }
  fs.writeFileSync(file, `${text}\n`);
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
      {
        stdio: "inherit",
        env: { ...process.env, SUPERPLANE_ANALYSIS_REWIND: "yes" },
      },
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

async function runFollowUpPrompt(text, helpers, followUpIndex) {
  const writeRecord = helpers.writeLiveLogRecord || writeLiveLogRecord;
  const now = helpers.now || Date.now;
  const index = FOLLOW_UP_CMD_INDEX_BASE + followUpIndex;
  const startedAt = now();
  emitFollowUpCommandStart(text, index, startedAt, writeRecord);
  const code = await helpers.runPrompt(text);
  emitFollowUpCommandEnd(index, code, startedAt, now(), writeRecord);
  return code;
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
        log(`stopped waiting after ${unreachableStreak} consecutive unreachable SuperPlane contacts\n`);
        return 0;
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
    persistAnalysisContinuation(helpers.taskDir, result);
    const promptText = await prepareFollowUpText(action.text, helpers);
    const code = await runFollowUpPrompt(promptText, helpers, followUpIndex);
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
    taskDir,
    runPrompt: (text) => runPromptFile(taskDir, writePrompt(taskDir, text), model, extraArgs),
  });
  process.exit(code);
}

module.exports = {
  FOLLOW_UP_CMD_INDEX_BASE,
  MAX_ATTACHMENT_BYTES,
  MAX_UNREACHABLE_WAITS,
  interpretWaitResponse,
  isAllowedSignedDownloadURL,
  materializeFollowUpAttachments,
  nextAction,
  persistAnalysisContinuation,
  runLoop,
  safeWaitRequest,
  signedFileURLs,
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
