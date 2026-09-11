#!/usr/bin/env node
"use strict";

/**
 * Run OpenCode against OpenRouter and format JSONL into live logs.
 *
 *   node run.js <prompt-file> [model]
 */

const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { spawn, spawnSync } = require("child_process");

const TOOL_RESULT_MAX_CHARS = 800;
const TOOL_RESULT_MAX_LINES = 24;
const DEFAULT_WAIT_CAP_MS = 3_600_000;
const SESSION_FILE = "opencode_session";
const MAX_ATTEMPTS = 4;
const RETRY_WAIT_MS = [30_000, 45_000, 60_000];

const PLANNING_SYSTEM_PROMPT =
  "This is a SuperPlane planning session. Call the propose_draft tool only when the user asked for a task in this turn. " +
  "Call propose_draft with a title and a description. The description must include the user's request and constraints. " +
  "After you show a draft, tell the user it is on the right and ask them to review it. " +
  "Call the survey tool to ask questions. SuperPlane waits after you stop. Do not create work orders yourself. " +
  "When the user creates or skips a draft, acknowledge that in one short sentence and ask what they want to do next. " +
  "Do not call propose_draft unless they ask for a task. When the user starts a refine, read the current task, tell " +
  "them you are ready, and ask what they want to change. Do not call propose_draft until they say what to change. " +
  "Write to the user in plain text. Only explore the repository (read files, search, run read-only commands); do not " +
  "edit or write any files.";

function envFlag(env, name) {
  return Boolean(String((env && env[name]) || "").trim());
}

function planningEnabled(env = process.env) {
  return envFlag(env, "SUPERPLANE_PLANNING_SESSION_ID");
}

function catalogModelId(model) {
  return String(model || "")
    .trim()
    .replace(/^openrouter\//, "");
}

function openRouterModelId(model) {
  const trimmed = String(model || "").trim();
  if (!trimmed) {
    return "";
  }
  if (trimmed.startsWith("openrouter/")) {
    return trimmed;
  }
  return `openrouter/${trimmed}`;
}

function classifyOpenRouterError(text) {
  const raw = String(text || "");
  const lower = raw.toLowerCase();
  if (
    /invalid api key|incorrect api key|unauthorized|authentication|insufficient credit|insufficient credits|unknown model|model not found|context (length|window|overflow)|maximum context|prompt is too long/.test(
      lower,
    )
  ) {
    return "hard";
  }
  if (
    /new-account-rpm|rate limit exceeded|rate limit reached|please retry shortly|\b429\b/.test(lower) ||
    /too many requests/.test(lower)
  ) {
    return "rate_limit";
  }
  if (
    /\b(502|503|504|529)\b/.test(lower) ||
    /overloaded|unavailable|no endpoints|no available provider|temporarily|capacity|timeout|econnreset|fetch failed|provider returned error|server error|bad gateway|service unavailable/.test(
      lower,
    )
  ) {
    return "retryable";
  }
  return "other";
}

function parseRetryAfterMs(text) {
  const match = String(text || "").match(/retry-after:\s*(\d+)/i);
  if (!match) {
    return null;
  }
  const seconds = Number(match[1]);
  if (!Number.isFinite(seconds) || seconds < 0) {
    return null;
  }
  return seconds * 1000;
}

function retryWaitMs(errorText, failedAttempt) {
  const fromHeader = parseRetryAfterMs(errorText);
  if (fromHeader != null) {
    return fromHeader;
  }
  const index = Math.max(0, Number(failedAttempt) - 1);
  if (index >= RETRY_WAIT_MS.length) {
    return RETRY_WAIT_MS[RETRY_WAIT_MS.length - 1];
  }
  return RETRY_WAIT_MS[index];
}

function isRetryableKind(kind) {
  return kind === "rate_limit" || kind === "retryable";
}

function uniqueCatalogModels(ids) {
  const seen = new Set();
  const out = [];
  const list = Array.isArray(ids) ? ids : [];
  for (const raw of list) {
    const id = catalogModelId(raw);
    if (!id || seen.has(id)) {
      continue;
    }
    seen.add(id);
    out.push(id);
  }
  return out;
}

function clearSession(taskDir) {
  const file = path.join(taskDir, SESSION_FILE);
  if (fs.existsSync(file)) {
    fs.unlinkSync(file);
  }
}

function restoreSession(taskDir, sessionID) {
  if (sessionID) {
    writeSessionID(taskDir, sessionID);
    return;
  }
  clearSession(taskDir);
}

function waitSecondsLabel(ms) {
  const seconds = Math.max(1, Math.round(Number(ms) / 1000));
  return `${seconds} seconds`;
}

function errorKindLabel(kind) {
  if (kind === "rate_limit") {
    return "Rate limit";
  }
  if (kind === "retryable") {
    return "Temporary error";
  }
  return "Error";
}

function retryWaitLine(kind, model, waitMs, nextAttempt) {
  return `${errorKindLabel(kind)} on ${catalogModelId(model)}. Waiting ${waitSecondsLabel(waitMs)}, then retrying (attempt ${nextAttempt} of ${MAX_ATTEMPTS}).`;
}

function stoppedAfterAttemptsLine(kind, model) {
  return `${errorKindLabel(kind)} on ${catalogModelId(model)}. Stopped after ${MAX_ATTEMPTS} attempts.`;
}

function waitExceededTimeoutLine(kind) {
  return `${errorKindLabel(kind)} wait exceeded the execution timeout`;
}

function readSessionID(taskDir) {
  const file = path.join(taskDir, SESSION_FILE);
  if (!fs.existsSync(file)) {
    return "";
  }
  return fs.readFileSync(file, "utf8").trim();
}

function writeSessionID(taskDir, sessionID) {
  const id = String(sessionID || "").trim();
  if (!id) {
    return;
  }
  fs.writeFileSync(path.join(taskDir, SESSION_FILE), `${id}\n`);
}

function opencodeRunArgs({ model, sessionID, prompt, cwd }) {
  const args = ["--pure", "run", "--format", "json", "--auto"];
  const prefixed = openRouterModelId(model);
  if (prefixed) {
    args.push("-m", prefixed);
  }
  if (cwd) {
    args.push("--dir", cwd);
  }
  if (sessionID) {
    args.push("--session", sessionID);
  }
  args.push(prompt);
  return args;
}

function buildOpenCodeConfig({ taskDir, env = process.env, planning = false, models = [] } = {}) {
  const config = {
    $schema: "https://opencode.ai/config.json",
    permission: planning
      ? { "*": "allow", edit: "deny", question: "deny" }
      : { "*": "allow" },
  };
  const options = {};
  const apiKey = String((env && env.OPENROUTER_API_KEY) || "").trim();
  const baseURL = String((env && env.OPENROUTER_BASE_URL) || "").trim();
  if (apiKey) {
    options.apiKey = apiKey;
  }
  if (baseURL) {
    options.baseURL = baseURL;
  }
  const modelIds = uniqueCatalogModels(models);
  const modelEntries = {};
  for (const id of modelIds) {
    modelEntries[id] = {
      options: {
        provider: {
          allow_fallbacks: false,
          sort: "throughput",
        },
      },
    };
  }
  if (Object.keys(options).length > 0 || Object.keys(modelEntries).length > 0) {
    config.provider = { openrouter: {} };
    if (Object.keys(options).length > 0) {
      config.provider.openrouter.options = options;
    }
    if (Object.keys(modelEntries).length > 0) {
      config.provider.openrouter.models = modelEntries;
    }
  }
  if (planning && taskDir) {
    config.mcp = {
      superplane: {
        type: "local",
        command: ["node", path.join(taskDir, "planning_session_mcp.js")],
        enabled: true,
      },
    };
  }
  return config;
}

function writeOpenCodeConfig(taskDir, env, models) {
  const config = buildOpenCodeConfig({
    taskDir,
    env,
    planning: planningEnabled(env),
    models,
  });
  fs.writeFileSync(path.join(taskDir, "opencode.json"), `${JSON.stringify(config, null, 2)}\n`);
}

function openCodeProcessEnv(taskDir, baseEnv = process.env) {
  const xdg = path.join(taskDir, "xdg");
  return {
    ...baseEnv,
    OPENCODE_CONFIG: path.join(taskDir, "opencode.json"),
    OPENCODE_DISABLE_AUTOUPDATE: "1",
    OPENCODE_DISABLE_MODELS_FETCH: "1",
    OPENCODE_DISABLE_LSP_DOWNLOAD: "1",
    OPENCODE_DISABLE_CLAUDE_CODE: "1",
    OPENCODE_PURE: "1",
    XDG_DATA_HOME: path.join(xdg, "data"),
    XDG_CONFIG_HOME: path.join(xdg, "config"),
    XDG_CACHE_HOME: path.join(xdg, "cache"),
  };
}

function ensureXdgDirs(taskDir) {
  for (const name of ["data", "config", "cache"]) {
    fs.mkdirSync(path.join(taskDir, "xdg", name), { recursive: true });
  }
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    console.error("usage: node run.js <prompt-file> [model]");
    process.exit(2);
  }
  runPrompt(args[0], args[1] || "")
    .then((code) => process.exit(code))
    .catch((err) => {
      console.error(err && err.message ? err.message : err);
      process.exit(1);
    });
}

async function runPrompt(promptFile, model, helpers = {}) {
  const env = helpers.env || process.env;
  const sp = env.SUPERPLANE_TASK_DIR;
  if (!sp) {
    throw new Error("SUPERPLANE_TASK_DIR is required");
  }
  const resultFile = env.SUPERPLANE_RESULT_FILE;
  if (!resultFile) {
    throw new Error("SUPERPLANE_RESULT_FILE is required");
  }

  let prompt = fs.readFileSync(promptFile, "utf8");
  const promptCountPath = path.join(sp, "prompt_count");
  const promptCount = Number.parseInt(fs.readFileSync(promptCountPath, "utf8").trim(), 10) || 0;
  const startedAt = Date.now();
  const now = helpers.now || Date.now;
  const sleep = helpers.sleep || defaultSleep;
  const cwd = helpers.cwd || process.cwd();
  const planning = planningEnabled(env);
  if (planning) {
    println("Planning session tools enabled");
    prompt = `${prompt}\n\n${PLANNING_SYSTEM_PROMPT}`;
  }

  ensureXdgDirs(sp);

  const currentModel = catalogModelId(model);
  writeOpenCodeConfig(sp, env, currentModel ? [currentModel] : []);
  const childEnv = openCodeProcessEnv(sp, env);

  const deadline = waitDeadlineMs(env, now);
  let lastResult = {};
  let lastUsage = emptyUsage();
  let lastCost = 0;
  let sessionID = readSessionID(sp);
  const continuing = promptCount > 0 && Boolean(sessionID);
  if (continuing) {
    println(`Continuing OpenCode session on ${openRouterModelId(currentModel) || currentModel}`);
  } else {
    const startModel = openRouterModelId(currentModel) || model;
    if (startModel) {
      println(`Starting OpenCode · ${startModel}`);
    } else {
      println("Starting OpenCode");
    }
  }

  const telemetry = loadTurnTelemetry();
  const formatter = createOpenCodeFormatter(telemetry, (id) => {
    sessionID = id || sessionID;
    writeSessionID(sp, sessionID);
  });

  let failed = false;
  let exitCode = 0;
  let lastErrorText = "";
  let attemptLabel = "";
  let attempt = 1;

  while (true) {
    if (attemptLabel) {
      println(attemptLabel);
      attemptLabel = "";
    }
    const sessionBeforeAttempt = sessionID;
    const args = opencodeRunArgs({
      model: currentModel,
      sessionID: sessionID || undefined,
      prompt,
      cwd,
    });
    const spawnResult = await spawnOpenCodeTurn(args, childEnv, cwd, formatter, helpers);
    if (spawnResult.sessionID) {
      sessionID = spawnResult.sessionID;
      writeSessionID(sp, sessionID);
    }
    if (spawnResult.usage && tokenTotal(spawnResult.usage) > 0) {
      lastUsage = spawnResult.usage;
      lastResult = spawnResult.lastEvent || lastResult;
    }
    if (spawnResult.cost != null) {
      lastCost = spawnResult.cost;
    }
    lastErrorText = spawnResult.errorText || "";
    const spawnFailed = spawnTurnFailed(spawnResult, formatter, planning);
    if (!spawnFailed) {
      failed = false;
      exitCode = 0;
      break;
    }
    const classKind = classifyOpenRouterError(lastErrorText);
    const failedExit = spawnResult.exitCode !== 0 ? spawnResult.exitCode : 1;
    if (!spawnResult.sessionID) {
      sessionID = sessionBeforeAttempt;
      restoreSession(sp, sessionID);
    }
    if (!isRetryableKind(classKind)) {
      failed = true;
      exitCode = failedExit;
      if (lastErrorText) {
        println(truncateText(lastErrorText));
      }
      break;
    }
    if (attempt >= MAX_ATTEMPTS) {
      failed = true;
      exitCode = failedExit;
      if (lastErrorText) {
        println(truncateText(lastErrorText));
      }
      println(stoppedAfterAttemptsLine(classKind, currentModel));
      break;
    }
    const waitMs = retryWaitMs(lastErrorText, attempt);
    const remaining = deadline == null ? waitMs : deadline - now();
    if (remaining < waitMs || remaining <= 0) {
      failed = true;
      exitCode = 1;
      if (lastErrorText) {
        println(truncateText(lastErrorText));
      }
      println(waitExceededTimeoutLine(classKind));
      break;
    }
    if (lastErrorText) {
      println(truncateText(lastErrorText));
    }
    println(retryWaitLine(classKind, currentModel, waitMs, attempt + 1));
    await sleep(waitMs);
    attempt += 1;
    attemptLabel = `Retrying OpenCode · ${openRouterModelId(currentModel)}`;
  }

  formatter.flush(failed);
  const usage = lastUsage;
  if (tokenTotal(usage) > 0) {
    telemetry.updateCurrentUsage(usage);
  }
  const payload = {
    type: "result",
    result: resultTextFrom(lastResult, formatter),
    model: openRouterModelId(currentModel) || model,
    usage,
  };
  if (lastCost) {
    payload.total_cost_usd = lastCost;
  }
  telemetry.attachToResult(payload, { name: promptSeriesName(promptFile) });
  fs.writeFileSync(resultFile, `${JSON.stringify(payload)}\n`);
  accumulateLLMUsage(payload);
  fs.writeFileSync(promptCountPath, `${promptCount + 1}\n`);
  formatTurnResult({
    is_error: failed,
    num_turns: payload.telemetry && payload.telemetry.num_turns ? payload.telemetry.num_turns : 1,
    duration_ms: Date.now() - startedAt,
    total_cost_usd: payload.total_cost_usd,
  });
  return failed ? exitCode || 1 : 0;
}

function waitDeadlineMs(env, now) {
  const seconds = Number((env && env.SUPERPLANE_EXECUTION_TIMEOUT_SECONDS) || 0);
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return now() + DEFAULT_WAIT_CAP_MS;
  }
  return now() + seconds * 1000;
}

function defaultSleep(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

function commandExists(name) {
  const result = spawnSync("sh", ["-c", `command -v ${name}`], { encoding: "utf8" });
  return result.status === 0;
}

function defaultSpawnOpenCode(args, options) {
  let command = "opencode";
  let spawnArgs = args;
  if (commandExists("stdbuf")) {
    command = "stdbuf";
    spawnArgs = ["-oL", "-eL", "opencode", ...args];
  }
  return spawn(command, spawnArgs, {
    stdio: ["pipe", "pipe", "pipe"],
    env: options.env,
    cwd: options.cwd,
  });
}

function spawnHasCompleteAssistantTurn(spawnResult, formatter) {
  const text = formatter && typeof formatter.lastText === "function" ? formatter.lastText() : "";
  if (!String(text || "").trim()) {
    return false;
  }
  if (spawnResult && spawnResult.roundOpen) {
    return false;
  }
  return tokenTotal(spawnResult && spawnResult.usage) > 0;
}

function spawnTurnFailed(spawnResult, formatter, allowSoftExitWithReply) {
  if (spawnResult.resultFailed) {
    return true;
  }
  if (spawnResult.exitCode === 0) {
    return false;
  }
  const kind = classifyOpenRouterError(spawnResult.errorText || "");
  if (kind === "rate_limit" || kind === "hard" || kind === "retryable") {
    return true;
  }
  if (!allowSoftExitWithReply) {
    return true;
  }
  return !spawnHasCompleteAssistantTurn(spawnResult, formatter);
}

async function spawnOpenCodeTurn(args, env, cwd, formatter, helpers) {
  if (formatter && typeof formatter.beginSpawn === "function") {
    formatter.beginSpawn();
  }
  const spawnOpenCode = helpers.spawnOpenCode || defaultSpawnOpenCode;
  const child = spawnOpenCode(args, { env, cwd });
  if (child.stdin && typeof child.stdin.end === "function") {
    child.stdin.end();
  }
  let stderrText = "";
  if (child.stderr) {
    child.stderr.on("data", (chunk) => {
      stderrText += String(chunk);
      process.stderr.write(chunk);
    });
  }

  const stdout = child.stdout;
  const rl = stdout ? readline.createInterface({ input: stdout, crlfDelay: Infinity }) : null;
  if (rl) {
    rl.on("line", (raw) => formatter.handleLine(raw));
  }
  const stdoutDone = rl
    ? new Promise((resolve) => {
        rl.on("close", resolve);
      })
    : Promise.resolve();

  let exitCode;
  try {
    exitCode = await new Promise((resolve, reject) => {
      child.on("error", (err) => {
        if (stdout && typeof stdout.destroy === "function") {
          stdout.destroy();
        }
        reject(err);
      });
      child.on("close", (code) => resolve(code == null ? 1 : code));
    });
  } catch (err) {
    await stdoutDone;
    throw err;
  }
  await stdoutDone;

  const snapshot = formatter.snapshot();
  const errorText = [snapshot.errorText, stderrText].filter(Boolean).join("\n");
  return {
    exitCode,
    errorText,
    resultFailed: snapshot.resultFailed,
    sessionID: snapshot.sessionID,
    usage: snapshot.usage,
    cost: snapshot.cost,
    lastEvent: snapshot.lastEvent,
    roundOpen: Boolean(snapshot.roundOpen),
  };
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

function tokenTotal(usage) {
  if (!usage || typeof usage !== "object") {
    return 0;
  }
  return (
    Number(usage.input_tokens || 0) +
    Number(usage.output_tokens || 0) +
    Number(usage.cache_read_input_tokens || 0) +
    Number(usage.cache_creation_input_tokens || 0) +
    Number(usage.reasoning_tokens || 0)
  );
}

function mergeUsage(left, right) {
  const merged = {
    input_tokens: Number((left && left.input_tokens) || 0) + Number((right && right.input_tokens) || 0),
    output_tokens: Number((left && left.output_tokens) || 0) + Number((right && right.output_tokens) || 0),
    cache_read_input_tokens:
      Number((left && left.cache_read_input_tokens) || 0) + Number((right && right.cache_read_input_tokens) || 0),
    cache_creation_input_tokens:
      Number((left && left.cache_creation_input_tokens) || 0) + Number((right && right.cache_creation_input_tokens) || 0),
    reasoning_tokens: Number((left && left.reasoning_tokens) || 0) + Number((right && right.reasoning_tokens) || 0),
  };
  const cost = Number((left && left.total_cost_usd) || 0) + Number((right && right.total_cost_usd) || 0);
  if (cost) {
    merged.total_cost_usd = cost;
  }
  return merged;
}

function usageFromStepFinish(part) {
  const tokens = (part && part.tokens) || {};
  const cache = tokens.cache || {};
  const usage = {
    input_tokens: Number(tokens.input || 0),
    output_tokens: Number(tokens.output || 0),
    cache_read_input_tokens: Number(cache.read || 0),
    cache_creation_input_tokens: Number(cache.write || 0),
    reasoning_tokens: Number(tokens.reasoning || 0),
  };
  if (part && part.cost != null && Number.isFinite(Number(part.cost))) {
    usage.total_cost_usd = Number(part.cost);
  }
  return usage;
}

function collectErrorText(value, parts, depth) {
  if (value == null || depth > 5) {
    return;
  }
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    const text = String(value).trim();
    if (text) {
      parts.push(text);
    }
    return;
  }
  if (typeof value !== "object") {
    return;
  }
  for (const key of ["message", "name", "code", "status", "statusCode", "type", "responseBody"]) {
    if (value[key] != null && value[key] !== value) {
      collectErrorText(value[key], parts, depth + 1);
    }
  }
  if (value.data && value.data !== value) {
    collectErrorText(value.data, parts, depth + 1);
  }
  if (value.error && value.error !== value) {
    collectErrorText(value.error, parts, depth + 1);
  }
  if (value.cause && value.cause !== value) {
    collectErrorText(value.cause, parts, depth + 1);
  }
}

function errorTextFromEvent(event) {
  const parts = [];
  collectErrorText(event, parts, 0);
  return [...new Set(parts)].join(" ");
}

function resultTextFrom(lastEvent, formatter) {
  if (formatter && typeof formatter.lastText === "function") {
    const text = formatter.lastText();
    if (text) {
      return text;
    }
  }
  const part = lastEvent && lastEvent.part;
  if (part && typeof part.text === "string") {
    return part.text;
  }
  return "";
}

function accumulateLLMUsage(payload) {
  const taskDir = process.env.SUPERPLANE_TASK_DIR;
  if (!taskDir) {
    return;
  }
  const script = path.join(taskDir, "llm_usage.js");
  if (!fs.existsSync(script)) {
    return;
  }
  require(script).accumulate(taskDir, payload);
}

function loadTurnTelemetry() {
  const taskDir = process.env.SUPERPLANE_TASK_DIR;
  const candidates = [];
  if (taskDir) {
    candidates.push(path.join(taskDir, "turn_telemetry.js"));
  }
  candidates.push(path.join(__dirname, "..", "turn_telemetry.js"));
  for (const file of candidates) {
    if (fs.existsSync(file)) {
      return require(file).createTurnTelemetry();
    }
  }
  return require("../turn_telemetry").createTurnTelemetry();
}

function promptSeriesName(promptFile) {
  const base = path.basename(promptFile || "", path.extname(promptFile || ""));
  const words = base.replace(/^\d+-/, "").split("-").filter(Boolean);
  if (words.length === 0) {
    return "";
  }
  return words.map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join(" ");
}

function writeLiveLogRecord(rec) {
  process.stdout.write(`${JSON.stringify(rec)}\n`);
}

function println(text = "") {
  process.stdout.write(`${text}\n`);
}

function createOpenCodeFormatter(telemetry, onSession) {
  const tracker = telemetry || loadTurnTelemetry();
  const tools = createToolTracker(tracker);
  let sessionID = "";
  let resultFailed = false;
  let errorText = "";
  let lastText = "";
  let lastEvent = {};
  let usage = emptyUsage();
  let cost = 0;
  let roundOpen = false;
  let announcedStart = false;

  function rememberSession(event) {
    const id = String((event && event.sessionID) || (event && event.part && event.part.sessionID) || "").trim();
    if (!id) {
      return;
    }
    sessionID = id;
    if (onSession) {
      onSession(id);
    }
  }

  function beginRound(eventUsage, extra) {
    tracker.beginTurn(eventUsage, extra);
    roundOpen = true;
  }

  return {
    beginSpawn() {
      resultFailed = false;
      errorText = "";
      announcedStart = false;
      lastText = "";
      usage = emptyUsage();
      cost = 0;
      roundOpen = false;
      sessionID = "";
    },
    handleLine(raw) {
      const line = String(raw || "").trim();
      if (!line) {
        return;
      }
      let event;
      try {
        event = JSON.parse(line);
      } catch {
        println(line);
        return;
      }
      if (!event || typeof event !== "object" || Array.isArray(event)) {
        return;
      }
      this.handleEvent(event);
    },
    handleEvent(event) {
      rememberSession(event);
      lastEvent = event;
      const type = String(event.type || "");
      const part = event.part && typeof event.part === "object" ? event.part : {};
      switch (type) {
        case "step_start":
          if (!announcedStart) {
            announcedStart = true;
            println("OpenCode started");
          }
          beginRound(undefined, {});
          break;
        case "text": {
          const text = typeof part.text === "string" ? part.text : "";
          if (text.trim()) {
            lastText = text.replace(/\s+$/, "");
            if (!roundOpen) {
              beginRound(undefined, { message: lastText });
            }
            println(lastText);
          }
          break;
        }
        case "reasoning": {
          const thinking = typeof part.text === "string" ? part.text : "";
          if (thinking.trim()) {
            println("Thinking");
            println(truncateText(thinking.trim()));
            println();
          }
          break;
        }
        case "tool_use":
          if (!roundOpen) {
            beginRound(undefined, { hasTools: true });
          }
          formatToolUse(part, tools);
          break;
        case "step_finish": {
          const stepUsage = usageFromStepFinish(part);
          usage = mergeUsage(usage, stepUsage);
          if (part.cost != null && Number.isFinite(Number(part.cost))) {
            cost = Number(cost || 0) + Number(part.cost);
          }
          if (!roundOpen) {
            beginRound(stepUsage, { message: lastText });
          } else {
            tracker.updateCurrentUsage(usage);
          }
          roundOpen = false;
          break;
        }
        case "error": {
          const text = errorTextFromEvent(event);
          errorText = text;
          if (classifyOpenRouterError(text) === "hard") {
            resultFailed = true;
          }
          break;
        }
        default:
          break;
      }
    },
    flush(failed) {
      tools.flush(Boolean(failed));
    },
    snapshot() {
      return {
        sessionID,
        resultFailed,
        errorText,
        usage,
        cost,
        lastEvent,
        roundOpen,
      };
    },
    lastText() {
      return lastText;
    },
    resultFailed() {
      return resultFailed;
    },
  };
}

function createToolTracker(telemetry) {
  const openTools = new Map();
  const fifo = [];
  let anonSeq = 0;

  function resolveKey(id, creating) {
    const key = id != null && String(id).trim() ? String(id).trim() : "";
    if (key) {
      return key;
    }
    if (creating) {
      const generated = `anon-${anonSeq}`;
      anonSeq += 1;
      fifo.push(generated);
      return generated;
    }
    return fifo.shift() || "";
  }

  return {
    start(kind, text, id) {
      const key = resolveKey(id, true);
      const startedAt = Date.now();
      openTools.set(key, { kind, text: text || kind, startedAt, emitted: false });
      return key;
    },
    emitStart(id) {
      let key = id != null && String(id).trim() ? String(id).trim() : "";
      if (!key || !openTools.has(key)) {
        key = fifo[0] || key;
      }
      const tool = openTools.get(key);
      if (!tool || tool.emitted) {
        return key;
      }
      tool.emitted = true;
      writeLiveLogRecord(
        telemetry.stampToolStart({
          type: "tool_start",
          id: key,
          kind: tool.kind,
          text: tool.text,
          started_at: tool.startedAt,
        }),
      );
      return key;
    },
    end(failed, id) {
      this.emitStart(id);
      let key = resolveKey(id, false);
      if (!openTools.has(key)) {
        key = fifo.shift() || key;
      }
      const tool = openTools.get(key);
      if (!tool) {
        return;
      }
      openTools.delete(key);
      const fifoIndex = fifo.indexOf(key);
      if (fifoIndex >= 0) {
        fifo.splice(fifoIndex, 1);
      }
      writeLiveLogRecord(
        telemetry.stampToolEnd({
          type: "tool_end",
          id: key,
          kind: tool.kind,
          status: failed ? "failed" : "passed",
          duration_ms: Math.max(0, Date.now() - tool.startedAt),
        }),
      );
    },
    flush(failed) {
      for (const key of [...openTools.keys()]) {
        this.end(failed, key);
      }
    },
  };
}

function formatToolUse(part, tools) {
  const name = String(part.tool || part.name || "tool");
  const kind = name.toLowerCase();
  const state = part.state && typeof part.state === "object" ? part.state : {};
  const id = part.callID || part.id || "";
  const started = tools.start(kind, toolInputDetail(name, state.input) || kind, id);
  tools.emitStart(started);
  const output = typeof state.output === "string" ? state.output : "";
  if (output.trim()) {
    println(truncateText(output.replace(/\s+$/, "")));
  } else if (state.error) {
    const errText = typeof state.error === "string" ? state.error : JSON.stringify(state.error);
    if (errText.trim()) {
      println(truncateText(errText));
    }
  }
  const failed = state.status === "error" || Boolean(state.error);
  tools.end(failed, started);
}

function toolInputDetail(name, rawInput) {
  if (rawInput == null || typeof rawInput !== "object" || Array.isArray(rawInput)) {
    if (rawInput == null) {
      return "";
    }
    return truncateText(String(rawInput));
  }
  const lowered = name.toLowerCase();
  if (lowered === "bash") {
    const command = rawInput.command;
    if (typeof command === "string" && command.trim()) {
      return command.trim().split(/\r?\n/).join(" ");
    }
  }
  if (["read", "write", "edit", "notebookedit"].includes(lowered)) {
    for (const key of ["file_path", "path", "notebook_path"]) {
      const value = rawInput[key];
      if (typeof value === "string" && value.trim()) {
        let detail = value.trim();
        if ((lowered === "write" || lowered === "edit") && typeof rawInput.content === "string") {
          detail += ` (${rawInput.content.length} chars)`;
        }
        return detail;
      }
    }
  }
  if (lowered === "grep") {
    const parts = [];
    if (rawInput.pattern) {
      parts.push(`pattern: ${rawInput.pattern}`);
    }
    if (rawInput.path) {
      parts.push(`path: ${rawInput.path}`);
    }
    if (parts.length) {
      return parts.join(" · ");
    }
  }
  if (lowered === "glob" && rawInput.pattern) {
    return String(rawInput.pattern);
  }
  try {
    return truncateText(JSON.stringify(rawInput));
  } catch {
    return truncateText(String(rawInput));
  }
}

function truncateText(text) {
  let lines = String(text || "").split(/\r?\n/);
  if (lines.length > TOOL_RESULT_MAX_LINES) {
    const kept = lines.slice(0, TOOL_RESULT_MAX_LINES);
    const omitted = lines.length - TOOL_RESULT_MAX_LINES;
    text = `${kept.join("\n")}\n… (${omitted} more lines)`;
    lines = text.split(/\r?\n/);
  }
  if (text.length > TOOL_RESULT_MAX_CHARS) {
    text = `${text.slice(0, TOOL_RESULT_MAX_CHARS - 1).replace(/\s+$/, "")}…`;
  }
  return text;
}

function formatTurnResult(event) {
  const isError = Boolean(event && event.is_error);
  const status = isError ? "failed" : "done";
  const parts = [isError ? `✗ ${status}` : `✓ ${status}`];
  if (event && event.num_turns != null) {
    parts.push(`${event.num_turns} turns`);
  }
  if (event && event.total_cost_usd != null) {
    const cost = Number(event.total_cost_usd);
    parts.push(Number.isFinite(cost) ? `$${cost.toFixed(4)}` : `$${event.total_cost_usd}`);
  }
  if (event && event.duration_ms != null) {
    const ms = Number(event.duration_ms);
    if (Number.isFinite(ms)) {
      parts.push(`${(ms / 1000).toFixed(1)}s`);
    }
  }
  process.stdout.write(`${parts.join(" · ")}\n`);
}

function formatOpenCodeJsonLines(rawLines) {
  const formatter = createOpenCodeFormatter(loadTurnTelemetry());
  for (const line of rawLines) {
    formatter.handleLine(line);
  }
  const failed = formatter.resultFailed();
  formatter.flush(failed);
  return { failed };
}

if (require.main === module) {
  main();
}

module.exports = {
  runPrompt,
  formatOpenCodeJsonLines,
  formatTurnResult,
  opencodeRunArgs,
  planningEnabled,
  classifyOpenRouterError,
  openRouterModelId,
  buildOpenCodeConfig,
  retryWaitMs,
  waitDeadlineMs,
  openCodeProcessEnv,
};
