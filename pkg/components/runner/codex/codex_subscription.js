const fs = require("fs");
const path = require("path");
const { spawn } = require("child_process");
const readline = require("readline");
const { stripVTControlCharacters } = require("node:util");

const CHECKPOINT_TIMEOUT_MS = 30000;

function subscriptionEnabled(env = process.env) {
  return Boolean(String(env.SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN || "").trim());
}

function codexHome(env = process.env) {
  const taskDir = env.SUPERPLANE_TASK_DIR;
  if (!taskDir) {
    throw new Error("SUPERPLANE_TASK_DIR is required");
  }
  return env.CODEX_HOME || path.join(taskDir, "codex-home");
}

function authPath(env = process.env) {
  return path.join(codexHome(env), "auth.json");
}

function subscriptionChildEnv(env = process.env) {
  if (!subscriptionEnabled(env)) {
    return env;
  }
  const next = { ...env };
  delete next.OPENAI_API_KEY;
  delete next.OPENAI_BASE_URL;
  next.CODEX_HOME = codexHome(env);
  return next;
}

function redactSecrets(text, env = process.env) {
  let output = String(text ?? "");
  const bundle = readBundle(env);
  if (!bundle) {
    return output;
  }
  for (const secret of secretValues(bundle)) {
    if (secret) {
      output = output.split(secret).join("[redacted]");
    }
  }
  return output;
}

function secretValues(value, found = []) {
  if (!value || typeof value !== "object") {
    return found;
  }
  for (const [key, child] of Object.entries(value)) {
    if (
      typeof child === "string" &&
      /token|secret|password/i.test(key) &&
      child.length > 8
    ) {
      found.push(child);
      continue;
    }
    secretValues(child, found);
  }
  return found;
}

function readBundle(env = process.env) {
  try {
    return JSON.parse(fs.readFileSync(authPath(env), "utf8"));
  } catch (_err) {
    return null;
  }
}

async function requestJSON(env, method, pathname, body) {
  const base = String(env.SUPERPLANE_BASE_URL || "").replace(/\/$/, "");
  const token = String(env.SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN || "").trim();
  if (!base || !token) {
    throw new Error("ChatGPT subscription endpoint is not configured");
  }
  const response = await fetch(`${base}${pathname}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(CHECKPOINT_TIMEOUT_MS),
  });
  if (!response.ok) {
    throw new Error(`ChatGPT subscription request failed (${response.status})`);
  }
  return response.json();
}

async function restore(env = process.env, options = {}) {
  const payload = await requestJSON(
    env,
    "GET",
    "/api/v1/runner/codex-subscription/credentials",
  );
  const bundle = payload && payload.bundle;
  if (!bundle || typeof bundle !== "object") {
    throw new Error("ChatGPT subscription credentials are missing");
  }
  const home = codexHome(env);
  fs.mkdirSync(home, { recursive: true, mode: 0o700 });
  fs.chmodSync(home, 0o700);
  const file = authPath(env);
  fs.writeFileSync(file, `${JSON.stringify(bundle)}\n`, { mode: 0o600 });
  fs.chmodSync(file, 0o600);
  // A run keeps the lease secret in auth.json. Only a catalog refresh
  // sends the real bundle back to SuperPlane.
  if (options.final) {
    return checkpoint({ env, final: true });
  }
}

async function checkpoint(options = {}) {
  const env = options.env || process.env;
  // Read limits before the bundle: app-server can renew the saved credentials.
  const usage = await readUsage(env).catch(() => {
    process.stderr.write(
      "Codex usage is unavailable. Credentials will still be saved.\n",
    );
    return undefined;
  });
  const bundle = readBundle(env);
  if (!bundle) {
    throw new Error("ChatGPT subscription credentials were not found");
  }
  const payload = await requestJSON(
    env,
    "POST",
    "/api/v1/runner/codex-subscription/checkpoint",
    {
      bundle,
      version: options.version,
      final: Boolean(options.final),
      usage,
    },
  );
  if (!payload || payload.accepted !== true) {
    throw new Error("ChatGPT subscription credentials were not saved");
  }
  return payload.version;
}

function readUsage(env) {
  return new Promise((resolve, reject) => {
    const child = spawn("codex", ["app-server", "--listen", "stdio://"], {
      env: subscriptionChildEnv(env),
      stdio: ["pipe", "pipe", "ignore"],
    });
    const lines = readline.createInterface({ input: child.stdout });
    const timeout = setTimeout(
      () => finish(new Error("Codex usage request timed out")),
      10000,
    );
    let settled = false;
    let limits;
    let models;
    const modelCatalog = [];
    function finish(error, usage) {
      if (settled) return;
      settled = true;
      clearTimeout(timeout);
      lines.close();
      child.stdin.destroy();
      child.kill("SIGKILL");
      if (error) reject(error);
      else resolve(usage);
    }
    function send(message) {
      child.stdin.write(`${JSON.stringify(message)}\n`);
    }
    function finishSnapshot() {
      if (limits !== undefined && models !== undefined) {
        finish(null, { limits, models });
      }
    }
    child.on("error", () => finish(new Error("Codex usage process failed")));
    child.stdin.on("error", () =>
      finish(new Error("Codex usage connection failed")),
    );
    child.on("close", () => finish(new Error("Codex usage process stopped")));
    lines.on("line", (line) => {
      try {
        const message = JSON.parse(line);
        if (message.id !== 1 && message.id !== 2 && message.id !== 3) return;
        if (message.error)
          return finish(new Error("Codex usage request failed"));
        if (message.id === 1) {
          send({ method: "initialized" });
          send({ id: 2, method: "account/rateLimits/read" });
          send({ id: 3, method: "model/list", params: { limit: 100 } });
          return;
        }
        if (message.id === 3) {
          const result = message.result || {};
          const page = Array.isArray(result.data)
            ? result.data.map((model) => ({
                id: model.id,
                model: model.model,
                displayName: model.displayName,
                defaultReasoningEffort: model.defaultReasoningEffort,
                supportedReasoningEfforts: model.supportedReasoningEfforts,
                isDefault: model.isDefault,
              }))
            : [];
          modelCatalog.push(...page);
          if (result.nextCursor) {
            send({ id: 3, method: "model/list", params: { cursor: result.nextCursor, limit: 100 } });
            return;
          }
          models = modelCatalog;
          finishSnapshot();
          return;
        }
        const result = message.result;
        const buckets = Object.entries(result.rateLimitsByLimitId || {});
        limits = buckets.length
          ? buckets.map(([id, limit]) => ({
              ...limit,
              limitId: limit.limitId || id,
            }))
          : result.rateLimits
            ? [
                {
                  ...result.rateLimits,
                  limitId: result.rateLimits.limitId || "codex",
                },
              ]
            : [];
        if (!limits.length)
          return finish(new Error("Codex usage was not returned"));
        finishSnapshot();
      } catch (_err) {
        finish(new Error("Codex usage response is invalid"));
      }
    });
    send({
      id: 1,
      method: "initialize",
      params: {
        clientInfo: {
          name: "superplane",
          title: "SuperPlane",
          version: "1.0.0",
        },
      },
    });
  });
}

function parseDeviceAuth(text) {
  const source = stripVTControlCharacters(String(text || ""));
  const urlMatch = source.match(/https:\/\/[^\s)]+/);
  const codeMatch = source.match(/\b([A-Z0-9]{4,8}(?:-[A-Z0-9]{4,8})+)\b/);
  return {
    verificationUrl: urlMatch ? urlMatch[0] : "",
    verificationCode: codeMatch ? codeMatch[1] : "",
  };
}

async function reportUsage(env = process.env) {
  let usage;
  try {
    usage = await readUsage(env);
  } catch (_err) {
    process.stdout.write("Codex usage is unavailable.\n");
    return;
  }
  await requestJSON(env, "POST", "/api/v1/runner/codex-subscription/checkpoint", { usage });
}

async function login(env = process.env) {
  const home = codexHome(env);
  fs.mkdirSync(home, { recursive: true, mode: 0o700 });
  const childEnv = subscriptionChildEnv(env);
  const child = spawn("codex", ["login", "--device-auth"], {
    env: childEnv,
    stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "";
  let progressPost = Promise.resolve();
  const capture = (chunk) => {
    output += chunk.toString();
    const parsed = parseDeviceAuth(output);
    if (parsed.verificationUrl && parsed.verificationCode && !capture.posted) {
      capture.posted = true;
      progressPost = requestJSON(
        env,
        "POST",
        "/api/v1/runner/codex-subscription/progress",
        parsed,
      ).catch((err) => {
        child.kill();
        throw err;
      });
    }
  };
  child.stdout.on("data", capture);
  child.stderr.on("data", capture);
  const exitCode = await new Promise((resolve, reject) => {
    child.on("error", reject);
    child.on("close", (code) => resolve(code == null ? 1 : code));
  });
  await progressPost;
  if (exitCode !== 0) {
    throw new Error("ChatGPT device sign-in failed");
  }
  if (!readBundle(env)) {
    throw new Error("ChatGPT device sign-in did not write credentials");
  }
  await checkpoint({ env, final: true });
  process.stdout.write("ChatGPT sign-in saved\n");
}

async function main(argv = process.argv.slice(2)) {
  const command = argv[0];
  if (command === "restore") {
    await restore();
    process.stdout.write("ChatGPT sign-in restored\n");
    return;
  }
  if (command === "refresh") {
    await restore(process.env, { final: true });
    process.stdout.write("ChatGPT model catalog refreshed\n");
    return;
  }
  if (command === "checkpoint") {
    await checkpoint({ final: argv.includes("--final") });
    process.stdout.write("ChatGPT sign-in saved\n");
    return;
  }
  if (command === "usage") {
    await reportUsage();
    process.stdout.write("ChatGPT usage saved\n");
    return;
  }
  if (command === "login") {
    await login();
    return;
  }
  throw new Error("usage: node codex_subscription.js restore|refresh|checkpoint|usage|login");
}

if (require.main === module) {
  main().catch((err) => {
    process.stderr.write(
      `${err && err.message ? err.message : "ChatGPT subscription failed"}\n`,
    );
    process.exit(1);
  });
}

module.exports = {
  subscriptionEnabled,
  subscriptionChildEnv,
  redactSecrets,
  restore,
  checkpoint,
  reportUsage,
  parseDeviceAuth,
  login,
};
