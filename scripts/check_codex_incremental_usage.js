"use strict";

// Exercise the real pinned CLI and the task wrapper without provider credentials.
// The local endpoint supplies deterministic provider measurements. This check
// verifies recording and delivery; it does not verify a live ChatGPT account.
const assert = require("node:assert/strict");
const { spawn, execFileSync } = require("node:child_process");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");

const repository = path.resolve(__dirname, "..");
const runner = path.join(repository, "pkg", "components", "runner");

function sendEvent(response, event) {
  response.write(`event: ${event.type}\ndata: ${JSON.stringify(event)}\n\n`);
}

function controlledProvider() {
  let responses = 0;
  const server = http.createServer((request, response) => {
    request.resume();
    if (request.url === "/api/v1/runner/planning-sessions/agent-messages") {
      response.writeHead(200, { "content-type": "application/json" });
      response.end("{}");
      return;
    }
    if (request.method !== "POST" || request.url !== "/v1/responses") {
      response.writeHead(404);
      response.end();
      return;
    }
    const number = ++responses;
    const position = number % 2 === 1 ? 1 : 2;
    const id = `controlled-response-${number}`;
    response.writeHead(200, { "content-type": "text/event-stream" });
    sendEvent(response, { type: "response.created", response: { id } });
    const item = position === 1
      ? { type: "function_call", call_id: `wait-${number}`, name: "exec_command",
          arguments: JSON.stringify({ cmd: "sleep 3", yield_time_ms: 10000 }) }
      : { type: "message", id: `answer-${number}`, role: "assistant",
          content: [{ type: "output_text", text: "Controlled acceptance complete." }] };
    sendEvent(response, { type: "response.output_item.done", item });
    sendEvent(response, {
      type: "response.completed",
      response: { id, usage: {
        input_tokens: 100 * position,
        input_tokens_details: { cached_tokens: 20 * position },
        output_tokens: 30 * position,
        output_tokens_details: { reasoning_tokens: 10 * position },
        total_tokens: 130 * position,
      } },
    });
    response.end();
  });
  return server;
}

function prepareTask(directory, baseURL) {
  fs.mkdirSync(directory, { recursive: true });
  for (const file of ["turn_telemetry.js", "llm_usage.js", "activity_stream.js",
    "analysis_protocol.js", "analysis_protocol_review.md", "agent_message.js"]) {
    fs.copyFileSync(path.join(runner, file), path.join(directory, file));
  }
  for (const file of ["run.js", "rollout_observer.js"]) {
    fs.copyFileSync(path.join(runner, "codex", file), path.join(directory, file));
  }
  const home = path.join(directory, "codex-home");
  fs.mkdirSync(home);
  fs.writeFileSync(path.join(home, "config.toml"), [
    'model_provider = "controlled"',
    "[model_providers.controlled]",
    'name = "Controlled acceptance"',
    `base_url = "${baseURL}/v1"`,
    'wire_api = "responses"',
    'env_key = "CODEX_API_KEY"',
    "supports_websockets = false",
    "",
  ].join("\n"));
  fs.writeFileSync(path.join(directory, "prompt_count"), "0\n");
  fs.writeFileSync(path.join(directory, "prompt.txt"), "Perform the controlled check.\n");
  return home;
}

async function runPrompt(directory, env) {
  const child = spawn("node", [path.join(directory, "run.js"),
    path.join(directory, "prompt.txt"), "gpt-5.4"], {
    cwd: directory, env, stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "";
  let errors = "";
  let observedWhileActive = false;
  const records = [];
  const timer = setTimeout(() => child.kill("SIGTERM"), 30000);
  child.stdout.on("data", (chunk) => {
    output += chunk;
    let newline;
    while ((newline = output.indexOf("\n")) >= 0) {
      const line = output.slice(0, newline);
      output = output.slice(newline + 1);
      try {
        const record = JSON.parse(line);
        if (record.type !== "turn") continue;
        records.push(record);
        if (record.usage_state === "measured" && record.usage?.total_tokens === 130 && child.exitCode === null) {
          observedWhileActive = true;
        }
      } catch (_) { /* Normal human-readable task output is not telemetry. */ }
    }
  });
  child.stderr.on("data", (chunk) => { errors = `${errors}${chunk}`.slice(-4000); });
  const code = await new Promise((resolve, reject) => {
    child.once("error", reject);
    child.once("close", resolve);
  });
  clearTimeout(timer);
  assert.equal(code, 0, `Codex wrapper failed: ${errors}`);
  assert.ok(observedWhileActive, "No measured snapshot arrived while Codex was active");
  const result = JSON.parse(fs.readFileSync(env.SUPERPLANE_RESULT_FILE, "utf8"));
  assert.equal(result.usage_state, "measured");
  return { result, records, observedWhileActive };
}

async function main() {
  const version = execFileSync("codex", ["--version"], { encoding: "utf8" }).trim();
  assert.equal(version, "codex-cli 0.163.0");
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "byos-cli-acceptance-"));
  const server = controlledProvider();
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const baseURL = `http://127.0.0.1:${server.address().port}`;
  const evidence = { version, provider: "controlled local fixture", modes: [] };
  try {
    for (const mode of ["execution", "planning"]) {
      const taskDirectory = path.join(directory, mode);
      const home = prepareTask(taskDirectory, baseURL);
      const env = {
        PATH: process.env.PATH,
        CODEX_HOME: home,
        CODEX_API_KEY: "controlled-fixture-key",
        SUPERPLANE_TASK_DIR: taskDirectory,
        SUPERPLANE_RESULT_FILE: path.join(taskDirectory, "result.json"),
        ...(mode === "planning" ? {
          SUPERPLANE_PLANNING_SESSION_KIND: "work_order_analysis",
          SUPERPLANE_PLANNING_SESSION_ID: "controlled-planning-session",
          SUPERPLANE_BASE_URL: baseURL,
          SUPERPLANE_RUN_TOKEN: "controlled-run-token",
        } : {}),
      };
      const first = await runPrompt(taskDirectory, env);
      assert.deepEqual(first.result.usage, {
        input_tokens: 240, output_tokens: 60, cache_read_input_tokens: 60,
        cache_creation_input_tokens: 0, reasoning_tokens: 30, total_tokens: 390,
      });
      const second = await runPrompt(taskDirectory, env);
      assert.equal(second.result.usage.total_tokens, 780, "Resumed history or final totals were counted twice");
      evidence.modes.push({ mode, incremental_while_active: true,
        first_prompt: first.result.usage, after_resume: second.result.usage,
        usage_state: second.result.usage_state });
    }
    process.stdout.write(`${JSON.stringify(evidence, null, 2)}\n`);
  } finally {
    await new Promise((resolve) => server.close(resolve));
    fs.rmSync(directory, { recursive: true, force: true });
  }
}

main().catch((error) => { process.stderr.write(`${error.message}\n`); process.exitCode = 1; });
