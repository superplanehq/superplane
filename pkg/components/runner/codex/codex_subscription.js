const fs = require("fs");
const path = require("path");
const { spawn } = require("child_process");
const readline = require("readline");
const { stripVTControlCharacters } = require("node:util");

const CHECKPOINT_TIMEOUT_MS = 30000;
const credentialRegistries = new Map();
const CREDENTIAL_FIELD_PATTERN = /token|secret|password|api[_-]?key|(?:account|user|organization)[_-]?id|email/i;

class CodexCleanupError extends Error {}

class CodexCredentialsUnavailableError extends Error {
  constructor() {
    super("ChatGPT credentials became unavailable. The run stopped to protect subscription credentials.");
  }
}

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
  observeCredentials(env);
  const next = { ...env };
  delete next.CODEX_API_KEY;
  delete next.OPENAI_API_KEY;
  delete next.OPENAI_BASE_URL;
  next.CODEX_HOME = codexHome(env);
  return next;
}

function redactSecrets(text, env = process.env, options = {}) {
  return createSecretRedactor(env)(text, options);
}

function createSecretRedactor(env = process.env) {
  const secrets = [...observeCredentials(env)].sort((left, right) => right.length - left.length);
  return (text, options = {}) => redactRegisteredSecrets(text, secrets, options);
}

function redactRegisteredSecrets(text, secrets, options) {
  let output = String(text ?? "");
  for (const secret of secrets) {
    output = output.split(secret).join("[redacted]");
  }
  if (options.incomplete) {
    // Keep a possible token prefix out of a streamed content delta. The next
    // cumulative message will either complete it or show that it is safe text.
    let withheld = 0;
    for (const secret of secrets) {
      for (let length = Math.min(output.length, secret.length - 1); length > withheld; length--) {
        if (output.endsWith(secret.slice(0, length))) {
          withheld = length;
          break;
        }
      }
    }
    if (withheld) output = output.slice(0, -withheld);
  }
  return output;
}

function credentialRegistry(env) {
  if (!env.SUPERPLANE_TASK_DIR) return null;
  const file = authPath(env);
  let registry = credentialRegistries.get(file);
  if (!registry) {
    registry = { secrets: new Set(), watcher: null };
    credentialRegistries.set(file, registry);
  }
  const lease = String(env.SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN || "").trim();
  if (lease) registry.secrets.add(lease);
  return registry;
}

function registerCredentials(bundle, env) {
  const registry = credentialRegistry(env);
  if (!registry) return;
  for (const secret of secretValues(bundle)) registry.secrets.add(secret);
}

function setCredentialFailureHandler(handler, env = process.env) {
  const registry = credentialRegistry(env);
  if (!registry) return;
  registry.onFailure = handler;
  registry.required = subscriptionEnabled(env);
}

function observeCredentials(env) {
  const registry = credentialRegistry(env);
  if (!registry) return [];
  const bundle = readBundle(env);
  if (registry.unavailable || (!bundle && (registry.observed || registry.required))) {
    // A replacement can be removed before the watcher reads it. Previously
    // retained values cannot protect output containing that unknown bundle.
    registry.unavailable = true;
    const error = new CodexCredentialsUnavailableError();
    registry.onFailure?.(error);
    throw error;
  }
  const file = authPath(env);
  if (!registry.watcher && fs.existsSync(path.dirname(file))) {
    // Watch the directory because Codex replaces auth.json atomically. Retain
    // every observed value in this output process until the run exits.
    try {
      const watcher = fs.watch(path.dirname(file), { persistent: false }, (_event, name) => {
        if (!name || String(name) === path.basename(file)) readBundle(env);
      });
      registry.watcher = watcher;
      watcher.on("error", () => {
        watcher.close();
        registry.watcher = null;
      });
    } catch (error) {
      // Cleanup can remove the directory between the existence check and watch.
      // Previously observed values remain registered, and output rereads auth.
      if (error.code !== "ENOENT" && error.code !== "ENOTDIR") {
        throw new Error("Could not monitor ChatGPT credentials");
      }
    }
  }
  return registry.secrets;
}

function sanitizeErrorMessage(message) {
  try {
    return redactErrorMessage(message);
  } catch (error) {
    if (error instanceof CodexCredentialsUnavailableError) return error.message;
    throw error;
  }
}

function redactErrorMessage(message) {
  const taskDir = process.env.SUPERPLANE_TASK_DIR || "";
  const activityModule = [
    path.join(taskDir, "activity_stream.js"),
    path.join(__dirname, "..", "activity_stream.js"),
  ].find((file) => fs.existsSync(file));
  const activity = activityModule ? require(activityModule) : null;
  const safe = activity?.redactSensitiveText
    ? activity.redactSensitiveText(message)
    : String(message);
  return redactSecrets(safe).slice(0, 8192);
}

function secretValues(value, found = []) {
  if (!value || typeof value !== "object") {
    return found;
  }
  for (const [key, child] of Object.entries(value)) {
    if (
      typeof child === "string" &&
      CREDENTIAL_FIELD_PATTERN.test(key) &&
      child.length > 0
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
    const bundle = JSON.parse(fs.readFileSync(authPath(env), "utf8"));
    registerCredentials(bundle, env);
    credentialRegistry(env).observed = true;
    return bundle;
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
  const response = await fetchWithRetry(`${base}${pathname}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!response.ok) {
    throw new Error(`ChatGPT subscription request failed (${response.status})`);
  }
  try {
    return await response.json();
  } catch (_error) {
    // JSON parse errors can include plaintext credential response snippets.
    throw new Error("ChatGPT subscription response is invalid");
  }
}

async function fetchWithRetry(url, options) {
  for (let attempt = 0; ; attempt++) {
    try {
      const response = await fetch(url, {
        ...options,
        signal: AbortSignal.timeout(CHECKPOINT_TIMEOUT_MS),
      });
      if (response.ok || (response.status < 500 && response.status !== 429) || attempt === 2) {
        return response;
      }
      await response.body?.cancel();
    } catch (error) {
      if (attempt === 2) throw error;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000 * (attempt + 1)));
  }
}

async function restore(env = process.env) {
  const payload = await requestJSON(
    env,
    "GET",
    "/api/v1/runner/codex-subscription/credentials",
  );
  const bundle = payload && payload.bundle;
  if (!bundle || typeof bundle !== "object") {
    throw new Error("ChatGPT subscription credentials are missing");
  }
  registerCredentials(bundle, env);
  const home = codexHome(env);
  fs.mkdirSync(home, { recursive: true, mode: 0o700 });
  fs.chmodSync(home, 0o700);
  const file = authPath(env);
  fs.writeFileSync(file, `${JSON.stringify(bundle)}\n`, { mode: 0o600 });
  fs.chmodSync(file, 0o600);
}

async function checkpoint(options = {}) {
  const env = options.env || process.env;
  // Read limits before the bundle: app-server can renew the saved credentials.
  const usage = await readUsage(env).catch((error) => {
    if (error instanceof CodexCleanupError) throw error;
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
      detached: true,
    });
    const stopped = new Promise((resolve) => child.once("close", resolve));
    const lines = readline.createInterface({ input: child.stdout });
    const timeout = setTimeout(() => {
      if (limits !== undefined || models !== undefined) {
        finish(null, { limits, models: models ?? undefined });
        return;
      }
      finish(new Error("Codex usage request timed out"));
    }, 10000);
    let settled = false;
    let limits;
    let models;
    let limitsComplete = false;
    let modelsComplete = false;
    const modelCatalog = [];
    const modelCursors = new Set();
    function finish(error, usage) {
      if (settled) return;
      settled = true;
      clearTimeout(timeout);
      lines.close();
      child.stdin.destroy();
      // The npm launcher starts a native child. Stop the entire group so
      // neither process can recreate credentials after cleanup.
      if (child.pid) {
        try {
          process.kill(-child.pid, "SIGKILL");
        } catch (stopError) {
          if (stopError.code !== "ESRCH") {
            reject(new CodexCleanupError("Could not stop the Codex process"));
            return;
          }
        }
      }
      // Codex can renew auth.json. Reap it before callers read the final
      // bundle or remove its home, so no process can write credentials later.
      stopped.then(() => {
        if (error) reject(error);
        else resolve(usage);
      });
    }
    function send(message) {
      child.stdin.write(`${JSON.stringify(message)}\n`);
    }
    function finishSnapshot() {
      if (limitsComplete && modelsComplete) {
        finish(null, { limits, models: models ?? undefined });
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
        if (message.error) {
          if (message.id === 1) return finish(new Error("Codex usage request failed"));
          if (message.id === 2) limitsComplete = true;
          if (message.id === 3) modelsComplete = true;
          finishSnapshot();
          return;
        }
        if (message.id === 1) {
          send({ method: "initialized" });
          send({ id: 2, method: "account/rateLimits/read" });
          send({ id: 3, method: "model/list", params: { limit: 100 } });
          return;
        }
        if (message.id === 3) {
          const result = message.result;
          if (
            !result ||
            !Array.isArray(result.data) ||
            result.data.some((model) => !validCatalogModel(model)) ||
            (result.nextCursor != null && typeof result.nextCursor !== "string") ||
            (result.nextCursor && modelCursors.has(result.nextCursor))
          ) {
            modelsComplete = true;
            finishSnapshot();
            return;
          }
          modelCatalog.push(...result.data.map((model) => ({
            id: model.id,
            model: model.model,
            displayName: model.displayName,
            defaultReasoningEffort: model.defaultReasoningEffort,
            supportedReasoningEfforts: model.supportedReasoningEfforts,
            isDefault: model.isDefault,
          })));
          if (result.nextCursor) {
            modelCursors.add(result.nextCursor);
            send({ id: 3, method: "model/list", params: { cursor: result.nextCursor, limit: 100 } });
            return;
          }
          modelsComplete = true;
          models = modelCatalog;
          finishSnapshot();
          return;
        }
        limitsComplete = true;
        limits = readQuotaLimits(message.result);
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

function readQuotaLimits(result) {
  if (result == null) return [];
  if (typeof result !== "object" || Array.isArray(result)) return undefined;
  const byLimitId = result.rateLimitsByLimitId;
  if (byLimitId != null && (typeof byLimitId !== "object" || Array.isArray(byLimitId))) return undefined;
  const buckets = Object.entries(byLimitId || {});
  if (buckets.length) {
    if (buckets.some(([, limit]) => !limit || typeof limit !== "object" || Array.isArray(limit))) return undefined;
    return buckets.map(([id, limit]) => ({ ...limit, limitId: limit.limitId || id }));
  }
  const limit = result.rateLimits;
  if (limit == null) return [];
  if (typeof limit !== "object" || Array.isArray(limit)) return undefined;
  return [{ ...limit, limitId: limit.limitId || "codex" }];
}

function validCatalogModel(model) {
  if (!model || typeof model.model !== "string" || !model.model.trim()) return false;
  const optionalStrings = [model.id, model.displayName, model.defaultReasoningEffort];
  if (optionalStrings.some((value) => value != null && typeof value !== "string")) return false;
  if (model.isDefault != null && typeof model.isDefault !== "boolean") return false;
  const efforts = model.supportedReasoningEfforts;
  if (efforts == null) return true;
  return Array.isArray(efforts) && efforts.every((effort) =>
    effort &&
    typeof effort === "object" &&
    [effort.reasoningEffort, effort.description].every((value) => value == null || typeof value === "string"),
  );
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
  try {
    await requestJSON(env, "POST", "/api/v1/runner/codex-subscription/checkpoint", { usage });
    process.stdout.write("ChatGPT usage saved\n");
  } catch (_err) {
    process.stderr.write("Codex usage could not be saved. The run can continue.\n");
  }
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
  await finishSetup(env);
  process.stdout.write("ChatGPT sign-in saved\n");
}

async function finishSetup(env = process.env) {
  await checkpoint({ env, final: true });
  completeSetup(env);
}

function completeSetup(env) {
  fs.rmSync(codexHome(env), { recursive: true, force: true });
  if (env.SUPERPLANE_RESULT_FILE) {
    fs.writeFileSync(
      env.SUPERPLANE_RESULT_FILE,
      `${JSON.stringify({ type: "result", is_error: false, codexSubscriptionSetupCleaned: true })}\n`,
    );
  }
}

async function refreshCatalog(env = process.env) {
  await restore(env);
  const usage = await readUsage(env);
  if (!Array.isArray(usage.models)) {
    throw new Error("ChatGPT models could not be loaded");
  }
  const payload = await requestJSON(env, "POST", "/api/v1/runner/codex-subscription/checkpoint", { usage });
  if (payload?.accepted !== true) {
    throw new Error("ChatGPT models were not saved");
  }
  completeSetup(env);
}

async function main(argv = process.argv.slice(2)) {
  const command = argv[0];
  if (command === "restore") {
    await restore();
    process.stdout.write("ChatGPT sign-in restored\n");
    return;
  }
  if (command === "refresh") {
    await refreshCatalog();
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
    const message = sanitizeErrorMessage(
      `ChatGPT subscription ${process.argv[2] || "command"} failed: ${err && err.message ? err.message : "Unknown error"}`,
    );
    process.stderr.write(`${message}\n`);
    const resultFile = process.env.SUPERPLANE_RESULT_FILE;
    if (resultFile) {
      try {
        fs.writeFileSync(
          resultFile,
          `${JSON.stringify({ type: "result", is_error: true, error: message })}\n`,
        );
      } catch (writeError) {
        const writeMessage = sanitizeErrorMessage(
          `ChatGPT subscription error could not be saved: ${writeError.message}`,
        );
        process.stderr.write(`${writeMessage}\n`);
      }
    }
    process.exit(1);
  });
}

module.exports = {
  subscriptionEnabled,
  subscriptionChildEnv,
  redactSecrets,
  createSecretRedactor,
  setCredentialFailureHandler,
  restore,
  checkpoint,
  reportUsage,
  parseDeviceAuth,
  login,
};
