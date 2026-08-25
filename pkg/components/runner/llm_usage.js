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
    asNumber(sidecar.total_cost_usd) > 0
  );
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
  return current;
}

function mergeResult(resultFile, taskDir) {
  if (!taskDir || !resultFile) {
    return;
  }
  const sidecar = readSidecar(taskDir);
  if (!sidecarHasUsage(sidecar) || !fs.existsSync(resultFile)) {
    return;
  }
  let parsed;
  try {
    parsed = JSON.parse(fs.readFileSync(resultFile, "utf8"));
  } catch (_err) {
    return;
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    return;
  }
  const merged = Object.assign({}, parsed, {
    usage: sidecar.usage,
  });
  if (sidecar.model) {
    merged.model = sidecar.model;
  }
  if (asNumber(sidecar.total_cost_usd) > 0) {
    merged.total_cost_usd = sidecar.total_cost_usd;
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

module.exports = { accumulate, mergeResult, addUsage };
