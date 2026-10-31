#!/usr/bin/env node
"use strict";

/**
 * Run Codex CLI and emit typed live-log records.
 *
 *   node run.js <prompt-file> [model] [thinking]
 */

const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { spawn } = require("child_process");

const SESSION_FILE = "codex_session";

// Codex starts stdio MCP servers with a minimal environment. Forward by name,
// not with mcp_servers.<id>.env, so tokens stay out of the codex argv.
const GLOBAL_MCP_SERVER = "superplane_global";
const GLOBAL_MCP_ENV_VARS = [
  "SUPERPLANE_BASE_URL",
  "SUPERPLANE_RUNNER_TOKEN",
  "SUPERPLANE_TASK_DIR",
  "SUPERPLANE_ANALYSIS_SPEC_FILE",
];

function loadActivityStreamModule() {
  const taskDir = process.env.SUPERPLANE_TASK_DIR || "";
  const candidates = [
    path.join(taskDir, "activity_stream.js"),
    path.join(__dirname, "..", "activity_stream.js"),
  ];
  for (const file of candidates) {
    if (file && fs.existsSync(file)) {
      return require(file);
    }
  }
  return { createActivityStream: () => createDisabledActivityStream() };
}

function createDisabledActivityStream() {
  const noop = () => undefined;
  return {
    enabled: false,
    activityId: "",
    start: noop,
    startContent: noop,
    appendContent: noop,
    endContent: noop,
    startTool: (input) => String((input && input.id) || ""),
    appendToolOutput: noop,
    endTool: noop,
    notice: noop,
    end: noop,
    flush: () => Promise.resolve(),
  };
}

function loadAnalysisProtocolModule() {
  const candidates = [
    path.join(__dirname, "analysis_protocol.js"),
    path.join(__dirname, "..", "analysis_protocol.js"),
  ];
  for (const file of candidates) {
    try {
      return require(file);
    } catch (_err) {
      // try the next path
    }
  }
  return {};
}

function loadAnalysisProtocol(env = process.env) {
  const mod = loadAnalysisProtocolModule();
  return typeof mod.analysisProtocol === "function"
    ? mod.analysisProtocol(env)
    : "";
}

function withoutEmbeddedAnalysisProtocol(prompt) {
  const mod = loadAnalysisProtocolModule();
  if (typeof mod.withoutEmbeddedAnalysisProtocol === "function") {
    return mod.withoutEmbeddedAnalysisProtocol(prompt);
  }
  return prompt;
}

function applyAnalysisContinuation(taskDir, promptCount, prompt) {
  if (!planningAnalysisEnabled()) {
    return prompt;
  }
  const mod = loadAnalysisProtocolModule();
  if (typeof mod.withAnalysisContinuation !== "function") {
    return prompt;
  }
  return mod.withAnalysisContinuation(taskDir, promptCount, prompt);
}

function envFlag(env, name) {
  return Boolean(envValue(env, name));
}

function envValue(env, name) {
  return String((env && env[name]) || "").trim();
}

function requireCodexAPIKey(env = process.env) {
  const apiKey = envValue(env, "CODEX_API_KEY") || envValue(env, "OPENAI_API_KEY");
  if (!apiKey) {
    throw new Error(
      "Codex API key is missing. Set OPENAI_API_KEY or CODEX_API_KEY.",
    );
  }
  return apiKey;
}

function codexExecEnv(env = process.env) {
  if (envValue(env, "SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN")) {
    const subscription = loadSubscriptionModule(env);
    if (!subscription) {
      throw new Error("ChatGPT subscription helper is missing");
    }
    return subscription.subscriptionChildEnv(env);
  }
  return {
    ...env,
    CODEX_API_KEY: requireCodexAPIKey(env),
  };
}

function planningEnabled(env = process.env) {
  return (
    planningAnalysisEnabled(env) &&
    envFlag(env, "SUPERPLANE_PLANNING_SESSION_ID")
  );
}

function planningAnalysisEnabled(env = process.env) {
  return env.SUPERPLANE_PLANNING_SESSION_KIND === "work_order_analysis";
}

function artifactEnabled(env = process.env) {
  return envFlag(env, "SUPERPLANE_ARTIFACT_TOKEN");
}

function visualEvidenceEnabled(env = process.env) {
  if (envFlag(env, "SUPERPLANE_VISUAL_EVIDENCE")) return true;
  return artifactEnabled(env) && !envFlag(env, "SUPERPLANE_PUBLISH_LINKS");
}

function planningSystemPrompt(env = process.env) {
  return planningAnalysisEnabled(env) ? loadAnalysisProtocol(env) : "";
}

function thinkingArgs(thinking) {
  const level = String(thinking || "")
    .trim()
    .toLowerCase();
  if (["minimal", "low", "medium", "high", "xhigh"].includes(level)) {
    return ["-c", `model_reasoning_effort=${tomlString(level)}`];
  }
  return [];
}

function codexExecArgs(
  env = process.env,
  model,
  mcpScriptPath,
  sessionID = "",
  thinking = "",
) {
  const args = ["exec"];
  if (sessionID) {
    args.push("resume", sessionID);
  }
  args.push("--json", "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox");
  args.push(...workspaceMCPConfigOverrides(env));
  if (planningEnabled(env)) {
    args.push(
      "-c",
      `developer_instructions=${tomlString(loadAnalysisProtocol(env))}`,
    );
  }
  args.push(...globalMCPConfigOverrides(taskDirFromMCPPath(mcpScriptPath), env));
  if (model) {
    args.push("-m", model);
  }
  args.push(...thinkingArgs(thinking));
  const baseURL = envValue(env, "OPENAI_BASE_URL");
  if (baseURL) {
    args.push("-c", `openai_base_url=${tomlString(baseURL)}`);
  }
  return args;
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
  if (id) {
    fs.writeFileSync(path.join(taskDir, SESSION_FILE), `${id}\n`);
  }
}

function codexSessionForPrompt(promptCount, sessionID, env = process.env) {
  if (planningAnalysisEnabled(env) && envFlag(env, "SUPERPLANE_ANALYSIS_REWIND")) {
    return "";
  }
  if (Number(promptCount) < 1) {
    return "";
  }
  const id = String(sessionID || "").trim();
  if (!id) {
    throw new Error("Codex session ID is missing for a follow-up prompt");
  }
  return id;
}

function codexSessionIDFromEvent(event) {
  return String(
    (event && (event.thread_id || (event.thread && event.thread.id))) || "",
  ).trim();
}

function taskDirFromMCPPath(value) {
  const candidate = String(value || "");
  return candidate.endsWith(".js") ? path.dirname(candidate) : candidate;
}

function loadGlobalMCPRegistry(taskDir) {
  const candidates = [
    path.join(taskDir, "superplane_mcp", "registry.js"),
    path.join(__dirname, "..", "superplane_mcp", "registry.js"),
  ];
  for (const file of candidates) {
    if (fs.existsSync(file)) {
      return require(file);
    }
  }
  return null;
}

// The global server lists only the tools that the run token allows. Approve
// every known global tool: Codex exec cannot answer an approval prompt, and the
// backend rejects a call outside the token scope.
function globalMCPConfigOverrides(taskDir, env = process.env) {
  if (!envFlag(env, "SUPERPLANE_RUNNER_TOKEN")) {
    return [];
  }
  const registry = loadGlobalMCPRegistry(taskDir);
  const toolNames = registry ? registry.allTools().map((tool) => tool.definition.name) : [];
  return [
    "-c",
    `mcp_servers.${GLOBAL_MCP_SERVER}.command=${tomlString("node")}`,
    "-c",
    `mcp_servers.${GLOBAL_MCP_SERVER}.args=${tomlStringArray([path.join(taskDir, "superplane_mcp", "server.js")])}`,
    "-c",
    `mcp_servers.${GLOBAL_MCP_SERVER}.env_vars=${tomlStringArray(GLOBAL_MCP_ENV_VARS)}`,
    ...toolNames.flatMap((name) => [
      "-c",
      `mcp_servers.${GLOBAL_MCP_SERVER}.tools.${tomlKey(name)}.approval_mode="approve"`,
    ]),
  ];
}

function workspaceMCPConfigPath(env = process.env) {
  const taskDir = String((env && env.SUPERPLANE_TASK_DIR) || "").trim();
  const configured = String((env && env.SUPERPLANE_WORKSPACE_MCP_CONFIG) || "").trim();
  const expanded = taskDir
    ? configured
        .replace(/\$\{SUPERPLANE_TASK_DIR\}/g, taskDir)
        .replace(/\$SUPERPLANE_TASK_DIR/g, taskDir)
    : configured;
  const candidates = [expanded, configured];
  if (taskDir) {
    candidates.push(path.join(taskDir, "workspace_mcp.json"));
  }
  for (const candidate of candidates) {
    if (candidate && fs.existsSync(candidate)) {
      return candidate;
    }
  }
  return "";
}

function workspaceMCPServers(env = process.env) {
  const configPath = workspaceMCPConfigPath(env);
  if (!configPath) {
    return [];
  }
  try {
    const parsed = JSON.parse(fs.readFileSync(configPath, "utf8"));
    return Array.isArray(parsed.servers) ? parsed.servers : [];
  } catch (_err) {
    return [];
  }
}

function workspaceMCPConfigOverrides(env = process.env) {
  const args = [];
  for (const server of workspaceMCPServers(env)) {
    const name = String((server && server.name) || "").trim();
    const url = String((server && server.url) || "").trim();
    if (!name || name === "superplane" || name === GLOBAL_MCP_SERVER || !url) {
      continue;
    }
    const serverKey = tomlKey(name);
    args.push("-c", `mcp_servers.${serverKey}.url=${tomlString(url)}`);
    const headers = server.headers && typeof server.headers === "object" ? server.headers : {};
    for (const [headerName, headerValue] of Object.entries(headers)) {
      if (!headerName) {
        continue;
      }
      args.push("-c", `mcp_servers.${serverKey}.http_headers.${tomlKey(headerName)}=${tomlString(String(headerValue))}`);
    }
    const disabledTools = Array.isArray(server.disabledTools)
      ? server.disabledTools.map((name) => String(name || "").trim()).filter(Boolean)
      : [];
    if (disabledTools.length > 0) {
      args.push("-c", `mcp_servers.${serverKey}.disabled_tools=${tomlStringArray(disabledTools)}`);
    }
  }
  return args;
}

function tomlKey(value) {
  const key = String(value);
  if (/^[A-Za-z0-9_]+$/.test(key)) {
    return key;
  }
  return tomlString(key);
}

function tomlString(value) {
  return `"${String(value)
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\n/g, "\\n")
    .replace(/\r/g, "\\r")}"`;
}

function tomlStringArray(values) {
  return `[${values.map(tomlString).join(", ")}]`;
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    writeStderr("usage: node run.js <prompt-file> [model] [thinking]\n");
    process.exit(2);
  }
  runPrompt(args[0], args[1] || "", args[2] || "")
    .then((code) => process.exit(code))
    .catch((err) => {
      const message = safeLogText(err && err.message ? err.message : err).slice(0, 8192);
      writeStderr(`Codex error: ${message}\n`);
      saveFailureResult(message);
      process.exit(1);
    });
}

function saveFailureResult(message) {
  const resultFile = process.env.SUPERPLANE_RESULT_FILE;
  if (!resultFile) {
    return;
  }
  let previous = {};
  try {
    const saved = JSON.parse(fs.readFileSync(resultFile, "utf8"));
    if (saved && typeof saved === "object" && !Array.isArray(saved)) {
      previous = saved;
    }
  } catch (_err) {
    // A failure before the first turn may not have a saved result.
  }
  const payload = {
    ...previous,
    type: "result",
    is_error: true,
    error: previous.is_error && previous.error ? previous.error : message,
    result: previous.result || "",
  };
  try {
    fs.writeFileSync(resultFile, `${safeLogText(JSON.stringify(payload))}\n`);
  } catch (err) {
    writeStderr(`Codex error result could not be saved: ${err.message}\n`);
  }
}

async function runPrompt(promptFile, model, thinking) {
  const sp = process.env.SUPERPLANE_TASK_DIR;
  if (!sp) {
    throw new Error("SUPERPLANE_TASK_DIR is required");
  }
  const resultFile = process.env.SUPERPLANE_RESULT_FILE;
  if (!resultFile) {
    throw new Error("SUPERPLANE_RESULT_FILE is required");
  }
  let child;
  const subscription = loadSubscriptionModule();
  subscription?.setCredentialFailureHandler?.((error) => {
    // No untrusted output is safe after losing sight of credential rotation.
    // Stop the npm launcher and its native child, and save only a fixed error.
    if (child?.pid) {
      try {
        process.kill(-child.pid, "SIGKILL");
      } catch (_error) {
        child.kill("SIGKILL");
      }
    }
    process.stderr.write(`${error.message}\n`);
    try {
      fs.writeFileSync(resultFile, `${JSON.stringify({ type: "result", is_error: true, error: error.message, result: "" })}\n`);
    } catch (_error) {
      process.stderr.write("The credential protection error could not be saved.\n");
    }
    process.exit(1);
  });
  const childEnv = codexExecEnv(process.env);
  // Restore runs in a separate process. Observe credentials here before Codex
  // can replace or remove the auth file.
  redactSubscription("");

  const promptCountPath = path.join(sp, "prompt_count");
  const promptCount =
    Number.parseInt(fs.readFileSync(promptCountPath, "utf8").trim(), 10) || 0;
  const sessionID = codexSessionForPrompt(promptCount, readSessionID(sp));
  let prompt = applyAnalysisContinuation(
    sp,
    sessionID ? promptCount : 0,
    fs.readFileSync(promptFile, "utf8"),
  );
  if (planningAnalysisEnabled()) {
    prompt = withoutEmbeddedAnalysisProtocol(prompt);
  }

  const startedAt = Date.now();
  if (visualEvidenceEnabled()) {
    const outputDir = path.join(sp, "evidence");
    fs.mkdirSync(outputDir, { recursive: true });
    process.env.PLAYWRIGHT_MCP_OUTPUT_DIR = outputDir;
    process.env.PLAYWRIGHT_MCP_BROWSER =
      process.env.PLAYWRIGHT_MCP_BROWSER || "chromium";
  }
  const planning = planningEnabled();
  const activity = loadActivityStreamModule().createActivityStream({
    provider: "codex",
    turn: promptCount + 1,
  });
  activity.start();
  const codexArgs = codexExecArgs(
    process.env,
    model,
    sp,
    sessionID,
    thinking,
  );
  if (planning) {
    writeStdout("Planning session tools enabled\n");
  }
  if (sessionID) {
    writeStdout("Continuing Codex session in the current directory\n");
  }
  codexArgs.push(prompt);

  child = spawn("codex", codexArgs, {
    env: childEnv,
    stdio: ["ignore", "pipe", "pipe"],
    detached: Boolean(subscription?.setCredentialFailureHandler),
  });
  const stderrDone = pipeRedactedStderr(child.stderr);

  let lastResult = {};
  const telemetry = loadTurnTelemetry();
  const formatter = createCodexFormatter(telemetry, activity);
  const rl = readline.createInterface({
    input: child.stdout,
    crlfDelay: Infinity,
  });
  rl.on("line", (raw) => {
    const line = raw.trim();
    if (!line) {
      return;
    }
    try {
      const event = JSON.parse(line);
      if (event && typeof event === "object") {
        const nextSessionID = codexSessionIDFromEvent(event);
        if (nextSessionID) {
          writeSessionID(sp, nextSessionID);
        }
        if (event.usage || (event.item && event.item.usage)) {
          lastResult = event;
        }
        if (
          event.type === "item.completed" ||
          event.type === "turn.completed" ||
          event.type === "result"
        ) {
          if (
            !lastResult.usage &&
            !(lastResult.item && lastResult.item.usage)
          ) {
            lastResult = event;
          }
        }
        formatter.handleEvent(event);
        return;
      }
    } catch (_err) {
      writeStdout(`${line}\n`);
    }
  });

  const [processExitCode, , stderr] = await Promise.all([
    new Promise((resolve, reject) => {
      child.on("error", reject);
      child.on("close", (code) => resolve(code == null ? 1 : code));
    }),
    new Promise((resolve) => rl.on("close", resolve)),
    stderrDone,
  ]);

  const exitCode = formatter.errorMessage() && processExitCode === 0 ? 1 : processExitCode;
  if (exitCode !== 0 && !formatter.errorMessage()) {
    const message = `Codex failed with exit code ${exitCode}.`;
    formatter.reportError(stderr ? `${message}\n${stderr}` : message);
  }

  formatter.flush(exitCode !== 0);
  activity.end(exitCode === 0 ? "passed" : "failed");
  await activity.flush();
  const usage = extractUsage(lastResult);
  if (tokenTotal(usage) > 0) {
    telemetry.updateCurrentUsage(usage);
  }
  const rollout = readRolloutContext(childEnv.CODEX_HOME, readSessionID(sp));
  const payload = {
    type: "result",
    is_error: exitCode !== 0,
    ...(exitCode !== 0 ? { error: formatter.errorMessage() } : {}),
    result: formatter.lastText() || lastResult.result || lastResult.text || "",
    model: rollout.model || lastResult.model || model,
    thinkingLevel: rollout.effort || thinking,
    usage,
  };
  telemetry.attachToResult(payload);
  const activityModule = loadActivityStreamModule();
  const safePayload = activityModule.sanitizeLogValue ? activityModule.sanitizeLogValue(payload) : payload;
  if (typeof safePayload.result === "string") {
    safePayload.result = redactSubscription(safePayload.result);
  }
  fs.writeFileSync(resultFile, `${redactSubscription(JSON.stringify(safePayload))}\n`);
  accumulateLLMUsage(safePayload);
  fs.writeFileSync(promptCountPath, `${promptCount + 1}\n`);
  if (planning) {
    await require(path.join(sp, "agent_message.js")).recordAgentMessage(
      safePayload.result,
    );
  }
  formatTurnResult({
    is_error: exitCode !== 0,
    num_turns:
      safePayload.telemetry && safePayload.telemetry.num_turns
        ? safePayload.telemetry.num_turns
        : 1,
    duration_ms: Date.now() - startedAt,
  });
  if (exitCode === 0 && subscription && subscription.subscriptionEnabled()) {
    try {
      await subscription.reportUsage();
    } catch (err) {
      writeStderr(`${err && err.message ? err.message : "ChatGPT usage was not saved"}\n`);
    }
  }
  return exitCode;
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

function loadTurnTelemetry() {
  const taskDir = process.env.SUPERPLANE_TASK_DIR;
  const candidates = [];
  if (taskDir) {
    candidates.push(path.join(taskDir, "turn_telemetry.js"));
  }
  candidates.push(path.join(__dirname, "..", "turn_telemetry.js"));
  for (const file of candidates) {
    if (fs.existsSync(file)) {
      return require(file).createTurnTelemetry({
        write: writeLiveLogRecord,
        sanitize: loadActivityStreamModule().sanitizeLogValue,
      });
    }
  }
  return require("../turn_telemetry").createTurnTelemetry({
    write: writeLiveLogRecord,
    sanitize: loadActivityStreamModule().sanitizeLogValue,
  });
}

function extractUsage(event) {
  const source = event.usage || (event.item && event.item.usage) || event;
  const usage = {
    input_tokens: Number(source.input_tokens || source.prompt_tokens || 0),
    output_tokens: Number(
      source.output_tokens || source.completion_tokens || 0,
    ),
    cache_read_input_tokens: Number(
      source.cached_input_tokens || source.cache_read_input_tokens || 0,
    ),
    cache_creation_input_tokens: Number(source.cache_write_input_tokens || 0),
    reasoning_tokens: Number(
      source.reasoning_output_tokens || source.reasoning_tokens || 0,
    ),
  };
  const cost = event.total_cost_usd ?? source.total_cost_usd;
  if (cost != null && Number.isFinite(Number(cost))) usage.total_cost_usd = Number(cost);
  return usage;
}

function readRolloutContext(home, sessionID) {
  const sessions = home && path.join(home, "sessions");
  if (!sessions || !sessionID || !fs.existsSync(sessions)) {
    return {};
  }
  try {
    const suffix = `-${sessionID}.jsonl`;
    const files = fs
      .readdirSync(sessions, { recursive: true })
      .filter((file) => path.basename(file).startsWith("rollout-") && file.endsWith(suffix))
      .sort();
    const context = {};
    for (const file of files) {
      for (const line of fs.readFileSync(path.join(sessions, file), "utf8").split("\n")) {
        if (!line.trim()) continue;
        try {
          const record = JSON.parse(line);
          if (record.type !== "turn_context" || !record.payload) continue;
          if (typeof record.payload.model === "string") context.model = record.payload.model;
          if (typeof record.payload.effort === "string") context.effort = record.payload.effort;
        } catch (_err) {
          // Rollout files are internal Codex state. Ignore records from formats we do not know.
        }
      }
    }
    return context;
  } catch (_err) {
    return {};
  }
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

function writeLiveLogRecord(rec) {
  const activity = loadActivityStreamModule();
  const safe = activity.sanitizeLogValue ? activity.sanitizeLogValue(rec) : rec;
  process.stdout.write(`${JSON.stringify(safe)}\n`);
}

function loadSubscriptionModule(env = process.env) {
  const taskDir = env.SUPERPLANE_TASK_DIR || "";
  const file = path.join(taskDir, "codex_subscription.js");
  if (!taskDir || !fs.existsSync(file)) {
    return null;
  }
  return require(file);
}

function redactSubscription(value, options = {}) {
  const helper = loadSubscriptionModule();
  if (!helper || !helper.redactSecrets) {
    return String(value);
  }
  return helper.redactSecrets(String(value), process.env, options);
}

function writeStdout(value) {
  process.stdout.write(safeLogText(value));
}

function writeStderr(value) {
  process.stderr.write(safeLogText(value));
}

function safeLogText(value) {
  const activity = loadActivityStreamModule();
  const safe = activity.sanitizeLogText ? activity.sanitizeLogText(value) : String(value);
  return redactSubscription(safe);
}

function pipeRedactedStderr(stream) {
  let tail = "";
  const lines = readline.createInterface({ input: stream, crlfDelay: Infinity });
  lines.on("line", (line) => {
    const safe = safeLogText(line);
    tail = `${tail}${safe}\n`.slice(-8192);
    writeStderr(`${safe}\n`);
  });
  return new Promise((resolve) => lines.on("close", () => resolve(tail.trim())));
}

function createCodexFormatter(telemetry, activityOverride) {
  const tracker = telemetry || loadTurnTelemetry();
  const activity =
    activityOverride ||
    loadActivityStreamModule().createActivityStream({ provider: "codex" });
  const open = new Map();
  const anonQueue = [];
  const contentText = new Map();
  const anonymousContent = new Map();
  let anonSeq = 0;
  let contentSeq = 0;
  let roundOpen = false;
  let lastText = "";
  let lastError = "";

  function reportError(message) {
    const safe = safeLogText(message).trim().slice(0, 8192) || "Codex failed without an error message.";
    if (safe === lastError) {
      return;
    }
    lastError = safe;
    writeStderr(`Codex error: ${safe}\n`);
    activity.notice("codex_error", safe);
  }

  function itemType(item) {
    return String((item && (item.type || item.item_type)) || "").toLowerCase();
  }

  function itemID(item, creating) {
    const id = item && item.id != null ? String(item.id).trim() : "";
    if (id) {
      return id;
    }
    if (creating) {
      const generated = `anon-${anonSeq}`;
      anonSeq += 1;
      anonQueue.push(generated);
      return generated;
    }
    return anonQueue.shift() || "";
  }

  function contentID(item, type, creating) {
    const id = item && item.id != null ? String(item.id).trim() : "";
    if (id) {
      return id;
    }
    const queue = anonymousContent.get(type) || [];
    if (creating) {
      const generated = `${type}-${contentSeq}`;
      contentSeq += 1;
      queue.push(generated);
      anonymousContent.set(type, queue);
      return generated;
    }
    return queue.shift() || `${type}-${contentSeq++}`;
  }

  function appendContentDelta(contentKind, id, text, options = {}) {
    if (typeof text !== "string" || !text) {
      return;
    }
    const safe = safeLogText(redactSubscription(text, options));
    const previous = contentText.get(id) || "";
    const delta = safe.startsWith(previous)
      ? safe.slice(previous.length)
      : safe;
    activity.appendContent(contentKind, id, delta);
    contentText.set(id, safe);
  }

  function rememberTool(item) {
    const id = itemID(item, true);
    if (open.has(id)) {
      return id;
    }
    open.set(id, {
      kind: normalizeCodexToolKind(item),
      text: toolTextForItem(item),
      startedAt: Date.now(),
      emitted: false,
    });
    activity.startTool({
      id,
      kind: normalizeCodexToolKind(item),
      name: String(item.tool || item.name || item.type || "tool"),
      input: toolTextForItem(item),
    });
    return id;
  }

  function emitStart(id) {
    const tracked = open.get(id);
    if (!tracked || tracked.emitted) {
      return;
    }
    tracked.emitted = true;
    writeLiveLogRecord(
      tracker.stampToolStart({
        type: "tool_start",
        id,
        kind: tracked.kind,
        text: tracked.text,
        started_at: tracked.startedAt,
      }),
    );
  }

  function completeTool(item) {
    let id = itemID(item, false);
    if (!open.has(id)) {
      id = rememberTool(item);
    }
    emitStart(id);
    const output =
      item.aggregated_output ||
      item.output ||
      (item.error && item.error.message) ||
      failedToolResultText(item) ||
      "";
    if (typeof output === "string" && output.trim()) {
      if (activity.enabled) {
        activity.appendToolOutput(
          id,
          output.replace(/\s+$/, ""),
          toolFailed(item) ? "stderr" : "stdout",
        );
      } else {
        writeStdout(`${output.replace(/\s+$/, "")}\n`);
      }
    }
    const tracked = open.get(id) || {
      kind: normalizeCodexToolKind(item),
      startedAt: Date.now(),
    };
    open.delete(id);
    const anonIndex = anonQueue.indexOf(id);
    if (anonIndex >= 0) {
      anonQueue.splice(anonIndex, 1);
    }
    writeLiveLogRecord(
      tracker.stampToolEnd({
        type: "tool_end",
        id,
        kind: tracked.kind,
        status: toolFailed(item) ? "failed" : "passed",
        duration_ms: Math.max(0, Date.now() - tracked.startedAt),
      }),
    );
    activity.endTool(id, {
      status: codexToolStatus(item),
      exitCode: item.exit_code,
      signal: item.signal,
    });
  }

  return {
    handleEvent(event) {
      if (event.type === "error" || event.type === "turn.failed") {
        reportError(event.message || (event.error && event.error.message) || "");
        roundOpen = false;
        return;
      }
      if (event.type === "turn.completed" || event.type === "result") {
        tracker.updateCurrentUsage(extractUsage(event));
        roundOpen = false;
        return;
      }
      const item = event.item;
      if (!item || typeof item !== "object") {
        return;
      }
      const type = itemType(item);
      if (isMessageItem(type)) {
        const contentKind = type === "reasoning" ? "reasoning" : "assistant";
        const id = contentID(item, type, event.type === "item.started");
        if (event.type === "item.started") {
          activity.startContent(contentKind, id);
          const initialText = item.text || item.result || "";
          appendContentDelta(contentKind, id, initialText, { incomplete: true });
          return;
        }
        if (event.type === "item.completed") {
          const text = item.text || item.result || "";
          if (!roundOpen) {
            tracker.beginTurn(item.usage || event.usage, {
              message:
                typeof text === "string" && type !== "reasoning"
                  ? text
                  : undefined,
            });
          }
          roundOpen = false;
          activity.startContent(contentKind, id);
          appendContentDelta(contentKind, id, text);
          activity.endContent(id);
          contentText.delete(id);
          if (typeof text === "string" && text.trim() && type !== "reasoning") {
            lastText = text.replace(/\s+$/, "");
            if (!activity.enabled) {
              writeStdout(`${lastText}\n`);
            }
          }
        }
        return;
      }
      if (!isToolItem(type)) {
        return;
      }
      if (event.type === "item.started") {
        if (!roundOpen) {
          tracker.beginTurn(item.usage || event.usage);
          roundOpen = true;
        }
        const id = rememberTool(item);
        emitStart(id);
        return;
      }
      if (event.type === "item.completed") {
        if (!roundOpen) {
          tracker.beginTurn(item.usage || event.usage);
          roundOpen = true;
        }
        completeTool(item);
      }
    },
    flush(failed) {
      for (const id of [...open.keys()]) {
        emitStart(id);
        const tracked = open.get(id);
        if (!tracked) {
          continue;
        }
        writeLiveLogRecord(
          tracker.stampToolEnd({
            type: "tool_end",
            id,
            kind: tracked.kind,
            status: failed ? "failed" : "passed",
            duration_ms: Math.max(0, Date.now() - tracked.startedAt),
          }),
        );
        activity.endTool(id, { status: failed ? "failed" : "interrupted" });
        open.delete(id);
      }
    },
    lastText() {
      return lastText;
    },
    errorMessage() {
      return lastError;
    },
    reportError,
  };
}

function isMessageItem(type) {
  return (
    type === "agent_message" ||
    type === "assistant_message" ||
    type === "reasoning"
  );
}

function isToolItem(type) {
  return (
    type === "command_execution" ||
    type === "file_change" ||
    type === "mcp_tool_call" ||
    type === "web_search" ||
    type === "tool_call" ||
    type === "bash" ||
    type === "read" ||
    type === "edit" ||
    type === "write"
  );
}

function normalizeCodexToolKind(item) {
  const type = String(item.type || item.item_type || "").toLowerCase();
  if (type === "command_execution" || type === "bash") {
    return "bash";
  }
  if (type === "file_change") {
    return fileChangeKind(item);
  }
  if (type === "mcp_tool_call") {
    return String(item.tool || item.name || "mcp").toLowerCase();
  }
  if (type === "web_search") {
    return "web_search";
  }
  if (type === "read" || type === "edit" || type === "write") {
    return type;
  }
  return type || "tool";
}

function fileChangeKind(item) {
  const changes = Array.isArray(item.changes) ? item.changes : [];
  const kinds = changes.map((change) =>
    String((change && change.kind) || "").toLowerCase(),
  );
  if (kinds.includes("add") && !kinds.includes("update")) {
    return "write";
  }
  if (kinds.length > 0) {
    return "edit";
  }
  return "edit";
}

function toolTextForItem(item) {
  const type = String(item.type || item.item_type || "").toLowerCase();
  if (type === "command_execution" || type === "bash") {
    return stripBashLc(String(item.command || item.text || "bash"));
  }
  if (type === "file_change") {
    const changes = Array.isArray(item.changes) ? item.changes : [];
    const paths = changes
      .map((change) => change && change.path)
      .filter(Boolean);
    return paths.length > 0
      ? paths.map(String).join("\n")
      : String(item.path || "file");
  }
  if (type === "mcp_tool_call") {
    return [item.server, item.tool || item.name || "mcp"]
      .filter(Boolean)
      .join(".");
  }
  return String(
    item.command || item.path || item.query || item.name || type || "tool",
  );
}

function stripBashLc(command) {
  return command.replace(/^bash\s+-lc\s+/, "").trim() || command;
}

function toolFailed(item) {
  const status = String(item.status || "").toLowerCase();
  if (
    [
      "failed",
      "declined",
      "cancelled",
      "canceled",
      "timed_out",
      "timeout",
      "interrupted",
    ].includes(status)
  ) {
    return true;
  }
  const exit = Number(item.exit_code);
  return Number.isFinite(exit) && exit !== 0;
}

function failedToolResultText(item) {
  if (!toolFailed(item)) {
    return "";
  }
  const content = item.result && Array.isArray(item.result.content) ? item.result.content : [];
  return content
    .filter((part) => part && part.type === "text" && typeof part.text === "string")
    .map((part) => part.text)
    .join("\n");
}

function codexToolStatus(item) {
  const status = String((item && item.status) || "").toLowerCase();
  if (status === "cancelled" || status === "canceled") return "cancelled";
  if (status === "timed_out" || status === "timeout") return "timed_out";
  if (status === "interrupted") return "interrupted";
  return toolFailed(item) ? "failed" : "passed";
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
    parts.push(
      Number.isFinite(cost)
        ? `$${cost.toFixed(4)}`
        : `$${event.total_cost_usd}`,
    );
  }
  if (event && event.duration_ms != null) {
    const ms = Number(event.duration_ms);
    if (Number.isFinite(ms)) {
      parts.push(`${(ms / 1000).toFixed(1)}s`);
    }
  }
  writeStdout(`${parts.join(" · ")}\n`);
}

function formatCodexJsonLines(rawLines) {
  const formatter = createCodexFormatter(loadTurnTelemetry());
  for (const line of rawLines) {
    const trimmed = String(line).trim();
    if (!trimmed) {
      continue;
    }
    try {
      formatter.handleEvent(JSON.parse(trimmed));
    } catch (_err) {
      writeStdout(`${trimmed}\n`);
    }
  }
  formatter.flush();
}

if (require.main === module) {
  main();
}

module.exports = {
  formatCodexJsonLines,
  formatTurnResult,
  createCodexFormatter,
  normalizeCodexToolKind,
  codexExecArgs,
  codexExecEnv,
  thinkingArgs,
  codexSessionForPrompt,
  codexSessionIDFromEvent,
  planningEnabled,
  planningSystemPrompt,
  planningAnalysisEnabled,
  workspaceMCPConfigOverrides,
};
