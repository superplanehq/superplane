"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");

const registry = require("./registry");
const { openPullRequest } = require("./modules/pull_requests");

function call(name, input, client) {
  return registry.findTool(name).handler(input, client);
}

function gitEnv(root, extra = {}) {
  return {
    PATH: process.env.PATH,
    HOME: root,
    GIT_CONFIG_NOSYSTEM: "1",
    GIT_AUTHOR_NAME: "Agent",
    GIT_AUTHOR_EMAIL: "agent@example.com",
    GIT_COMMITTER_NAME: "Agent",
    GIT_COMMITTER_EMAIL: "agent@example.com",
    SUPERPLANE_TASK_DIR: root,
    ...extra,
  };
}

function run(directory, args, env) {
  return execFileSync("git", args, { cwd: directory, env, encoding: "utf8", stdio: "pipe" }).trim();
}

// workspace creates a bare origin with one commit on main and clones it to
// the given directory under the task root.
function workspace(directory, extra) {
  const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "sp-pr-")));
  const env = gitEnv(root, extra);
  const origin = path.join(root, "origins", "acme", `${path.basename(directory)}.git`);
  fs.mkdirSync(origin, { recursive: true });
  run(origin, ["init", "--bare", "--initial-branch=main"], env);
  const seed = path.join(root, "seed");
  run(root, ["clone", origin, seed], env);
  fs.writeFileSync(path.join(seed, "README.md"), "hello\n");
  run(seed, ["add", "-A"], env);
  run(seed, ["commit", "-m", "init"], env);
  run(seed, ["push", "origin", "HEAD:main"], env);
  const checkout = path.join(root, directory);
  fs.mkdirSync(path.dirname(checkout), { recursive: true });
  run(root, ["clone", origin, checkout], env);
  return { root, env, origin, checkout };
}

function recordingClient(env) {
  const requests = [];
  return {
    env,
    requests,
    request: async (method, requestPath, body) => {
      requests.push({ method, path: requestPath, body });
      return { url: "https://github.com/acme/repo/pull/1" };
    },
  };
}

test("open_pull_request commits leftovers, pushes the branch, and opens the pull request", async () => {
  const { env, origin, checkout } = workspace("repo", { BASE: "main" });
  run(checkout, ["checkout", "-b", "feature"], env);
  fs.writeFileSync(path.join(checkout, "change.txt"), "change\n");
  const client = recordingClient(env);

  await call("open_pull_request", { title: " Add change ", body: "Why", draft: true }, client);

  assert.equal(run(origin, ["rev-parse", "feature"], env), run(checkout, ["rev-parse", "HEAD"], env));
  assert.equal(run(checkout, ["status", "--porcelain"], env), "");
  assert.deepEqual(client.requests, [
    {
      method: "POST",
      path: "/api/v1/runner/task/pull-requests",
      body: { repository: "acme/repo", head: "feature", base: "main", title: "Add change", body: "Why", draft: true },
    },
  ]);
});

test("open_pull_request uses the default branch of a context repository", async () => {
  const { env, checkout } = workspace(path.join("context", "acme", "web"), { BASE: "develop" });
  run(checkout, ["checkout", "-b", "feature"], env);
  fs.writeFileSync(path.join(checkout, "change.txt"), "change\n");
  run(checkout, ["add", "-A"], env);
  run(checkout, ["commit", "-m", "change"], env);
  const client = recordingClient(env);

  await call("open_pull_request", { directory: "context/acme/web", title: "Change web" }, client);

  assert.equal(client.requests[0].body.repository, "acme/web");
  assert.equal(client.requests[0].body.base, "main");
  assert.equal(client.requests[0].body.draft, false);
});

test("open_pull_request opens a draft when the workspace asks for drafts", async () => {
  const { env, checkout } = workspace("repo", { BASE: "main", DRAFT_PULL_REQUESTS: "true" });
  run(checkout, ["checkout", "-b", "feature"], env);
  fs.writeFileSync(path.join(checkout, "change.txt"), "change\n");
  const client = recordingClient(env);

  await call("open_pull_request", { title: "Change" }, client);

  assert.equal(client.requests[0].body.draft, true);
});

test("open_pull_request fails before the backend call when nothing changed", async () => {
  const { env, checkout } = workspace("repo", { BASE: "main" });
  const client = recordingClient(env);

  await assert.rejects(call("open_pull_request", { title: "Nothing" }, client), /create a branch first/);
  run(checkout, ["checkout", "-b", "feature"], env);
  await assert.rejects(call("open_pull_request", { title: "Nothing" }, client), /no commits ahead of main/);
  await assert.rejects(call("open_pull_request", { title: "Outside", directory: "/" }, client), /directory must be repo/);
  await assert.rejects(call("open_pull_request", {}, client), /title is required/);
  assert.deepEqual(client.requests, []);
});

// remoteServer serves the push URL of the checkout and tracks open
// connections, so a test can see when git lets go of the remote.
async function remoteServer(t, checkout, env, respond) {
  const sockets = new Set();
  const server = http.createServer(respond);
  server.on("connection", (socket) => {
    sockets.add(socket);
    socket.on("close", () => sockets.delete(socket));
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  t.after(() => {
    server.closeAllConnections();
    server.close();
  });
  run(checkout, ["remote", "set-url", "--push", "origin", `http://127.0.0.1:${server.address().port}/acme/repo.git`], env);
  return sockets;
}

function featureCommit(checkout, env) {
  run(checkout, ["checkout", "-b", "feature"], env);
  fs.writeFileSync(path.join(checkout, "change.txt"), "change\n");
}

test("open_pull_request fails at once when git has no credential for the remote", async (t) => {
  const { env, checkout } = workspace("repo", { BASE: "main" });
  featureCommit(checkout, env);
  await remoteServer(t, checkout, env, (_request, response) => {
    response.writeHead(401, { "WWW-Authenticate": 'Basic realm="GitHub"' });
    response.end();
  });
  const client = recordingClient(env);

  await assert.rejects(
    call("open_pull_request", { title: "Change" }, client),
    /git push could not authenticate to the remote\..*A retry will not help/,
  );
  assert.deepEqual(client.requests, []);
});

test("open_pull_request stops a push that the remote never answers", async (t) => {
  const { env, checkout } = workspace("repo", { BASE: "main" });
  featureCommit(checkout, env);
  const sockets = await remoteServer(t, checkout, env, () => {});
  const client = recordingClient(env);

  await assert.rejects(
    openPullRequest({ title: "Change" }, client, { local: 30000, fetch: 30000, push: 500 }),
    /git push did not finish in 0.5s and was stopped/,
  );
  await waitFor(() => sockets.size === 0);
  assert.deepEqual(client.requests, []);
});

async function waitFor(condition, timeoutMs = 5000) {
  const deadline = Date.now() + timeoutMs;
  while (!condition()) {
    if (Date.now() > deadline) throw new Error("condition was not met in time");
    await new Promise((resolve) => setTimeout(resolve, 20));
  }
}

test("list_pull_requests reads the pull requests of the task", async () => {
  const client = recordingClient({});

  await call("list_pull_requests", {}, client);

  assert.deepEqual(client.requests, [{ method: "GET", path: "/api/v1/runner/task/pull-requests", body: undefined }]);
});
