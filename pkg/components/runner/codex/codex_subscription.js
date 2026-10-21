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
  return response.json();
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
  const home = codexHome(env);
  fs.mkdirSync(home, { recursive: true, mode: 0o700 });
  fs.chmodSync(home, 0o700);
  const file = authPath(env);
  fs.writeFileSync(file, `${JSON.stringify(bundle)}\n`, { mode: 0o600 });
  fs.chmodSync(file, 0o600);
  // A run keeps the lease secret in auth.json.

}

async function checkpoint(options = {}) {
  const env = options.env || process.env;
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
    },
  );
  if (!payload || payload.accepted !== true) {
    throw new Error("ChatGPT subscription credentials were not saved");
  }
  return payload.version;
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
  if (command === "checkpoint") {
    await checkpoint({ final: argv.includes("--final") });
    process.stdout.write("ChatGPT sign-in saved\n");
    return;
  }
  if (command === "login") {
    await login();
    return;
  }
  throw new Error("usage: node codex_subscription.js restore|checkpoint|login");
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
  parseDeviceAuth,
  login,
};
