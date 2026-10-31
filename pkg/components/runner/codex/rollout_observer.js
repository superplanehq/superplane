"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");

// Persisted protocol: openai/codex rust-v0.163.0, protocol/src/protocol.rs.
// Exec's turn.completed usage is cumulative and cannot identify this attempt.
const ENTITLEMENT_ERROR = "To use Codex with your ChatGPT plan, upgrade to Plus: https://chatgpt.com/explore/plus.";
const MAX_DISCOVERY_ENTRIES = 4096;
const MAX_LINE_BYTES = 1024 * 1024;
const READ_BYTES = 256 * 1024;
const MAX_RESPONSES = 100000;

function emptyUsage() {
  return { input_tokens: 0, output_tokens: 0, cache_read_input_tokens: 0,
    cache_creation_input_tokens: 0, reasoning_tokens: 0, total_tokens: 0 };
}

function normalizeResponseUsage(raw) {
  if (!raw || typeof raw !== "object") return null;
  const fields = ["input_tokens", "cached_input_tokens", "output_tokens", "reasoning_output_tokens", "total_tokens"];
  if (fields.some(key => !Number.isSafeInteger(raw[key]) || raw[key] < 0)) return null;
  const writes = raw.cache_write_input_tokens ?? 0;
  if (!Number.isSafeInteger(writes) || writes < 0) return null;
  const input = raw.input_tokens - raw.cached_input_tokens - writes;
  const output = raw.output_tokens - raw.reasoning_output_tokens;
  if (input < 0 || output < 0 || !Number.isSafeInteger(raw.input_tokens + raw.output_tokens) ||
      raw.total_tokens !== raw.input_tokens + raw.output_tokens) return null;
  return { input_tokens: input, output_tokens: output, cache_read_input_tokens: raw.cached_input_tokens,
    cache_creation_input_tokens: writes, reasoning_tokens: raw.reasoning_output_tokens, total_tokens: raw.total_tokens };
}

function discoverRollouts(home) {
  const found = new Map();
  const directories = [{ directory: path.join(home, "sessions"), depth: 0 }];
  let entries = 0;
  let exhausted = false;
  while (directories.length && !exhausted) {
    const { directory, depth } = directories.pop();
    let listing;
    try {
      listing = fs.opendirSync(directory);
      let entry;
      while ((entry = listing.readSync())) {
        if (++entries > MAX_DISCOVERY_ENTRIES) { exhausted = true; break; }
        const file = path.join(directory, entry.name);
        if (entry.isDirectory() && depth < 4) directories.push({ directory: file, depth: depth + 1 });
        if (entry.isFile() && /^rollout-.+\.jsonl$/.test(entry.name)) {
          try { found.set(file, fs.statSync(file).size); } catch (_) { exhausted = true; }
        }
      }
    } catch (error) { if (error.code !== "ENOENT") exhausted = true; }
    finally { if (listing) listing.closeSync(); }
  }
  return { found, exhausted };
}

function sessionMetadata(file) {
  let descriptor;
  try {
    descriptor = fs.openSync(file, "r");
    const buffer = Buffer.alloc(64 * 1024);
    const count = fs.readSync(descriptor, buffer, 0, buffer.length, 0);
    const newline = buffer.indexOf(10, 0);
    if (newline < 0 || newline >= count) return null;
    const record = JSON.parse(buffer.subarray(0, newline).toString("utf8"));
    return record.type === "session_meta" ? record.payload : null;
  } catch (_) { return null; }
  finally { if (descriptor !== undefined) fs.closeSync(descriptor); }
}

function quotaFailure(error, rateLimits, now) {
  if (!error || error.codex_error_info !== "usage_limit_exceeded" || error.message === ENTITLEMENT_ERROR) return undefined;
  const failure = { reason: "chatgpt_quota_exhausted" };
  if (!rateLimits || typeof rateLimits !== "object") return failure;
  if (typeof rateLimits.limit_id === "string" && /^[a-zA-Z0-9_.:-]{1,128}$/.test(rateLimits.limit_id)) failure.limit_id = rateLimits.limit_id;
  const workspaceCap = rateLimits.spend_control_reached === true ||
    String(rateLimits.rate_limit_reached_type || "").startsWith("workspace_") ||
    (rateLimits.individual_limit && rateLimits.individual_limit.remaining_percent <= 0) ||
    (rateLimits.credits && rateLimits.credits.has_credits === false && rateLimits.credits.unlimited !== true);
  if (workspaceCap) return failure;
  const exhausted = [rateLimits.primary, rateLimits.secondary].filter(window =>
    window && Number.isFinite(window.used_percent) && window.used_percent >= 100);
  if (exhausted.length !== 1) return failure;
  const reset = exhausted[0].resets_at;
  if (!Number.isSafeInteger(reset) || reset <= now / 1000 || reset > 8640000000000) return failure;
  failure.resets_at = new Date(reset * 1000).toISOString();
  return failure;
}

function createRolloutObserver(options) {
  const opts = options || {};
  const home = opts.home || process.env.CODEX_HOME || path.join(os.homedir(), ".codex");
  const baseline = discoverRollouts(home);
  let threadId = opts.threadId || "";
  let selected = null;
  let offset = 0;
  let pending = Buffer.alloc(0);
  let ignoreBoundaryFragment = false;
  let incomplete = baseline.exhausted;
  let measured = false;
  let finalized = false;
  let currentTurn = "";
  let rateLimits;
  let failure;
  let terminal = false;
  let context = {};
  let usage = emptyUsage();
  let timer;
  const responses = new Map();
  const discoveredAt = Date.now();
  let lastSnapshot = "";

  function snapshot() {
    const usageState = finalized
      ? (measured ? (incomplete ? "partial" : "measured") : "unavailable")
      : (incomplete && measured ? "partial" : measured ? "measured" : "pending");
    return { attempt_id: opts.attemptId, usage: { ...usage }, usage_state: usageState,
      ...(failure ? { failure: { ...failure } } : {}), ...context };
  }

  function publish() {
    const next = snapshot();
    const encoded = JSON.stringify(next);
    if (encoded === lastSnapshot) return;
    lastSnapshot = encoded;
    if (opts.onSnapshot) opts.onSnapshot(next);
  }

  function selectFile() {
    if (!threadId || selected || Date.now() - discoveredAt > 30000) return;
    const discovery = discoverRollouts(home);
    incomplete ||= discovery.exhausted;
    const candidates = [];
    for (const [file, size] of discovery.found) {
      const name = path.basename(file);
      // Revert preserves the thread identity before an extra rollout ID.
      const marker = `-${threadId}`;
      const suffix = name.slice(name.lastIndexOf(marker) + marker.length);
      if (!name.includes(marker) || !(suffix === ".jsonl" || /^_[A-Za-z0-9-]+\.jsonl$/.test(suffix))) continue;
      const previousSize = baseline.found.get(file) || 0;
      if (size <= previousSize) continue;
      const meta = sessionMetadata(file);
      if (!meta || meta.id !== threadId || typeof meta.session_id !== "string" || !meta.session_id) continue;
      candidates.push({ file, meta, previousSize });
    }
    if (candidates.length > 1) { incomplete = true; return; }
    if (candidates.length !== 1) return;
    selected = candidates[0];
    offset = selected.previousSize;
    if (offset > 0) {
      let descriptor;
      try {
        descriptor = fs.openSync(selected.file, "r");
        const byte = Buffer.alloc(1);
        fs.readSync(descriptor, byte, 0, 1, offset - 1);
        ignoreBoundaryFragment = byte[0] !== 10;
      } catch (_) { incomplete = true; }
      finally { if (descriptor !== undefined) fs.closeSync(descriptor); }
    }
  }

  function recordResponse(payload) {
    if (!payload || payload.thread_id !== threadId || payload.session_id !== selected.meta.session_id || payload.turn_id !== currentTurn) return;
    if (typeof payload.response_id !== "string" || !payload.response_id || payload.response_id.length > 1024) { incomplete = true; return; }
    const normalized = normalizeResponseUsage(payload.usage);
    if (!normalized) { incomplete = true; return; }
    const identity = `${threadId}:${payload.response_id}`;
    const encoded = JSON.stringify(normalized);
    if (responses.has(identity)) {
      if (responses.get(identity) !== encoded) incomplete = true;
      return;
    }
    if (responses.size >= MAX_RESPONSES) { incomplete = true; return; }
    const next = { ...usage };
    for (const key of Object.keys(next)) {
      next[key] += normalized[key];
      if (!Number.isSafeInteger(next[key])) { incomplete = true; return; }
    }
    responses.set(identity, encoded);
    usage = next;
    measured = true;
  }

  function consume(record) {
    if (!record || typeof record !== "object" || !record.payload) return;
    const payload = record.payload;
    if (record.type === "turn_context") {
      if (typeof payload.model === "string") context.model = payload.model;
      if (typeof payload.effort === "string") context.thinkingLevel = payload.effort;
      return;
    }
    if (record.type === "token_usage_record") { recordResponse(payload); return; }
    if (record.type !== "event_msg") return;
    if (payload.type === "task_started" && typeof payload.turn_id === "string" && payload.turn_id) {
      currentTurn = payload.turn_id;
      terminal = false;
      rateLimits = undefined;
      failure = undefined;
      return;
    }
    if (!currentTurn) return;
    if (payload.type === "token_count" && !terminal && payload.rate_limits) rateLimits = payload.rate_limits;
    if (payload.type === "task_complete" && payload.turn_id === currentTurn) {
      terminal = true;
      if (opts.subscription) failure = quotaFailure(payload.error, rateLimits, Date.now());
    }
  }

  function poll(chunks = 4) {
    try {
      selectFile();
      if (!selected) { publish(); return false; }
      const descriptor = fs.openSync(selected.file, "r");
      let drained = false;
      try {
        if (fs.fstatSync(descriptor).size < offset) { incomplete = true; return false; }
        for (let i = 0; i < chunks; i++) {
          const buffer = Buffer.alloc(READ_BYTES);
          const count = fs.readSync(descriptor, buffer, 0, buffer.length, offset);
          if (count === 0) { drained = true; break; }
          offset += count;
          pending = Buffer.concat([pending, buffer.subarray(0, count)]);
          let newline;
          while ((newline = pending.indexOf(10)) >= 0) {
            const line = pending.subarray(0, newline);
            pending = pending.subarray(newline + 1);
            if (ignoreBoundaryFragment) { ignoreBoundaryFragment = false; continue; }
            if (line.length > MAX_LINE_BYTES) { incomplete = true; continue; }
            if (!line.length) continue;
            try { consume(JSON.parse(line.toString("utf8"))); } catch (_) { incomplete = true; }
          }
          if (pending.length > MAX_LINE_BYTES) { incomplete = true; pending = Buffer.alloc(0); ignoreBoundaryFragment = true; }
        }
      } finally { fs.closeSync(descriptor); }
      publish();
      return drained;
    } catch (_) { incomplete = true; publish(); return false; }
  }

  return {
    setThreadId(id) {
      if (threadId && threadId !== id) { incomplete = true; return; }
      if (typeof id === "string" && /^[A-Za-z0-9-]{1,128}$/.test(id)) threadId = id;
      poll();
    },
    start() { publish(); timer = setInterval(poll, 100); timer.unref(); },
    poll,
    snapshot,
    finish({ successful } = {}) {
      clearInterval(timer);
      const drained = poll(32);
      incomplete ||= !successful || Boolean(failure) || !drained || pending.length > 0 || !terminal;
      finalized = true;
      publish();
      return snapshot();
    },
  };
}

module.exports = { createRolloutObserver };
