"use strict";

const { spawn } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const LEFTOVER_COMMIT_MESSAGE = "fix: implement task via SuperPlane";
const MAX_TITLE_LENGTH = 256;
const AUTHENTICATION_FAILURE =
  /terminal prompts disabled|could not read (username|password)|authentication failed|returned error: 401/i;
const MAX_OUTPUT_BYTES = 16384;
const MAX_CAUSE_BYTES = 2048;

function redactedCause(cause, env) {
  let safe = String(cause);
  const secrets = ["GITHUB_TOKEN", "GH_TOKEN", "GH_ENTERPRISE_TOKEN", "BITBUCKET_TOKEN", "SUPERPLANE_RUNNER_TOKEN"].map(
    (name) => env[name],
  );
  for (const secret of [...secrets, ...(env.publicationSecrets || [])]) {
    if (secret) {
      for (const value of [
        secret,
        encodeURIComponent(secret),
        Buffer.from(`x-access-token:${secret}`).toString("base64"),
      ]) {
        safe = safe.split(value).join("[REDACTED]");
        // Captured output can stop in the middle of a credential.
        for (let length = value.length - 1; length >= 4; length--) {
          if (safe.endsWith(value.slice(0, length))) {
            safe = safe.slice(0, -length) + "[REDACTED]";
            break;
          }
        }
      }
    }
  }
  safe = safe.replace(/https?:\/\/[^\s/]*@/gi, "https://[REDACTED]@");
  safe = safe.replace(/\b(Bearer|Basic)\s+[A-Za-z0-9+/=_-]+/gi, "$1 [REDACTED]");
  return Buffer.from(safe).subarray(0, MAX_CAUSE_BYTES).toString();
}

// The server's whole tool call must end before the agent stops waiting for
// it, so every git command has its own limit.
const DEFAULT_TIMEOUTS = { local: 30000, fetch: 60000, push: 90000 };

function text(input, name) {
  return String((input && input[name]) || "").trim();
}

// Git must never wait for input. The server shares the agent's terminal
// session, and a credential prompt there stops the whole server process group.
function nonInteractiveEnv(env) {
  const result = { ...env, GIT_TERMINAL_PROMPT: "0", GCM_INTERACTIVE: "never" };
  if (!result.GIT_SSH_COMMAND) result.GIT_SSH_COMMAND = "ssh -o BatchMode=yes";
  return result;
}

// Git commands that are still running. They run outside the process group of
// the server, so stopGitCommands must end them when the server stops.
const running = new Set();

// git runs in its own session, without a controlling terminal. A timeout
// kills the whole process group, so a git-remote-https helper cannot stay
// behind.
function git(directory, args, env, timeoutMs) {
  return new Promise((resolve, reject) => {
    let settled = false;
    const settle = (fn, value) => {
      running.delete(child);
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      fn(value);
    };
    const child = spawn("git", args, {
      cwd: directory,
      env: nonInteractiveEnv(env),
      detached: true,
      stdio: ["ignore", "pipe", "pipe"],
    });
    running.add(child);
    let stdout = "";
    let stderr = "";
    let stdoutTruncated = false;
    child.stdout.setEncoding("utf8").on("data", (chunk) => {
      stdoutTruncated ||= stdout.length + chunk.length > MAX_OUTPUT_BYTES;
      stdout = (stdout + chunk).slice(0, MAX_OUTPUT_BYTES);
    });
    child.stderr.setEncoding("utf8").on("data", (chunk) => (stderr = (stderr + chunk).slice(0, MAX_OUTPUT_BYTES)));
    const timer = setTimeout(() => {
      killProcessGroup(child);
      settle(reject, new Error(`git ${args[0]} did not finish in ${timeoutMs / 1000}s and was stopped`));
    }, timeoutMs);
    child.on("error", (error) =>
      settle(reject, new Error(`git ${args[0]} failed: ${redactedCause(error.message, env)}`)),
    );
    child.on("close", (status) => settle(resolve, { status, stdout: stdout.trim(), stderr: stderr.trim(), stdoutTruncated }));
  });
}

function killProcessGroup(child) {
  try {
    process.kill(-child.pid, "SIGKILL");
  } catch (_error) {
    child.kill("SIGKILL");
  }
}

// stopGitCommands kills every running git command, for example when the task
// is cancelled and the agent stops the server.
function stopGitCommands() {
  for (const child of running) killProcessGroup(child);
  running.clear();
}

async function gitCommand(directory, args, env, timeoutMs) {
  const result = await git(directory, args, env, timeoutMs);
  if (result.status === 0) return result;
  const cause = redactedCause(result.stderr || `exit ${result.status}`, env);
  let classification = "failed";
  if (/permission to .* denied|write access .* not granted/i.test(cause)) classification = "permission denied";
  else if (/returned error: 403/i.test(cause)) classification = "access denied (authentication or permission)";
  else if (AUTHENTICATION_FAILURE.test(cause)) classification = "authentication rejected";
  else if (/hook declined|pre-push|hook.*failed/i.test(cause)) classification = "hook rejected";
  else if (/could not resolve|failed to connect|connection.*(reset|refused)|timed out/i.test(cause))
    classification = "network failure";
  throw new Error(`git ${args[0]} ${classification}: ${cause}`);
}

async function gitOutput(directory, args, env, timeoutMs) {
  const result = await gitCommand(directory, args, env, timeoutMs);
  if (result.stdoutTruncated) throw new Error(`git ${args[0]} output exceeded the publication limit`);
  return result.stdout;
}

// workspaceRoot is the directory that holds repo/ and context/. The checkout
// writes it to task_cwd. Without that file it is the task directory.
function workspaceRoot(env) {
  const taskDir = String(env.SUPERPLANE_TASK_DIR || "").trim();
  if (!taskDir) throw new Error("SUPERPLANE_TASK_DIR is required");
  try {
    const recorded = fs.readFileSync(path.join(taskDir, "task_cwd"), "utf8").trim();
    if (recorded) return fs.realpathSync(recorded);
  } catch (_error) {
    // No task_cwd file. Use the task directory.
  }
  return fs.realpathSync(taskDir);
}

function repositoryDirectory(input, env) {
  const root = workspaceRoot(env);
  const requested = text(input, "directory") || "repo";
  const absolute = path.isAbsolute(requested) ? path.resolve(requested) : path.resolve(root, requested);
  let resolved;
  try {
    resolved = fs.realpathSync(absolute);
  } catch (_error) {
    throw new Error(`directory not found: ${requested}`);
  }
  const relative = path.relative(root, resolved);
  if (relative.startsWith("..") || path.isAbsolute(relative)) {
    throw new Error("directory must be repo/ or a repository under context/");
  }
  if (!fs.existsSync(path.join(resolved, ".git"))) throw new Error(`${requested} is not a git repository`);
  return { directory: resolved, primary: relative === "repo" };
}

// remoteRepository reads owner/name from the origin URL, for example
// https://github.com/acme/web.git.
async function remoteRepository(directory, env, timeouts) {
  const url = await gitOutput(directory, ["remote", "get-url", "origin"], env, timeouts.local);
  const match = url.replace(/\.git$/, "").match(/[/:]([^/:]+)\/([^/]+)$/);
  if (!match) throw new Error("cannot read the repository from the origin URL");
  return `${match[1]}/${match[2]}`;
}

function githubDestination(raw) {
  let url;
  try {
    url = new URL(raw);
  } catch {
    throw new Error("Unsupported GitHub publication destination");
  }
  const repository = url.pathname.replace(/^\//, "").replace(/\.git$/, "");
  if (
    url.protocol !== "https:" ||
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    (url.port && url.port !== "443") ||
    !/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(repository)
  ) {
    throw new Error("Unsupported or credential-bearing GitHub publication destination");
  }
  return { host: url.hostname.toLowerCase(), repository };
}

async function publicationEnvironment(directory, client, timeouts) {
  const env = client.env;
  const fetchURLs = (await gitOutput(directory, ["remote", "get-url", "--all", "origin"], env, timeouts.local)).split(
    "\n",
  );
  const pushURLs = (
    await gitOutput(directory, ["remote", "get-url", "--push", "--all", "origin"], env, timeouts.local)
  ).split("\n");
  const bitbucket = fetchURLs.every((url) => /^https:\/\/bitbucket\.org\//i.test(url));
  const github =
    !bitbucket && (env.GITHUB_TOKEN || env.GH_HOST || fetchURLs.some((url) => /github\.com|\.ghe\.com/.test(url)));
  if (!github)
    return {
      env,
      repository: await remoteRepository(directory, env, timeouts),
    };
  const destination = githubDestination(fetchURLs[0]);
  for (const raw of [...fetchURLs, ...pushURLs]) {
    const candidate = githubDestination(raw);
    if (
      candidate.host !== destination.host ||
      candidate.repository.toLowerCase() !== destination.repository.toLowerCase()
    ) {
      throw new Error("GitHub fetch and push destinations do not match");
    }
  }
  let credential;
  try {
    credential = await client.request(
      "POST",
      "/api/v1/runner/task/publication-credential",
      destination,
      timeouts.request,
    );
  } catch (error) {
    throw new Error(`GitHub publication credential resolution failed: ${redactedCause(error.message, env)}`);
  }
  if (!credential.token || credential.host !== destination.host)
    throw new Error("Invalid GitHub publication credential response");
  const renewed = {
    ...env,
    GITHUB_TOKEN: credential.token,
    GH_TOKEN: credential.token,
    GH_HOST: credential.host,
  };
  Object.defineProperty(renewed, "publicationSecrets", {
    value: [env.GITHUB_TOKEN, env.GH_TOKEN, env.GH_ENTERPRISE_TOKEN],
  });
  if (credential.host !== "github.com") renewed.GH_ENTERPRISE_TOKEN = credential.token;
  return { env: renewed, repository: destination.repository };
}

async function baseBranch(input, directory, primary, env, timeouts) {
  const requested = text(input, "base");
  if (requested) return requested;
  if (primary && String(env.BASE || "").trim()) return String(env.BASE).trim();
  const head = await git(directory, ["symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD"], env, timeouts.local);
  const base = head.status === 0 ? head.stdout.replace(/^origin\//, "") : "";
  if (!base) throw new Error("cannot read the default branch; pass base");
  return base;
}

async function ensureRemoteBase(directory, base, env, timeouts) {
  const remoteRef = `refs/remotes/origin/${base}`;
  const local = await git(directory, ["rev-parse", "--verify", "--quiet", remoteRef], env, timeouts.local);
  if (local.status === 0) return;
  await gitCommand(
    directory,
    ["fetch", "--depth", "1", "origin", `refs/heads/${base}:${remoteRef}`],
    env,
    timeouts.fetch,
  );
}

async function commitLeftovers(directory, env, timeouts) {
  await gitCommand(directory, ["add", "-A"], env, timeouts.local);
  if ((await git(directory, ["diff", "--cached", "--quiet"], env, timeouts.local)).status === 0) return;
  await gitCommand(directory, ["commit", "-s", "-m", LEFTOVER_COMMIT_MESSAGE], env, timeouts.local);
}

/**
 * openPullRequest pushes the current branch of one workspace repository and
 * asks SuperPlane to open or update its pull request on this task.
 */
async function openPullRequest(input, client, timeouts = DEFAULT_TIMEOUTS) {
  // Leave time for the MCP response before its 270s deadline.
  const deadline = Date.now() + 240000;
  const limits = { ...timeouts, request: 60000 };
  timeouts = {};
  for (const [operation, limit] of Object.entries(limits)) {
    Object.defineProperty(timeouts, operation, {
      get: () => Math.max(1, Math.min(limit, deadline - Date.now())),
    });
  }
  let env = client.env;
  const title = text(input, "title");
  if (!title) throw new Error("title is required");
  if (title.length > MAX_TITLE_LENGTH) throw new Error(`title must be at most ${MAX_TITLE_LENGTH} characters`);

  const { directory, primary } = repositoryDirectory(input, env);
  const head = await gitOutput(directory, ["rev-parse", "--abbrev-ref", "HEAD"], env, timeouts.local);
  if (!head || head === "HEAD") throw new Error("check out a branch first");
  const base = await baseBranch(input, directory, primary, env, timeouts);
  if (head === base) throw new Error(`create a branch first; do not commit on ${base}`);

  const publication = await publicationEnvironment(directory, client, timeouts);
  env = publication.env;
  await commitLeftovers(directory, env, timeouts);
  await ensureRemoteBase(directory, base, env, timeouts);
  const ahead = Number(
    await gitOutput(directory, ["rev-list", "--count", `origin/${base}..HEAD`], env, timeouts.local),
  );
  if (!ahead) throw new Error(`${head} has no commits ahead of ${base}`);
  await gitCommand(directory, ["push", "-u", "origin", "HEAD"], env, timeouts.push);
  const repository = publication.repository;

  try {
    return await client.request(
      "POST",
      "/api/v1/runner/task/pull-requests",
      {
        repository,
        head,
        base,
        title,
        body: String((input && input.body) || ""),
        draft: (input && input.draft === true) || String(env.DRAFT_PULL_REQUESTS || "").trim() === "true",
      },
      timeouts.request,
    );
  } catch (error) {
    throw new Error(`Pull request creation failed: ${redactedCause(error.message, env)}`);
  }
}

async function listPullRequests(_input, client) {
  return client.request("GET", "/api/v1/runner/task/pull-requests");
}

const OPEN_PULL_REQUEST = {
  definition: {
    name: "open_pull_request",
    description:
      "Push the current branch of one workspace repository and open its pull request on this task. If the branch already has an open pull request, SuperPlane updates its title and description. Call it once for each repository that you changed. directory is repo for the primary repository or context/<owner>/<name> for another workspace repository. The tool commits uncommitted changes and fails when the branch has no commits ahead of the base branch. SuperPlane adds the task link and closing keywords to the description.",
    inputSchema: {
      type: "object",
      properties: {
        directory: {
          type: "string",
          description: "Optional. repo (the default) or context/<owner>/<name>.",
        },
        title: {
          type: "string",
          maxLength: MAX_TITLE_LENGTH,
          description: "The pull request title.",
        },
        body: {
          type: "string",
          description: "Optional. The pull request description in Markdown.",
        },
        base: {
          type: "string",
          description:
            "Optional. The base branch. The default is the task branch for repo and the repository default branch for other repositories.",
        },
        draft: {
          type: "boolean",
          description:
            "Optional. When the draft setting is false or absent, omit this argument or pass false. Pass true only if the task description or approved specification explicitly requests a draft. Do not infer a draft request from incomplete work, failed checks, repository convention, or agent preference. When DRAFT_PULL_REQUESTS is true, create a draft. Updating an existing pull request does not change its draft state.",
        },
      },
      required: ["title"],
    },
  },
  handler: openPullRequest,
};

const LIST_PULL_REQUESTS = {
  definition: {
    name: "list_pull_requests",
    description:
      "List the pull requests on this run's task. Each pull request has repository, number, url, title, and state.",
    inputSchema: { type: "object", properties: {} },
  },
  handler: listPullRequests,
};

module.exports = {
  tools: [OPEN_PULL_REQUEST, LIST_PULL_REQUESTS],
  openPullRequest,
  stop: stopGitCommands,
};
