#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { spawn } = require("child_process");

const SESSION_FILE = "muse_session";

function resolveTaskPath(value, taskDir) {
  return String(value || "")
    .replace(/\$\{SUPERPLANE_TASK_DIR\}/g, taskDir)
    .replace(/\$SUPERPLANE_TASK_DIR/g, taskDir);
}

function buildMuseSettings(taskDir, env = process.env) {
  const servers = {};
  const configured = resolveTaskPath(
    env.SUPERPLANE_WORKSPACE_MCP_CONFIG,
    taskDir,
  );
  const workspaceConfig =
    configured || path.join(taskDir, "workspace_mcp.json");
  if (fs.existsSync(workspaceConfig)) {
    try {
      const parsed = JSON.parse(fs.readFileSync(workspaceConfig, "utf8"));
      for (const server of Array.isArray(parsed.servers)
        ? parsed.servers
        : []) {
        const name = String((server && server.name) || "").trim();
        const url = String((server && server.url) || "").trim();
        if (!name || !url) continue;
        servers[name] = {
          transport: "streamable_http",
          url,
          headers:
            server.headers && typeof server.headers === "object"
              ? server.headers
              : {},
          enabled: true,
          mode: "required",
        };
      }
    } catch (_err) {
      // Invalid optional workspace configuration must not hide the user task.
    }
  }
  const planningMCP = path.join(taskDir, "planning_session_mcp.js");
  if (env.SUPERPLANE_PLANNING_SESSION_ID && fs.existsSync(planningMCP)) {
    servers.superplane = {
      transport: "stdio",
      command: "node",
      args: [planningMCP],
      env: {},
      enabled: true,
      mode: "required",
    };
  }
  const artifactMCP = path.join(taskDir, "task_artifact_mcp.js");
  if (
    !servers.superplane &&
    env.SUPERPLANE_ARTIFACT_TOKEN &&
    fs.existsSync(artifactMCP)
  ) {
    servers.superplane = {
      transport: "stdio",
      command: "node",
      args: [artifactMCP],
      env: {},
      enabled: true,
      mode: "required",
    };
  }
  return Object.keys(servers).length ? { mcp_servers: servers } : {};
}

function writeMuseSettings(taskDir, env = process.env) {
  const configRoot = path.join(taskDir, "muse-config");
  const settings = buildMuseSettings(taskDir, env);
  if (Object.keys(settings).length) {
    const dir = path.join(configRoot, "muse");
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(
      path.join(dir, "settings.json"),
      `${JSON.stringify(settings)}\n`,
    );
  }
  return configRoot;
}

function museExecArgs({
  model = "",
  thinking = "",
  workspace = "",
  sessionID = "",
  env = process.env,
} = {}) {
  const args = [
    "exec",
    "--json",
    "--disable-approval",
    "--trust-workspace",
    "--user-input-auto-resolve",
    "--workspace",
    workspace || process.cwd(),
  ];
  if (sessionID) {
    args.push("--session-id", sessionID);
  }
  if (model) {
    args.push("--model", String(model).replace(/^(?:custom|meta)\//, ""));
  }
  if (thinking) {
    args.push("--reasoning-effort", thinking);
  }
  const baseURL = String(env.CUSTOM_LLM_BASE_URL || "").trim();
  if (baseURL) {
    args.push("--base-url", baseURL);
  }
  return args;
}

function readSessionID(taskDir) {
  const file = path.join(taskDir, SESSION_FILE);
  return fs.existsSync(file) ? fs.readFileSync(file, "utf8").trim() : "";
}

function writeSessionID(taskDir, sessionID) {
  const value = String(sessionID || "").trim();
  if (value) {
    fs.writeFileSync(path.join(taskDir, SESSION_FILE), `${value}\n`);
  }
}

function usageFromPayload(payload) {
  const usage = payload && payload.usage;
  if (!usage || typeof usage !== "object") {
    return {};
  }
  return {
    input_tokens: Number(usage.promptTokens || usage.input_tokens || 0),
    output_tokens: Number(usage.completionTokens || usage.output_tokens || 0),
    cache_read_input_tokens: Number(
      usage.cachedPromptTokens || usage.cache_read_input_tokens || 0,
    ),
    reasoning_tokens: Number(
      usage.reasoningTokens || usage.reasoning_tokens || 0,
    ),
  };
}

function handleMuseEvent(event, state, taskDir) {
  const sessionID =
    event && event.stream && event.stream.kind === "session"
      ? event.stream.id
      : "";
  if (sessionID) {
    state.sessionID = sessionID;
    writeSessionID(taskDir, sessionID);
  }
  const payload =
    event && event.payload && typeof event.payload === "object"
      ? event.payload
      : {};
  const type = String((event && event.payload_type) || "");
  if (type === "run.output.delta" && payload.text) {
    state.text += String(payload.text);
    process.stdout.write(String(payload.text));
  }
  if (
    type === "run.terminal.completed" ||
    type === "run.terminal.failed" ||
    type === "run.terminal.cancelled"
  ) {
    if (!state.text && payload.text) {
      state.text = String(payload.text);
      process.stdout.write(`${state.text}\n`);
    }
    state.terminal = String(payload.terminal || "");
    state.error = String(payload.reason || payload.error || "");
  }
  const usage = usageFromPayload(payload);
  if (Object.values(usage).some((value) => value > 0)) {
    state.usage = usage;
  }
  if (payload.modelId || payload.model_id) {
    state.model = String(payload.modelId || payload.model_id);
  }
}

async function runPrompt(promptFile, model, thinking, env = process.env) {
  const taskDir = env.SUPERPLANE_TASK_DIR;
  if (!taskDir) {
    throw new Error("SUPERPLANE_TASK_DIR is required");
  }
  const resultFile = env.SUPERPLANE_RESULT_FILE;
  if (!resultFile) {
    throw new Error("SUPERPLANE_RESULT_FILE is required");
  }
  const promptCountPath = path.join(taskDir, "prompt_count");
  const promptCount =
    Number.parseInt(fs.readFileSync(promptCountPath, "utf8").trim(), 10) || 0;
  const sessionID = promptCount > 0 ? readSessionID(taskDir) : "";
  if (promptCount > 0 && !sessionID) {
    throw new Error("Muse session ID is missing for a follow-up prompt");
  }

  const childEnv = { ...env };
  if (!childEnv.META_API_KEY && childEnv.CUSTOM_LLM_API_KEY) {
    childEnv.META_API_KEY = childEnv.CUSTOM_LLM_API_KEY;
  }
  childEnv.XDG_CONFIG_HOME = writeMuseSettings(taskDir, childEnv);
  const args = museExecArgs({
    model,
    thinking,
    workspace: process.cwd(),
    sessionID,
    env: childEnv,
  });
  args.push("--prompt-file", promptFile);
  const child = spawn("muse", args, {
    stdio: ["ignore", "pipe", "pipe"],
    env: childEnv,
  });
  child.stderr.on("data", (chunk) => process.stderr.write(chunk));

  const state = {
    text: "",
    terminal: "",
    error: "",
    usage: {},
    model: model || "",
    sessionID: "",
  };
  const lines = readline.createInterface({
    input: child.stdout,
    crlfDelay: Infinity,
  });
  lines.on("line", (raw) => {
    const line = raw.trim();
    if (!line) return;
    try {
      handleMuseEvent(JSON.parse(line), state, taskDir);
    } catch (_err) {
      process.stdout.write(`${line}\n`);
    }
  });

  const exitCode = await Promise.all([
    new Promise((resolve, reject) => {
      child.on("error", reject);
      child.on("close", (code) => resolve(code == null ? 1 : code));
    }),
    new Promise((resolve) => lines.on("close", resolve)),
  ]).then(([code]) => code);

  if (state.text && !state.text.endsWith("\n")) {
    process.stdout.write("\n");
  }
  const payload = {
    type: "result",
    result: state.text,
    model: state.model || model,
    usage: state.usage,
    session_id: state.sessionID || sessionID,
  };
  if (state.error) payload.error = state.error;
  fs.writeFileSync(resultFile, `${JSON.stringify(payload)}\n`);
  const usageModule = path.join(taskDir, "llm_usage.js");
  if (fs.existsSync(usageModule)) {
    require(usageModule).accumulate(taskDir, payload);
  }
  const planningModule = path.join(taskDir, "planning_session_mcp.js");
  if (env.SUPERPLANE_PLANNING_SESSION_ID && fs.existsSync(planningModule)) {
    await require(planningModule).recordAgentMessage(payload.result);
  }
  fs.writeFileSync(promptCountPath, `${promptCount + 1}\n`);
  return exitCode;
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    process.stderr.write(
      "usage: node run.js <prompt-file> [model] [thinking]\n",
    );
    process.exit(2);
  }
  runPrompt(args[0], args[1] || "", args[2] || "")
    .then((code) => process.exit(code))
    .catch((err) => {
      process.stderr.write(`${err && err.message ? err.message : err}\n`);
      process.exit(1);
    });
}

module.exports = {
  buildMuseSettings,
  handleMuseEvent,
  museExecArgs,
  runPrompt,
  usageFromPayload,
};

if (require.main === module) {
  main();
}
