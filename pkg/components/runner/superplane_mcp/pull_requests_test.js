"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { execFileSync, spawn } = require("node:child_process");
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
  return execFileSync("git", args, {
    cwd: directory,
    env,
    encoding: "utf8",
    stdio: "pipe",
  }).trim();
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

test("open_pull_request publishes a leftover commit whose file list exceeds the output limit", async (t) => {
  const { root, env, origin, checkout } = workspace("repo", { BASE: "main" });
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  run(checkout, ["checkout", "-b", "feature"], env);
  for (let index = 0; index < 600; index++) {
    fs.writeFileSync(path.join(checkout, `new-file-with-a-long-name-${index}.txt`), "change\n");
  }
  const client = recordingClient(env);

  await call("open_pull_request", { title: " Add change ", body: "Why", draft: true }, client);

  assert.equal(run(origin, ["rev-parse", "feature"], env), run(checkout, ["rev-parse", "HEAD"], env));
  assert.equal(run(checkout, ["status", "--porcelain"], env), "");
  assert.deepEqual(client.requests, [
    {
      method: "POST",
      path: "/api/v1/runner/task/pull-requests",
      body: {
        repository: "acme/repo",
        head: "feature",
        base: "main",
        title: "Add change",
        body: "Why",
        draft: true,
      },
    },
  ]);
});

test("open_pull_request uses the default branch of a context repository", async () => {
  const { env, checkout } = workspace(path.join("context", "acme", "web"), {
    BASE: "develop",
  });
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
  const { env, checkout } = workspace("repo", {
    BASE: "main",
    DRAFT_PULL_REQUESTS: "true",
  });
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
  await assert.rejects(
    call("open_pull_request", { title: "Outside", directory: "/" }, client),
    /directory must be repo/,
  );
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
  run(
    checkout,
    ["remote", "set-url", "--push", "origin", `http://127.0.0.1:${server.address().port}/acme/repo.git`],
    env,
  );
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

  await assert.rejects(call("open_pull_request", { title: "Change" }, client), /git push authentication rejected/);
  assert.deepEqual(client.requests, []);
});

test("open_pull_request stops a push that the remote never answers", async (t) => {
  const { env, checkout } = workspace("repo", { BASE: "main" });
  featureCommit(checkout, env);
  const sockets = await remoteServer(t, checkout, env, () => {});
  const client = recordingClient(env);

  await assert.rejects(
    openPullRequest({ title: "Change" }, client, {
      local: 30000,
      fetch: 30000,
      push: 500,
    }),
    /git push did not finish in 0.5s and was stopped/,
  );
  await waitFor(() => sockets.size === 0);
  assert.deepEqual(client.requests, []);
});

test("the server stops a running push when the agent closes its input", async (t) => {
  const { env, checkout, root } = workspace("repo", { BASE: "main" });
  featureCommit(checkout, env);
  const sockets = await remoteServer(t, checkout, env, () => {});
  const backend = http.createServer((_request, response) => response.end('{"tools":["open_pull_request"]}'));
  await new Promise((resolve) => backend.listen(0, "127.0.0.1", resolve));
  t.after(() => backend.close());
  const server = spawn(process.execPath, [path.join(__dirname, "server.js")], {
    env: {
      ...env,
      SUPERPLANE_BASE_URL: `http://127.0.0.1:${backend.address().port}`,
      SUPERPLANE_RUNNER_TOKEN: "token",
      SUPERPLANE_TASK_DIR: root,
    },
    stdio: ["pipe", "ignore", "ignore"],
  });
  const exited = new Promise((resolve) => server.on("exit", resolve));
  t.after(() => server.kill("SIGKILL"));

  server.stdin.write(
    `${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "tools/call", params: { name: "open_pull_request", arguments: { title: "Change" } } })}\n`,
  );
  await waitFor(() => sockets.size > 0);
  server.stdin.end();

  await exited;
  await waitFor(() => sockets.size === 0);
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

  assert.deepEqual(client.requests, [
    {
      method: "GET",
      path: "/api/v1/runner/task/pull-requests",
      body: undefined,
    },
  ]);
});

// The https helper sends the canonical GitHub destination to a local smart
// HTTP fixture. Git still runs real fetch, receive-pack, and credential helpers.
async function authenticatedGitHub(t, startupToken, failure) {
  const fixture = workspace("repo", {
    BASE: "main",
    GITHUB_TOKEN: startupToken,
  });
  const { root, checkout, origin, env } = fixture;
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  run(origin, ["config", "http.receivepack", "true"], env);
  const execPath = run(root, ["--exec-path"], env);
  const seen = [];
  const server = http.createServer((request, response) => {
    const auth = request.headers.authorization || "";
    if (auth)
      seen.push(
        Buffer.from(auth.replace(/^Basic /, ""), "base64")
          .toString()
          .split(":")[1],
      );
    if (auth !== `Basic ${Buffer.from("x-access-token:current-token").toString("base64")}`) {
      response.writeHead(401, { "WWW-Authenticate": 'Basic realm="GitHub"' });
      response.end();
      return;
    }
    if (failure === "timeout") return;
    if (failure) {
      response.writeHead(failure);
      response.end(`rejected ${startupToken} current-token`);
      return;
    }
    const url = new URL(request.url, "http://localhost");
    const backend = spawn("git", ["http-backend"], {
      env: {
        ...env,
        GIT_EXEC_PATH: execPath,
        GIT_PROJECT_ROOT: path.join(root, "origins"),
        GIT_HTTP_EXPORT_ALL: "1",
        PATH_INFO: url.pathname,
        QUERY_STRING: url.search.slice(1),
        REQUEST_METHOD: request.method,
        CONTENT_TYPE: request.headers["content-type"] || "",
        REMOTE_USER: "agent",
      },
      stdio: ["pipe", "pipe", "ignore"],
    });
    request.pipe(backend.stdin);
    let headers = Buffer.alloc(0);
    let sent = false;
    backend.stdout.on("data", (chunk) => {
      if (sent) {
        response.write(chunk);
        return;
      }
      headers = Buffer.concat([headers, chunk]);
      const end = headers.indexOf("\r\n\r\n");
      if (end < 0) return;
      const values = {};
      for (const line of headers.subarray(0, end).toString().split("\r\n")) {
        const colon = line.indexOf(":");
        values[line.slice(0, colon)] = line.slice(colon + 1).trim();
      }
      response.writeHead(200, values);
      response.write(headers.subarray(end + 4));
      sent = true;
    });
    backend.on("close", () => response.end());
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  t.after(() => {
    server.closeAllConnections();
    server.close();
  });
  const bin = path.join(root, "bin");
  fs.mkdirSync(bin);
  fs.writeFileSync(
    path.join(bin, "git-remote-https"),
    `#!/bin/sh\nexport GIT_EXEC_PATH="${execPath}"\nexec "${execPath}/git-remote-http" "$1" "http://127.0.0.1:${server.address().port}/acme/repo.git"\n`,
    { mode: 0o755 },
  );
  env.GIT_EXEC_PATH = bin;
  run(
    checkout,
    ["config", "credential.helper", "!f() { printf 'username=x-access-token\\npassword=%s\\n' \"$GITHUB_TOKEN\"; }; f"],
    env,
  );
  run(checkout, ["remote", "set-url", "origin", "https://github.com/acme/repo.git"], env);
  featureCommit(checkout, env);
  run(checkout, ["add", "-A"], env);
  run(checkout, ["commit", "-s", "-m", "existing change"], env);
  run(checkout, ["update-ref", "-d", "refs/remotes/origin/main"], env);
  fs.writeFileSync(path.join(checkout, ".git", "hooks", "pre-push"), `#!/bin/sh\ntouch "${root}/hook-ran"\n`, {
    mode: 0o755,
  });
  return { ...fixture, seen };
}

for (const startup of ["expired-token", "current-token"]) {
  test(`publication refreshes ${startup} before fetch and pushes the existing commit`, async (t) => {
    const { env, checkout, origin, root, seen } = await authenticatedGitHub(t, startup);
    const sha = run(checkout, ["rev-parse", "HEAD"], env);
    // Control: the HTTP fixture rejects the expired startup authorization.
    if (startup === "expired-token") {
      const status = await new Promise((resolve) => {
        const child = spawn("git", ["fetch", "origin", "main"], {
          cwd: checkout,
          env: { ...env, GIT_TERMINAL_PROMPT: "0" },
          stdio: "ignore",
        });
        child.on("close", resolve);
      });
      assert.notEqual(status, 0);
    }
    const client = recordingClient(env);
    client.request = async (method, requestPath, body) => {
      client.requests.push({ method, path: requestPath, body });
      if (requestPath.endsWith("publication-credential")) return { token: "current-token", host: "github.com" };
      return { url: "https://github.com/acme/repo/pull/1" };
    };
    await openPullRequest({ title: "Publish existing work" }, client);
    assert.equal(run(origin, ["rev-parse", "feature"], env), sha);
    assert.equal(run(checkout, ["rev-parse", "HEAD"], env), sha);
    assert.ok(fs.existsSync(path.join(root, "hook-ran")));
    assert.equal(env.GITHUB_TOKEN, startup);
    assert.ok(seen.includes("current-token"));
    assert.equal(client.requests.at(-1).path, "/api/v1/runner/task/pull-requests");
  });
}

test("publication rejects a separate foreign push destination before credential issuance", async () => {
  const { env, checkout } = workspace("repo", {
    BASE: "main",
    GITHUB_TOKEN: "old-token",
  });
  featureCommit(checkout, env);
  run(checkout, ["remote", "set-url", "origin", "https://github.com/acme/repo.git"], env);
  run(checkout, ["remote", "set-url", "--push", "origin", "https://foreign.example/acme/repo.git"], env);
  const client = recordingClient(env);
  await assert.rejects(openPullRequest({ title: "Change" }, client), /destinations do not match/);
  assert.deepEqual(client.requests, []);
});

test("publication fails closed when the remote destination list exceeds the capture limit", async () => {
  const { env, checkout } = workspace("repo", { BASE: "main", GITHUB_TOKEN: "old-token" });
  featureCommit(checkout, env);
  run(checkout, ["remote", "set-url", "origin", "https://github.com/acme/repo.git"], env);
  run(checkout, ["config", "--add", "remote.origin.pushurl", `https://github.com/acme/${"a".repeat(20000)}.git`], env);
  const client = recordingClient(env);
  await assert.rejects(openPullRequest({ title: "Change" }, client), /output exceeded/);
  assert.deepEqual(client.requests, []);
});

for (const [failure, classification] of [
  [401, /authentication rejected/],
  [403, /access denied/],
  ["timeout", /did not finish.*was stopped/],
  ["resolver", /credential resolution failed/],
  ["hook", /hook rejected/],
]) {
  test(`renewed publication reports ${failure} without exposing credentials or changing commits`, async (t) => {
    const { env, checkout } = await authenticatedGitHub(
      t,
      "expired-token",
      typeof failure === "number" || failure === "timeout" ? failure : undefined,
    );
    const sha = run(checkout, ["rev-parse", "HEAD"], env);
    if (failure === "hook") {
      run(checkout, ["update-ref", "refs/remotes/origin/main", "HEAD~1"], env);
      fs.writeFileSync(
        path.join(checkout, ".git", "hooks", "pre-push"),
        `#!/bin/sh\nprintf 'pre-push hook failed: expired-token current-token\\n${"x".repeat(20000)}' >&2\nexit 1\n`,
        { mode: 0o755 },
      );
    }
    const client = recordingClient(env);
    client.request = async (_method, requestPath) => {
      assert.ok(requestPath.endsWith("publication-credential"), "failure must stop before PR creation");
      if (failure === "resolver") throw new Error("resolver rejected expired-token");
      return { token: "current-token", host: "github.com" };
    };
    await assert.rejects(
      openPullRequest({ title: "Change" }, client, {
        local: 30000,
        fetch: 250,
        push: 250,
      }),
      (error) => {
        assert.match(error.message, classification);
        assert.ok(Buffer.byteLength(error.message) < 2300);
        assert.ok(!error.message.includes("expired-token"));
        assert.ok(!error.message.includes("current-token"));
        return true;
      },
    );
    assert.equal(run(checkout, ["rev-parse", "HEAD"], env), sha);
  });
}
