#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");

function sidecarPath(taskDir) {
  return path.join(taskDir, "turn_telemetry.json");
}

function seriesPath(taskDir) {
  return path.join(taskDir, "turn_telemetry_series.json");
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

function normalizeUsage(raw) {
  const src = raw && typeof raw === "object" && !Array.isArray(raw) ? raw : {};
  const nested = src.usage && typeof src.usage === "object" ? src.usage : src;
  const usage = {
    input_tokens: asNumber(nested.input_tokens || nested.prompt_tokens),
    output_tokens: asNumber(nested.output_tokens || nested.completion_tokens),
    cache_read_input_tokens: asNumber(
      nested.cache_read_input_tokens || nested.cached_input_tokens || nested.cache_read_tokens,
    ),
    cache_creation_input_tokens: asNumber(nested.cache_creation_input_tokens || nested.cache_write_tokens),
    reasoning_tokens: asNumber(nested.reasoning_tokens),
  };
  const cost = src.total_cost_usd != null ? src.total_cost_usd : nested.total_cost_usd;
  if (cost != null && Number.isFinite(Number(cost))) {
    usage.total_cost_usd = Number(cost);
  }
  return usage;
}

function tokenTotal(usage) {
  return (
    asNumber(usage.input_tokens) +
    asNumber(usage.output_tokens) +
    asNumber(usage.cache_read_input_tokens) +
    asNumber(usage.cache_creation_input_tokens) +
    asNumber(usage.reasoning_tokens)
  );
}

function addUsage(total, delta) {
  total.input_tokens += asNumber(delta.input_tokens);
  total.output_tokens += asNumber(delta.output_tokens);
  total.cache_read_input_tokens += asNumber(delta.cache_read_input_tokens);
  total.cache_creation_input_tokens += asNumber(delta.cache_creation_input_tokens);
  total.reasoning_tokens += asNumber(delta.reasoning_tokens);
  if (delta.total_cost_usd != null || total.total_cost_usd != null) {
    total.total_cost_usd = asNumber(total.total_cost_usd) + asNumber(delta.total_cost_usd);
  }
  return total;
}

function resolveCumulative(previous, incoming) {
  const next = normalizeUsage(incoming);
  if (!previous) {
    return next;
  }
  if (tokenTotal(next) >= tokenTotal(previous) && tokenTotal(next) > 0) {
    if (next.total_cost_usd == null && previous.total_cost_usd != null) {
      next.total_cost_usd = previous.total_cost_usd;
    }
    return next;
  }
  return addUsage({ ...previous }, next);
}

function emptyState() {
  return { currentTurn: 0, turns: [], lastMessageId: "" };
}

function usageEquals(left, right) {
  const a = normalizeUsage(left);
  const b = normalizeUsage(right);
  return (
    a.input_tokens === b.input_tokens &&
    a.output_tokens === b.output_tokens &&
    a.cache_read_input_tokens === b.cache_read_input_tokens &&
    a.cache_creation_input_tokens === b.cache_creation_input_tokens &&
    a.reasoning_tokens === b.reasoning_tokens
  );
}

function loadState(_taskDir) {
  return emptyState();
}

function readPromptSeries(taskDir) {
  if (!taskDir || !fs.existsSync(seriesPath(taskDir))) {
    return [];
  }
  try {
    const parsed = JSON.parse(fs.readFileSync(seriesPath(taskDir), "utf8"));
    if (!parsed || typeof parsed !== "object" || !Array.isArray(parsed.series)) {
      return [];
    }
    return parsed.series.filter((item) => item && typeof item === "object");
  } catch (_err) {
    return [];
  }
}

function appendPromptSeries(taskDir, name, telemetry) {
  if (!taskDir || !telemetry) {
    return;
  }
  const series = readPromptSeries(taskDir);
  series.push({ name: name || "", telemetry });
  fs.writeFileSync(seriesPath(taskDir), `${JSON.stringify({ series })}\n`);
}

function saveState(taskDir, state) {
  if (!taskDir) {
    return;
  }
  fs.writeFileSync(sidecarPath(taskDir), `${JSON.stringify(state)}\n`);
}

function defaultWrite(record) {
  process.stdout.write(`${JSON.stringify(record)}\n`);
}

function countTools(turns) {
  const counts = {};
  for (const turn of turns) {
    for (const tool of turn.tools || []) {
      const kind = String(tool.kind || "tool");
      counts[kind] = (counts[kind] || 0) + 1;
    }
  }
  return counts;
}

function buildTelemetry(state) {
  const turns = (state.turns || []).map((turn) => {
    const row = {
      turn: turn.turn,
      usage: { ...emptyUsage(), ...(turn.usage || {}) },
      tools: Array.isArray(turn.tools) ? turn.tools : [],
    };
    if (turn.message) {
      row.message = turn.message;
    }
    return row;
  });
  return {
    num_turns: turns.length,
    usage: turns.reduce((total, turn) => addUsage(total, turn.usage), emptyUsage()),
    tool_counts: countTools(turns),
    turns,
  };
}

function currentSnapshot(state) {
  if (state.currentTurn < 1) {
    return null;
  }
  return state.turns.find((turn) => turn.turn === state.currentTurn) || state.turns[state.turns.length - 1] || null;
}

function findTool(snapshot, id) {
  if (!snapshot || !Array.isArray(snapshot.tools)) {
    return null;
  }
  if (id != null && String(id).trim()) {
    const key = String(id).trim();
    const match = snapshot.tools.find((tool) => tool.id === key);
    if (match) {
      return match;
    }
  }
  return snapshot.tools[snapshot.tools.length - 1] || null;
}

function createTurnTelemetry(options) {
  const opts = options || {};
  const taskDir = opts.taskDir !== undefined ? opts.taskDir : process.env.SUPERPLANE_TASK_DIR || "";
  const write = typeof opts.write === "function" ? opts.write : defaultWrite;
  const state = loadState(taskDir);

  function persist() {
    saveState(taskDir, state);
  }

  function emitTurn(snapshot) {
    const record = {
      type: "turn",
      turn: snapshot.turn,
      usage: snapshot.usage,
    };
    if (snapshot.message) {
      record.message = snapshot.message;
    }
    write(record);
  }

  return {
    currentTurn() {
      return state.currentTurn;
    },
    beginTurn(usage, extra) {
      const opts = extra && typeof extra === "object" && !Array.isArray(extra) ? extra : {};
      const messageId = opts.messageId != null ? String(opts.messageId) : "";
      const incoming = normalizeUsage(usage);
      const previous = currentSnapshot(state);
      if (messageId && state.lastMessageId === messageId) {
        return state.currentTurn;
      }
      if (
        previous &&
        usageEquals(previous.usage, incoming) &&
        !opts.hasTools &&
        tokenTotal(incoming) > 0
      ) {
        return state.currentTurn;
      }
      state.currentTurn += 1;
      state.lastMessageId = messageId;
      const snapshot = {
        turn: state.currentTurn,
        usage: incoming,
        tools: [],
      };
      if (opts.message != null && String(opts.message).trim()) {
        snapshot.message = String(opts.message).trim();
      }
      state.turns.push(snapshot);
      persist();
      emitTurn(snapshot);
      return state.currentTurn;
    },
    updateCurrentUsage(usage, totalCostUsd) {
      const snapshot = currentSnapshot(state);
      if (!snapshot) {
        const incoming = normalizeUsage(usage);
        if (totalCostUsd != null && Number.isFinite(Number(totalCostUsd))) {
          incoming.total_cost_usd = Number(totalCostUsd);
        }
        return this.beginTurn(incoming);
      }
      const incoming = normalizeUsage(usage);
      if (totalCostUsd != null && Number.isFinite(Number(totalCostUsd))) {
        incoming.total_cost_usd = Number(totalCostUsd);
      }
      snapshot.usage = resolveCumulative(snapshot.usage, incoming);
      persist();
      return snapshot.turn;
    },
    stampToolStart(record) {
      if (state.currentTurn < 1) {
        this.beginTurn({});
      }
      const snapshot = currentSnapshot(state);
      const rec = record && typeof record === "object" ? record : {};
      const tool = {
        id: rec.id != null && String(rec.id).trim() ? String(rec.id).trim() : undefined,
        kind: String(rec.kind || "tool"),
        text: rec.text != null ? String(rec.text) : String(rec.kind || "tool"),
        status: "running",
      };
      if (snapshot) {
        snapshot.tools.push(tool);
        persist();
      }
      return Object.assign({}, rec, { turn: state.currentTurn });
    },
    stampToolEnd(record) {
      const rec = record && typeof record === "object" ? record : {};
      const turn = rec.turn != null && Number(rec.turn) > 0 ? Number(rec.turn) : state.currentTurn || 1;
      const snapshot = state.turns.find((item) => item.turn === turn) || currentSnapshot(state);
      const tool = findTool(snapshot, rec.id);
      if (tool) {
        if (rec.status) {
          tool.status = rec.status;
        }
        if (rec.duration_ms != null) {
          tool.duration_ms = asNumber(rec.duration_ms);
        }
        persist();
      }
      return Object.assign({}, rec, { turn });
    },
    snapshot() {
      return buildTelemetry(state);
    },
    attachToResult(result, extra) {
      if (!result || typeof result !== "object" || Array.isArray(result)) {
        return result;
      }
      const opts = extra && typeof extra === "object" && !Array.isArray(extra) ? extra : {};
      const telemetry = buildTelemetry(state);
      if (result.usage) {
        telemetry.usage = normalizeUsage(result.usage);
      }
      if (result.num_turns != null && Number(result.num_turns) > telemetry.num_turns) {
        telemetry.num_turns = Number(result.num_turns);
      }
      result.telemetry = telemetry;
      appendPromptSeries(taskDir, opts.name || "", telemetry);
      return result;
    },
  };
}

function resolveTurnTelemetryModule() {
  const taskDir = process.env.SUPERPLANE_TASK_DIR || "";
  const candidates = [];
  if (taskDir) {
    candidates.push(path.join(taskDir, "turn_telemetry.js"));
  }
  candidates.push(path.join(__dirname, "turn_telemetry.js"));
  candidates.push(path.join(__dirname, "..", "turn_telemetry.js"));

  for (const file of candidates) {
    if (fs.existsSync(file)) {
      return require(file);
    }
  }
  return module.exports;
}

function loadTurnTelemetry(options) {
  return resolveTurnTelemetryModule().createTurnTelemetry(options);
}

module.exports = {
  createTurnTelemetry,
  loadTurnTelemetry,
  emptyUsage,
  normalizeUsage,
  resolveCumulative,
  usageEquals,
  readPromptSeries,
  appendPromptSeries,
};
