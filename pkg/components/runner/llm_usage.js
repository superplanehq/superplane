#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");

function sidecarPath(taskDir) {
  return path.join(taskDir, "llm_usage.json");
}

function emptyUsage() {
  return {
    input_tokens: 0,
    output_tokens: 0,
    cache_read_input_tokens: 0,
    cache_creation_input_tokens: 0,
    reasoning_tokens: 0,
  };
}

function asNumber(value) {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

function addUsage(total, delta) {
  if (!delta || typeof delta !== "object") {
    return total;
  }
  total.input_tokens += asNumber(delta.input_tokens || delta.prompt_tokens);
  total.output_tokens += asNumber(delta.output_tokens || delta.completion_tokens);
  total.cache_read_input_tokens += asNumber(
    delta.cache_read_input_tokens || delta.cached_input_tokens || delta.cache_read_tokens,
  );
  total.cache_creation_input_tokens += asNumber(
    delta.cache_creation_input_tokens || delta.cache_write_tokens,
  );
  total.reasoning_tokens += asNumber(delta.reasoning_tokens);
  if (delta.total_tokens != null) total.total_tokens = asNumber(total.total_tokens) + asNumber(delta.total_tokens);
  return total;
}

function readSidecar(taskDir) {
  const file = sidecarPath(taskDir);
  if (!fs.existsSync(file)) {
    return { usage: emptyUsage() };
  }
  try {
    const parsed = JSON.parse(fs.readFileSync(file, "utf8"));
    if (!parsed || typeof parsed !== "object") {
      return { usage: emptyUsage() };
    }
    parsed.usage = addUsage(emptyUsage(), parsed.usage || {});
    return parsed;
  } catch (_err) {
    return { usage: emptyUsage() };
  }
}

function sidecarHasUsage(sidecar) {
  const usage = sidecar.usage || emptyUsage();
  return (
    usage.input_tokens > 0 ||
    usage.output_tokens > 0 ||
    usage.cache_read_input_tokens > 0 ||
    usage.cache_creation_input_tokens > 0 ||
    usage.reasoning_tokens > 0 ||
    asNumber(sidecar.total_cost_usd) > 0 ||
    Boolean(sidecar.usage_state)
  );
}

function readPromptSeries(taskDir) {
  const file = path.join(taskDir, "turn_telemetry_series.json");
  if (!taskDir || !fs.existsSync(file)) {
    return [];
  }
  try {
    const parsed = JSON.parse(fs.readFileSync(file, "utf8"));
    if (!parsed || typeof parsed !== "object" || !Array.isArray(parsed.series)) {
      return [];
    }
    return parsed.series.filter((item) => item && typeof item === "object");
  } catch (_err) {
    return [];
  }
}

function telemetryForResult(series) {
  if (series.length === 0) {
    return null;
  }
  if (series.length === 1) {
    return series[0].telemetry || series[0];
  }
  return { prompts: series };
}

function accumulate(taskDir, payload) {
  if (!taskDir || !payload || typeof payload !== "object") {
    return readSidecar(taskDir || "");
  }
  const current = readSidecar(taskDir);
  current.usage = addUsage(current.usage || emptyUsage(), payload.usage || payload);
  if (payload.model) {
    current.model = String(payload.model);
  }
  if (payload.total_cost_usd != null) {
    current.total_cost_usd = asNumber(current.total_cost_usd) + asNumber(payload.total_cost_usd);
  }
  fs.writeFileSync(sidecarPath(taskDir), `${JSON.stringify(current)}\n`);
  mergeResult(process.env.SUPERPLANE_RESULT_FILE, taskDir);
  return current;
}

function snapshotAttempt(taskDir, payload) {
  if (!taskDir || !payload || !payload.attempt_id) return;
  const current = readSidecar(taskDir);
  if (!current.attempts) {
    current.legacy_usage = current.usage;
    current.attempts = {};
  }
  current.attempts[payload.attempt_id] = {
    usage: payload.usage, usage_state: payload.usage_state,
    ...(payload.failure ? { failure: payload.failure } : {}),
  };
  current.latest_attempt_id = payload.attempt_id;
  current.usage = Object.values(current.attempts).reduce((total, attempt) => addUsage(total, attempt.usage), addUsage(emptyUsage(), current.legacy_usage));
  const states = Object.values(current.attempts).map(attempt => attempt.usage_state);
  current.usage_state = states.includes("partial") || (states.includes("unavailable") && states.includes("measured"))
    ? "partial" : states.includes("pending") ? "pending" : states.includes("unavailable") ? "unavailable" : "measured";
  if (payload.model) current.model = String(payload.model);
  const file = sidecarPath(taskDir);
  fs.writeFileSync(`${file}.next`, `${JSON.stringify(current)}\n`);
  fs.renameSync(`${file}.next`, file);
  mergeResult(process.env.SUPERPLANE_RESULT_FILE, taskDir);
  return current;
}

function mergeResult(resultFile, taskDir) {
  if (!taskDir || !resultFile) {
    return;
  }
  const sidecar = readSidecar(taskDir);
  const series = readPromptSeries(taskDir);
  const telemetry = telemetryForResult(series);
  if (!sidecarHasUsage(sidecar) && !telemetry) {
    return;
  }
  let parsed = {};
  if (fs.existsSync(resultFile)) {
    try {
      parsed = JSON.parse(fs.readFileSync(resultFile, "utf8"));
    } catch (_err) {
      parsed = {};
    }
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return;
    }
  }
  const merged = Object.assign({}, parsed);
  if (sidecarHasUsage(sidecar)) {
    merged.usage = sidecar.usage;
    if (sidecar.usage_state) merged.usage_state = sidecar.usage_state;
    const latest = sidecar.attempts && sidecar.attempts[sidecar.latest_attempt_id];
    if (latest) {
      merged.attempt_id = sidecar.latest_attempt_id;
      if (latest.failure) merged.failure = latest.failure;
      else delete merged.failure;
    }
    if (sidecar.model) {
      merged.model = sidecar.model;
    }
    if (asNumber(sidecar.total_cost_usd) > 0) {
      merged.total_cost_usd = sidecar.total_cost_usd;
    }
  }
  if (telemetry) {
    merged.telemetry = telemetry;
  }
  fs.writeFileSync(resultFile, `${JSON.stringify(merged)}\n`);
}

function main() {
  const command = process.argv[2];
  const taskDir = process.env.SUPERPLANE_TASK_DIR;
  if (command === "accumulate") {
    accumulate(taskDir, JSON.parse(process.argv[3] || "{}"));
    return;
  }
  if (command === "merge") {
    mergeResult(process.env.SUPERPLANE_RESULT_FILE, taskDir);
  }
}

if (require.main === module) {
  main();
}

module.exports = { accumulate, mergeResult, addUsage, snapshotAttempt };
