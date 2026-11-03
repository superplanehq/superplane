"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const LEFTOVER_COMMIT_MESSAGE = "fix: implement task via SuperPlane";
const MAX_TITLE_LENGTH = 256;

function text(input, name) {
  return String((input && input[name]) || "").trim();
}

function git(directory, args, env) {
  const result = spawnSync("git", args, { cwd: directory, env, encoding: "utf8" });
  if (result.error) throw new Error(`git ${args[0]} failed: ${result.error.message}`);
  return { status: result.status, stdout: String(result.stdout || "").trim(), stderr: String(result.stderr || "").trim() };
}

function gitOutput(directory, args, env) {
  const result = git(directory, args, env);
  if (result.status !== 0) throw new Error(`git ${args.join(" ")} failed: ${result.stderr || `exit ${result.status}`}`);
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
function remoteRepository(directory, env) {
  const url = gitOutput(directory, ["remote", "get-url", "origin"], env);
  const match = url.replace(/\.git$/, "").match(/[/:]([^/:]+)\/([^/]+)$/);
  if (!match) throw new Error(`cannot read the repository from the origin URL ${url}`);
  return `${match[1]}/${match[2]}`;
}

function baseBranch(input, directory, primary, env) {
  const requested = text(input, "base");
  if (requested) return requested;
  if (primary && String(env.BASE || "").trim()) return String(env.BASE).trim();
  const head = git(directory, ["symbolic-ref", "-q", "--short", "refs/remotes/origin/HEAD"], env);
  const base = head.status === 0 ? head.stdout.replace(/^origin\//, "") : "";
  if (!base) throw new Error("cannot read the default branch; pass base");
  return base;
}

function ensureRemoteBase(directory, base, env) {
  const remoteRef = `refs/remotes/origin/${base}`;
  if (git(directory, ["rev-parse", "--verify", "--quiet", remoteRef], env).status === 0) return;
  gitOutput(directory, ["fetch", "--depth", "1", "origin", `refs/heads/${base}:${remoteRef}`], env);
}

function commitLeftovers(directory, env) {
  gitOutput(directory, ["add", "-A"], env);
  if (git(directory, ["diff", "--cached", "--quiet"], env).status === 0) return;
  gitOutput(directory, ["commit", "-s", "-m", LEFTOVER_COMMIT_MESSAGE], env);
}

/**
 * openPullRequest pushes the current branch of one workspace repository and
 * asks SuperPlane to open or update its pull request on this task.
 */
async function openPullRequest(input, client) {
  const env = client.env;
  const title = text(input, "title");
  if (!title) throw new Error("title is required");
  if (title.length > MAX_TITLE_LENGTH) throw new Error(`title must be at most ${MAX_TITLE_LENGTH} characters`);

  const { directory, primary } = repositoryDirectory(input, env);
  const head = gitOutput(directory, ["rev-parse", "--abbrev-ref", "HEAD"], env);
  if (!head || head === "HEAD") throw new Error("check out a branch first");
  const base = baseBranch(input, directory, primary, env);
  if (head === base) throw new Error(`create a branch first; do not commit on ${base}`);

  commitLeftovers(directory, env);
  ensureRemoteBase(directory, base, env);
  const ahead = Number(gitOutput(directory, ["rev-list", "--count", `origin/${base}..HEAD`], env));
  if (!ahead) throw new Error(`${head} has no commits ahead of ${base}`);
  gitOutput(directory, ["push", "-u", "origin", "HEAD"], env);

  return client.request("POST", "/api/v1/runner/task/pull-requests", {
    repository: remoteRepository(directory, env),
    head,
    base,
    title,
    body: String((input && input.body) || ""),
    draft: (input && input.draft === true) || String(env.DRAFT_PULL_REQUESTS || "").trim() === "true",
  });
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
        draft: { type: "boolean", description: "Optional. Open the pull request as a draft. The workspace setting can also make every pull request a draft." },
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
};
