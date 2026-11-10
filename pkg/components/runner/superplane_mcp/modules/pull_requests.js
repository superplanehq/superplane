"use strict";

const { spawn } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const LEFTOVER_COMMIT_MESSAGE = "fix: implement task via SuperPlane";
const MAX_TITLE_LENGTH = 256;
const NO_RETRY = "A retry will not help. Report this failure to the user.";
const AUTHENTICATION_FAILURE =
  /terminal prompts disabled|could not read (username|password)|authentication failed|permission to .* denied|returned error: 40[13]/i;

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
    child.stdout.setEncoding("utf8").on("data", (chunk) => (stdout += chunk));
    child.stderr.setEncoding("utf8").on("data", (chunk) => (stderr += chunk));
    const timer = setTimeout(() => {
      killProcessGroup(child);
      settle(reject, new Error(`git ${args[0]} did not finish in ${timeoutMs / 1000}s and was stopped`));
    }, timeoutMs);
    child.on("error", (error) => settle(reject, new Error(`git ${args[0]} failed: ${error.message}`)));
    child.on("close", (status) => settle(resolve, { status, stdout: stdout.trim(), stderr: stderr.trim() }));
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

async function gitOutput(directory, args, env, timeoutMs) {
  const result = await git(directory, args, env, timeoutMs);
  if (result.status === 0) return result.stdout;
  if (AUTHENTICATION_FAILURE.test(result.stderr)) {
    throw new Error(
      `git ${args[0]} could not authenticate to the remote. No usable credential is available to the SuperPlane tool. ${NO_RETRY}`,
    );
  }
  throw new Error(`git ${args.join(" ")} failed: ${result.stderr || `exit ${result.status}`}`);
}

// A rejected request does not change on retry. Server errors and timeouts can.
function withRetryAdvice(error) {
  const status = Number(error && error.status);
  if (status >= 400 && status < 500 && status !== 408 && status !== 429) {
    error.message = `${error.message} ${NO_RETRY}`;
  }
  return error;
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
  if (!match) throw new Error(`cannot read the repository from the origin URL ${url}`);
  return `${match[1]}/${match[2]}`;
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
  await gitOutput(directory, ["fetch", "--depth", "1", "origin", `refs/heads/${base}:${remoteRef}`], env, timeouts.fetch);
}

async function commitLeftovers(directory, env, timeouts) {
  await gitOutput(directory, ["add", "-A"], env, timeouts.local);
  if ((await git(directory, ["diff", "--cached", "--quiet"], env, timeouts.local)).status === 0) return;
  await gitOutput(directory, ["commit", "-s", "-m", LEFTOVER_COMMIT_MESSAGE], env, timeouts.local);
}

/**
 * openPullRequest pushes the current branch of one workspace repository and
 * asks SuperPlane to open or update its pull request on this task.
 */
async function openPullRequest(input, client, timeouts = DEFAULT_TIMEOUTS) {
  const env = client.env;
  const title = text(input, "title");
  if (!title) throw new Error("title is required");
  if (title.length > MAX_TITLE_LENGTH) throw new Error(`title must be at most ${MAX_TITLE_LENGTH} characters`);

  const { directory, primary } = repositoryDirectory(input, env);
  const head = await gitOutput(directory, ["rev-parse", "--abbrev-ref", "HEAD"], env, timeouts.local);
  if (!head || head === "HEAD") throw new Error("check out a branch first");
  const base = await baseBranch(input, directory, primary, env, timeouts);
  if (head === base) throw new Error(`create a branch first; do not commit on ${base}`);

  await commitLeftovers(directory, env, timeouts);
  await ensureRemoteBase(directory, base, env, timeouts);
  const ahead = Number(await gitOutput(directory, ["rev-list", "--count", `origin/${base}..HEAD`], env, timeouts.local));
  if (!ahead) throw new Error(`${head} has no commits ahead of ${base}`);
  await gitOutput(directory, ["push", "-u", "origin", "HEAD"], env, timeouts.push);
  const repository = await remoteRepository(directory, env, timeouts);

  try {
    return await client.request("POST", "/api/v1/runner/task/pull-requests", {
      repository,
      head,
      base,
      title,
      body: String((input && input.body) || ""),
      draft: (input && input.draft === true) || String(env.DRAFT_PULL_REQUESTS || "").trim() === "true",
    });
  } catch (error) {
    throw withRetryAdvice(error);
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
        directory: { type: "string", description: "Optional. repo (the default) or context/<owner>/<name>." },
        title: { type: "string", maxLength: MAX_TITLE_LENGTH, description: "The pull request title." },
        body: { type: "string", description: "Optional. The pull request description in Markdown." },
        base: { type: "string", description: "Optional. The base branch. The default is the task branch for repo and the repository default branch for other repositories." },
        draft: { type: "boolean", description: "Optional. Set true only when the task plan explicitly requires a draft pull request. The workspace setting can also require drafts." },
      },
      required: ["title"],
    },
  },
  handler: openPullRequest,
};

const LIST_PULL_REQUESTS = {
  definition: {
    name: "list_pull_requests",
    description: "List the pull requests on this run's task. Each pull request has repository, number, url, title, and state.",
    inputSchema: { type: "object", properties: {} },
  },
  handler: listPullRequests,
};

module.exports = {
  tools: [OPEN_PULL_REQUEST, LIST_PULL_REQUESTS],
  openPullRequest,
  stop: stopGitCommands,
};
