"use strict";

const CHECK_LEVELS = ["positive", "caution", "critical"];
const MAX_CHECKS = 20;
const MAX_CHECK_KEY_BYTES = 255;
const MAX_CHECK_NAME_BYTES = 255;
const MAX_CHECK_REASON_LENGTH = 1000;

function exceedsCodePoints(value, max) {
  let count = 0;
  for (let index = 0; index < value.length; ) {
    count += 1;
    if (count > max) return true;
    const code = value.codePointAt(index);
    index += code > 0xffff ? 2 : 1;
  }
  return false;
}

function requiredText(value, label, maxBytes) {
  const text = String(value == null ? "" : value).trim();
  if (!text) throw new Error(`${label} is required`);
  if (Buffer.byteLength(text) > maxBytes) throw new Error(`${label} must be at most ${maxBytes} bytes`);
  return text;
}

function number(value, label) {
  if (value == null || value === "") throw new Error(`${label} is required`);
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) throw new Error(`${label} must be a number`);
  return parsed;
}

function normalizeCheck(raw, index) {
  const label = `checks[${index}]`;
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) throw new Error(`${label} must be an object`);
  const key = requiredText(raw.key, `${label}.key`, MAX_CHECK_KEY_BYTES);
  const name = requiredText(raw.name, `${label}.name`, MAX_CHECK_NAME_BYTES);
  const score = number(raw.score, `${label}.score`);
  const maxScore = number(raw.max_score, `${label}.max_score`);
  if (maxScore <= 0) throw new Error(`${label}.max_score must be greater than 0`);
  if (score < 0 || score > maxScore) throw new Error(`${label}.score must be from 0 to max_score`);
  const level = String(raw.level || "").trim().toLowerCase();
  if (!CHECK_LEVELS.includes(level)) throw new Error(`${label}.level must be positive, caution, or critical`);
  const reason = String(raw.reason || "").trim();
  if (!reason) throw new Error(`${label}.reason is required`);
  if (exceedsCodePoints(reason, MAX_CHECK_REASON_LENGTH)) {
    throw new Error(`${label}.reason must be at most ${MAX_CHECK_REASON_LENGTH} characters`);
  }
  return { key, name, score, max_score: maxScore, level, reason };
}

async function listChecks(input, client) {
  const task = String((input && input.task) || "").trim();
  if (!task) throw new Error("task is required");
  return client.request("GET", `/api/v1/runner/checks?${new URLSearchParams({ task }).toString()}`);
}

async function addChecks(input, client) {
  const raw = input && input.checks;
  if (!Array.isArray(raw) || raw.length === 0) throw new Error("checks is required");
  if (raw.length > MAX_CHECKS) throw new Error(`checks must have at most ${MAX_CHECKS} items`);
  const checks = raw.map(normalizeCheck);
  const seen = new Set();
  checks.forEach((check, index) => {
    if (seen.has(check.key)) throw new Error(`checks[${index}].key repeats an earlier key`);
    seen.add(check.key);
  });
  return client.request("POST", "/api/v1/runner/task/checks", { checks });
}

async function removeChecks(input, client) {
  const raw = input && input.keys;
  const values = Array.isArray(raw) ? raw : raw == null ? [] : [raw];
  const keys = [...new Set(values.map((value) => String(value == null ? "" : value).trim()).filter(Boolean))];
  if (keys.length === 0) throw new Error("keys is required");
  if (keys.length > MAX_CHECKS) throw new Error(`keys must have at most ${MAX_CHECKS} items`);
  return client.request("DELETE", "/api/v1/runner/task/checks", { keys });
}

const checkSchema = {
  type: "object",
  properties: {
    key: { type: "string", description: "A stable id, for example api-latency. A check with the same key is replaced." },
    name: { type: "string", description: "The label that people see, for example API latency." },
    score: { type: "number", description: "For example 2." },
    max_score: { type: "number", description: "For example 5." },
    level: { type: "string", enum: CHECK_LEVELS, description: "Sets the color of the check." },
    reason: { type: "string", description: "One sentence that explains the score." },
  },
  required: ["key", "name", "score", "max_score", "level", "reason"],
};

const LIST_CHECKS = {
  definition: {
    name: "list_checks",
    description:
      "List all checks on a task: the plan scores, the merge confidence checks, and the checks from add_checks. Each check has key, name, score, max_score, level, and reason.",
    inputSchema: {
      type: "object",
      properties: {
        task: { type: "string", description: "The id or key of the task, for example from list_tasks." },
      },
      required: ["task"],
    },
  },
  handler: listChecks,
};

const ADD_CHECKS = {
  definition: {
    name: "add_checks",
    description:
      "Add checks to this run's task. A check shows a score and a reason on the task. On a planning run, this edits the session draft. A check with the same key replaces your earlier check. You cannot replace a plan score or a merge confidence check. Returns checks, the saved checks.",
    inputSchema: {
      type: "object",
      properties: {
        checks: { type: "array", minItems: 1, maxItems: MAX_CHECKS, items: checkSchema },
      },
      required: ["checks"],
    },
  },
  handler: addChecks,
};

const REMOVE_CHECKS = {
  definition: {
    name: "remove_checks",
    description:
      "Remove checks that add_checks added to this run's task. Returns removed, the keys that SuperPlane removed.",
    inputSchema: {
      type: "object",
      properties: {
        keys: { type: "array", minItems: 1, maxItems: MAX_CHECKS, items: { type: "string" } },
      },
      required: ["keys"],
    },
  },
  handler: removeChecks,
};

module.exports = {
  tools: [LIST_CHECKS, ADD_CHECKS, REMOVE_CHECKS],
};
