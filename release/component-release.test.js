#!/usr/bin/env node

const assert = require("node:assert/strict");
const crypto = require("node:crypto");
const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const {
  componentTag,
  createManifest,
  generateReleaseNotes,
  parseVersion,
  publishGitHubRelease,
  stageGitHubRelease,
  validateFinalization,
  verifyManifest,
  verifyPublishedPrerelease,
} = require("./component-release");

function git(repository, args) {
  return execFileSync("git", args, {
    cwd: repository,
    encoding: "utf8",
  }).trim();
}

function createReleaseHistory() {
  const repository = fs.mkdtempSync(
    path.join(os.tmpdir(), "component-release-history-"),
  );
  git(repository, ["init", "--quiet"]);
  git(repository, ["config", "user.name", "Release Test"]);
  git(repository, ["config", "user.email", "release-test@example.com"]);
  fs.writeFileSync(path.join(repository, "README.md"), "initial\n");
  git(repository, ["add", "README.md"]);
  git(repository, ["commit", "--quiet", "-m", "chore: Initial release"]);
  git(repository, ["-c", "tag.gpgSign=false", "tag", "v0.30.0"]);

  fs.mkdirSync(path.join(repository, "cmd", "runner"), { recursive: true });
  fs.writeFileSync(
    path.join(repository, "cmd", "runner", "main.go"),
    "package main\n",
  );
  git(repository, ["add", "cmd/runner/main.go"]);
  git(repository, [
    "commit",
    "--quiet",
    "-m",
    "feat(runner): Add runner release",
  ]);
  const runnerSha = git(repository, ["rev-parse", "HEAD"]);

  fs.mkdirSync(path.join(repository, "release"), { recursive: true });
  fs.writeFileSync(
    path.join(repository, "release", "component-release.js"),
    "// shared release code\n",
  );
  git(repository, ["add", "release/component-release.js"]);
  git(repository, [
    "commit",
    "--quiet",
    "-m",
    "feat(fleet-manager): Add Fleet Manager release",
  ]);
  const fleetManagerSha = git(repository, ["rev-parse", "HEAD"]);
  return {
    repository,
    releaseSha: fleetManagerSha,
    runnerSha,
    fleetManagerSha,
  };
}

function commitSharedReleaseChange(repository, subject) {
  const filePath = path.join(repository, "release", "component-release.js");
  fs.appendFileSync(filePath, `// ${subject}\n`);
  git(repository, ["add", "release/component-release.js"]);
  git(repository, ["commit", "--quiet", "-m", subject]);
  return git(repository, ["rev-parse", "HEAD"]);
}

test("accepts stable and prerelease versions", () => {
  assert.equal(parseVersion("v1.2.3"), "v1.2.3");
  assert.equal(parseVersion("v1.2.3-rc.1"), "v1.2.3-rc.1");
});

test("rejects build metadata and malformed prereleases", () => {
  for (const version of ["1.2.3", "v1.2", "v1.2.3+build.1", "v1.2.3-"]) {
    assert.throws(() => parseVersion(version), /version/);
  }
});

test("validates release candidate finalization pairs", () => {
  assert.equal(validateFinalization("v1.0.0-rc8", "v1.0.0"), "v1.0.0");
  assert.equal(
    validateFinalization("v2.3.4-rc.12", "v2.3.4"),
    "v2.3.4",
  );

  assert.throws(
    () => validateFinalization("v1.0.0", "v1.0.0"),
    /candidate version must be a prerelease/,
  );
  assert.throws(
    () => validateFinalization("v1.0.0-rc8", "v1.0.1"),
    /stable version must be v1.0.0/,
  );
  assert.throws(
    () => validateFinalization("v1.0.0-rc8", "v1.0.0-final"),
    /stable version must not be a prerelease/,
  );
});

test("verifies a published release candidate at the expected commit", async () => {
  const sha = "a".repeat(40);
  let release = {
    draft: false,
    prerelease: true,
    tag_name: "v1.0.0-rc8",
    target_commitish: sha,
  };
  const server = http.createServer((_request, response) => {
    response.writeHead(200, { "Content-Type": "application/json" });
    response.end(JSON.stringify(release));
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  const options = {
    apiURL: `http://127.0.0.1:${address.port}`,
    repository: "superplanehq/superplane",
    component: "superplane",
    version: "v1.0.0-rc8",
    sha,
  };

  try {
    await verifyPublishedPrerelease(options);

    release = { ...release, draft: true };
    await assert.rejects(
      verifyPublishedPrerelease(options),
      /is not a published prerelease/,
    );

    release = { ...release, draft: false, target_commitish: "b".repeat(40) };
    await assert.rejects(
      verifyPublishedPrerelease(options),
      /does not match source SHA/,
    );
  } finally {
    server.close();
  }
});

test("maps each component to its independent tag", () => {
  assert.equal(componentTag("superplane", "v1.2.3"), "v1.2.3");
  assert.equal(componentTag("cli", "v1.2.3"), "cli/v1.2.3");
  assert.equal(componentTag("runner", "v1.2.3"), "runner/v1.2.3");
  assert.equal(
    componentTag("fleet-manager", "v2.0.0-rc.1"),
    "fleet-manager/v2.0.0-rc.1",
  );
  assert.throws(() => componentTag("server", "v1.2.3"), /component/);
});

test("generates SuperPlane notes from unprefixed stable tags", () => {
  const { repository, releaseSha, runnerSha, fleetManagerSha } =
    createReleaseHistory();
  const output = path.join(repository, "superplane-release-notes.md");
  let invocation;

  generateReleaseNotes({
    component: "superplane",
    version: "v1.0.0",
    sha: releaseSha,
    output,
    repository,
    dockerExec: (command, args) => {
      invocation = { command, args };
      return "## v1.0.0\n\n- Add the application release.\n";
    },
  });

  assert.equal(invocation.command, "docker");
  assert.ok(invocation.args.includes(`v0.30.0..${releaseSha}`));
  assert.ok(invocation.args.includes("v1.0.0"));
  assert.ok(invocation.args.includes(runnerSha));
  assert.ok(invocation.args.includes(fleetManagerSha));
});

test("generates stable notes from component release ranges", () => {
  const { repository, releaseSha, runnerSha, fleetManagerSha } =
    createReleaseHistory();
  const firstOutput = path.join(repository, "first-release-notes.md");
  const calls = [];
  const dockerExec = (command, args) => {
    calls.push({ command, args });
    return "## runner/v1.0.0\n\n- Add runner release.\n";
  };

  assert.equal(
    generateReleaseNotes({
      component: "runner",
      version: "v1.0.0",
      sha: releaseSha,
      output: firstOutput,
      repository,
      dockerExec,
    }),
    true,
  );
  assert.equal(
    fs.readFileSync(firstOutput, "utf8"),
    "## runner/v1.0.0\n\n- Add runner release.\n",
  );
  assert.equal(calls[0].command, "docker");
  assert.ok(calls[0].args.includes("v0.30.0.." + releaseSha));
  assert.ok(calls[0].args.includes("cmd/runner/**"));
  assert.ok(calls[0].args.includes("runner/v1.0.0"));
  assert.ok(calls[0].args.includes(fleetManagerSha));
  assert.equal(calls[0].args.includes(runnerSha), false);

  git(repository, [
    "-c",
    "tag.gpgSign=false",
    "tag",
    "runner/v1.0.0",
    releaseSha,
  ]);
  fs.appendFileSync(
    path.join(repository, "cmd", "runner", "main.go"),
    "// next release\n",
  );
  git(repository, ["add", "cmd/runner/main.go"]);
  git(repository, ["commit", "--quiet", "-m", "fix: Update runner release"]);
  const nextSha = git(repository, ["rev-parse", "HEAD"]);
  generateReleaseNotes({
    component: "runner",
    version: "v1.1.0",
    sha: nextSha,
    output: path.join(repository, "next-release-notes.md"),
    repository,
    dockerExec,
  });
  assert.ok(calls[1].args.includes(`runner/v1.0.0..${nextSha}`));
});

test("uses a protected token for GitHub author attribution", () => {
  const { repository, releaseSha } = createReleaseHistory();
  const output = path.join(repository, "release-notes.md");
  const githubToken = "release-token";
  let invocation;

  assert.throws(
    () =>
      generateReleaseNotes({
        component: "runner",
        version: "v1.0.0",
        sha: releaseSha,
        output,
        repository,
        includeGitHubAuthors: true,
        dockerExec: () => "notes must not be generated",
      }),
    /GITHUB_TOKEN is required/,
  );

  generateReleaseNotes({
    component: "runner",
    version: "v1.0.0",
    sha: releaseSha,
    output,
    repository,
    includeGitHubAuthors: true,
    githubToken,
    dockerExec: (command, args, options) => {
      invocation = { command, args, options };
      return "## runner/v1.0.0\n\n- Add runner release by @author.\n";
    },
  });

  assert.equal(invocation.command, "docker");
  assert.equal(invocation.args.includes("--offline"), false);
  assert.deepEqual(
    invocation.args.slice(
      invocation.args.indexOf("--env"),
      invocation.args.indexOf("--env") + 2,
    ),
    ["--env", "GITHUB_TOKEN"],
  );
  assert.equal(invocation.args.includes(githubToken), false);
  assert.equal(invocation.options.env.GITHUB_TOKEN, githubToken);
});

test("routes established component scopes to their component notes", () => {
  const { repository, runnerSha, fleetManagerSha } =
    createReleaseHistory();
  const runnerShas = [
    runnerSha,
    commitSharedReleaseChange(repository, "fix(runners): Update runner"),
    commitSharedReleaseChange(
      repository,
      "chore(release/runner): Package runner",
    ),
  ];
  const fleetManagerShas = [
    fleetManagerSha,
    commitSharedReleaseChange(
      repository,
      "fix(fleets): Update Fleet Manager",
    ),
    commitSharedReleaseChange(
      repository,
      "chore(release/fleet-manager): Package Fleet Manager",
    ),
  ];
  const releaseSha = git(repository, ["rev-parse", "HEAD"]);
  const calls = [];
  const dockerExec = (command, args) => {
    calls.push({ command, args });
    return "## release notes\n";
  };

  generateReleaseNotes({
    component: "runner",
    version: "v1.0.0",
    sha: releaseSha,
    output: path.join(repository, "runner-notes.md"),
    repository,
    dockerExec,
  });
  generateReleaseNotes({
    component: "fleet-manager",
    version: "v1.0.0",
    sha: releaseSha,
    output: path.join(repository, "fleet-manager-notes.md"),
    repository,
    dockerExec,
  });

  const runnerArgs = calls[0].args;
  const fleetManagerArgs = calls[1].args;
  assert.ok(runnerArgs.includes("--skip-commit"));
  for (const sha of fleetManagerShas) {
    assert.ok(runnerArgs.includes(sha));
  }
  for (const sha of runnerShas) {
    assert.equal(runnerArgs.includes(sha), false);
  }
  assert.ok(fleetManagerArgs.includes("--skip-commit"));
  for (const sha of runnerShas) {
    assert.ok(fleetManagerArgs.includes(sha));
  }
  for (const sha of fleetManagerShas) {
    assert.equal(fleetManagerArgs.includes(sha), false);
  }
});

test("skips git-cliff for release candidates", () => {
  const { repository, releaseSha } = createReleaseHistory();
  const output = path.join(repository, "release-notes.md");

  assert.equal(
    generateReleaseNotes({
      component: "runner",
      version: "v1.0.0-rc.1",
      sha: releaseSha,
      output,
      repository,
      dockerExec: () => {
        throw new Error("Docker must not run for a prerelease");
      },
    }),
    false,
  );
  assert.equal(fs.existsSync(output), false);
});

test("manifest verification rejects altered assets", () => {
  const directory = fs.mkdtempSync(
    path.join(os.tmpdir(), "component-release-"),
  );
  fs.writeFileSync(path.join(directory, "runner-linux-amd64.tar.gz"), "amd64");
  fs.writeFileSync(path.join(directory, "checksums.txt"), "checksums");
  fs.writeFileSync(
    path.join(directory, "release-notes.md"),
    "## runner/v1.2.3\n",
  );

  const manifest = createManifest({
    component: "runner",
    version: "v1.2.3",
    sha: "a".repeat(40),
    workflowID: "workflow-1",
    pipelineID: "pipeline-1",
    directory,
    assetNames: ["runner-linux-amd64.tar.gz", "checksums.txt"],
  });
  verifyManifest(manifest, {
    component: "runner",
    version: "v1.2.3",
    sha: "a".repeat(40),
    workflowID: "workflow-1",
    directory,
    assetNames: ["runner-linux-amd64.tar.gz", "checksums.txt"],
  });

  assert.throws(
    () =>
      verifyManifest(manifest, {
        component: "runner",
        version: "v1.2.3",
        sha: "a".repeat(40),
        workflowID: "workflow-2",
        directory,
        assetNames: ["runner-linux-amd64.tar.gz", "checksums.txt"],
      }),
    /manifest identity/,
  );

  fs.writeFileSync(
    path.join(directory, "runner-linux-amd64.tar.gz"),
    "changed",
  );
  assert.throws(
    () =>
      verifyManifest(manifest, {
        component: "runner",
        version: "v1.2.3",
        sha: "a".repeat(40),
        workflowID: "workflow-1",
        directory,
        assetNames: ["runner-linux-amd64.tar.gz", "checksums.txt"],
      }),
    /checksum/,
  );

  assert.throws(
    () =>
      verifyManifest(manifest, {
        component: "runner",
        version: "v1.2.3",
        sha: "a".repeat(40),
        workflowID: "workflow-1",
        directory,
        assetNames: ["runner-linux-amd64.tar.gz"],
      }),
    /filenames/,
  );
});

function sha256OfZeros(size) {
  const hash = crypto.createHash("sha256");
  const chunk = Buffer.alloc(8 * 1024 * 1024);
  let remaining = size;
  while (remaining > 0) {
    const length = Math.min(chunk.length, remaining);
    hash.update(length === chunk.length ? chunk : chunk.subarray(0, length));
    remaining -= length;
  }
  return hash.digest("hex");
}

test("manifest records the checksum of an asset larger than 2 GiB", () => {
  const directory = fs.mkdtempSync(
    path.join(os.tmpdir(), "component-release-large-asset-"),
  );
  const assetSize = 2 * 1024 * 1024 * 1024 + 1;
  const asset = fs.openSync(path.join(directory, "runner.oci.tar"), "w");
  try {
    fs.ftruncateSync(asset, assetSize);
  } finally {
    fs.closeSync(asset);
  }
  fs.writeFileSync(
    path.join(directory, "release-notes.md"),
    "## runner/v1.2.3\n",
  );

  try {
    const manifest = createManifest({
      component: "runner",
      version: "v1.2.3",
      sha: "a".repeat(40),
      workflowID: "workflow-1",
      pipelineID: "pipeline-1",
      directory,
      assetNames: ["runner.oci.tar"],
    });

    assert.equal(manifest.assets[0].size, assetSize);
    assert.equal(manifest.assets[0].sha256, sha256OfZeros(assetSize));
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test("manifest binds stable notes and excludes them from prereleases", () => {
  const stableDirectory = fs.mkdtempSync(
    path.join(os.tmpdir(), "stable-release-notes-"),
  );
  fs.writeFileSync(path.join(stableDirectory, "checksums.txt"), "checksums");
  fs.writeFileSync(
    path.join(stableDirectory, "release-notes.md"),
    "## runner/v1.2.3\n",
  );
  const manifest = createManifest({
    component: "runner",
    version: "v1.2.3",
    sha: "a".repeat(40),
    workflowID: "workflow-1",
    pipelineID: "pipeline-1",
    directory: stableDirectory,
    assetNames: ["checksums.txt"],
  });
  fs.writeFileSync(
    path.join(stableDirectory, "release-notes.md"),
    "altered notes\n",
  );
  assert.throws(
    () =>
      verifyManifest(manifest, {
        component: "runner",
        version: "v1.2.3",
        sha: "a".repeat(40),
        workflowID: "workflow-1",
        directory: stableDirectory,
        assetNames: ["checksums.txt"],
      }),
    /release notes checksum/,
  );

  const prereleaseDirectory = fs.mkdtempSync(
    path.join(os.tmpdir(), "prerelease-notes-"),
  );
  fs.writeFileSync(
    path.join(prereleaseDirectory, "checksums.txt"),
    "checksums",
  );
  fs.writeFileSync(
    path.join(prereleaseDirectory, "release-notes.md"),
    "notes must not exist\n",
  );
  assert.throws(
    () =>
      createManifest({
        component: "runner",
        version: "v1.2.3-rc.1",
        sha: "a".repeat(40),
        workflowID: "workflow-1",
        pipelineID: "pipeline-1",
        directory: prereleaseDirectory,
        assetNames: ["checksums.txt"],
      }),
    /release candidates must not include/,
  );
});

test("draft resumption follows pagination and rejects collisions", async () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "github-release-"));
  const assetName = "runner-linux-amd64.tar.gz";
  fs.writeFileSync(path.join(directory, assetName), "prepared artifact");
  const requests = [];
  let remoteAsset = Buffer.from("prepared artifact");

  const server = http.createServer((request, response) => {
    requests.push(`${request.method} ${request.url}`);
    if (request.method === "GET" && request.url.includes("/releases/tags/")) {
      response.writeHead(404, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ message: "Not Found" }));
      return;
    }
    if (request.method === "GET" && request.url.endsWith("?per_page=100")) {
      response.writeHead(200, {
        "Content-Type": "application/json",
        Link: `<${apiURL}/repos/superplanehq/superplane/releases?per_page=100&page=2>; rel="next"`,
      });
      response.end("[]");
      return;
    }
    if (request.method === "GET" && request.url.endsWith("&page=2")) {
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(
        JSON.stringify([
          {
            id: 42,
            draft: true,
            tag_name: "runner/v1.2.3",
            target_commitish: "a".repeat(40),
            assets: [{ id: 9, name: assetName }],
          },
        ]),
      );
      return;
    }
    if (
      request.method === "GET" &&
      request.url.endsWith("/releases/assets/9")
    ) {
      response.writeHead(200, { "Content-Type": "application/octet-stream" });
      response.end(remoteAsset);
      return;
    }
    response.writeHead(500);
    response.end("unexpected request");
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  const apiURL = `http://127.0.0.1:${address.port}`;

  try {
    await stageGitHubRelease({
      apiURL,
      repository: "superplanehq/superplane",
      token: "test-token",
      component: "runner",
      version: "v1.2.3",
      sha: "a".repeat(40),
      directory,
      assetNames: [assetName],
    });
    assert.equal(
      requests.filter((entry) => entry.startsWith("POST ")).length,
      0,
    );

    remoteAsset = Buffer.from("different artifact");
    await assert.rejects(
      stageGitHubRelease({
        apiURL,
        repository: "superplanehq/superplane",
        token: "test-token",
        component: "runner",
        version: "v1.2.3",
        sha: "a".repeat(40),
        directory,
        assetNames: [assetName],
      }),
      /does not match/,
    );
  } finally {
    server.close();
  }
});

test("stable releases include cliff notes and become latest", async () => {
  let payload;
  const server = http.createServer((request, response) => {
    const chunks = [];
    request.on("data", (chunk) => chunks.push(chunk));
    request.on("end", () => {
      payload = JSON.parse(Buffer.concat(chunks).toString("utf8"));
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ id: 42 }));
    });
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();

  try {
    await publishGitHubRelease({
      apiURL: `http://127.0.0.1:${address.port}`,
      repository: "superplanehq/superplane",
      token: "test-token",
      releaseID: "42",
      component: "runner",
      version: "v1.2.3",
      sha: "a".repeat(40),
      releaseNotes: "## Runner changes\n\n- Add a release workflow.",
    });
  } finally {
    server.close();
  }

  assert.deepEqual(payload, {
    draft: false,
    prerelease: false,
    make_latest: "true",
    body: `${"Source commit: "}${"a".repeat(40)}\n\n## Runner changes\n\n- Add a release workflow.`,
  });
});

test("release candidates omit cliff notes and do not become latest", async () => {
  let payload;
  const server = http.createServer((request, response) => {
    const chunks = [];
    request.on("data", (chunk) => chunks.push(chunk));
    request.on("end", () => {
      payload = JSON.parse(Buffer.concat(chunks).toString("utf8"));
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ id: 43 }));
    });
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();

  try {
    await publishGitHubRelease({
      apiURL: `http://127.0.0.1:${address.port}`,
      repository: "superplanehq/superplane",
      token: "test-token",
      releaseID: "43",
      component: "runner",
      version: "v1.2.3-rc.1",
      sha: "b".repeat(40),
    });
  } finally {
    server.close();
  }

  assert.deepEqual(payload, {
    draft: false,
    prerelease: true,
    make_latest: "false",
    body: `${"Source commit: "}${"b".repeat(40)}`,
  });
});

test("Fleet Manager releases record image and chart digests", async () => {
  let payload;
  const server = http.createServer((request, response) => {
    const chunks = [];
    request.on("data", (chunk) => chunks.push(chunk));
    request.on("end", () => {
      payload = JSON.parse(Buffer.concat(chunks).toString("utf8"));
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ id: 44 }));
    });
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();

  try {
    await publishGitHubRelease({
      apiURL: `http://127.0.0.1:${address.port}`,
      repository: "superplanehq/superplane",
      token: "test-token",
      releaseID: "44",
      component: "fleet-manager",
      version: "v1.2.3",
      sha: "c".repeat(40),
      imageDigest: `sha256:${"d".repeat(64)}`,
      chartDigest: `sha256:${"e".repeat(64)}`,
      releaseNotes: "## Fleet Manager changes\n\n- Publish the chart.",
    });
  } finally {
    server.close();
  }

  assert.match(payload.body, /Fleet Manager image digest: `sha256:d{64}`/);
  assert.match(payload.body, /Fleet Manager chart digest: `sha256:e{64}`/);
});

test("SuperPlane releases record image and chart digests", async () => {
  let payload;
  const server = http.createServer((request, response) => {
    const chunks = [];
    request.on("data", (chunk) => chunks.push(chunk));
    request.on("end", () => {
      payload = JSON.parse(Buffer.concat(chunks).toString("utf8"));
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ id: 45 }));
    });
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();

  try {
    await publishGitHubRelease({
      apiURL: `http://127.0.0.1:${address.port}`,
      repository: "superplanehq/superplane",
      token: "test-token",
      releaseID: "45",
      component: "superplane",
      version: "v1.2.3",
      sha: "f".repeat(40),
      imageDigest: `sha256:${"a".repeat(64)}`,
      chartDigest: `sha256:${"b".repeat(64)}`,
      releaseNotes: "## SuperPlane changes\n\n- Publish the application.",
    });
  } finally {
    server.close();
  }

  assert.match(payload.body, /SuperPlane image digest: `sha256:a{64}`/);
  assert.match(payload.body, /SuperPlane chart digest: `sha256:b{64}`/);
});

test("release notes match the release type", async () => {
  const common = {
    apiURL: "http://127.0.0.1:1",
    repository: "superplanehq/superplane",
    token: "test-token",
    releaseID: "42",
    component: "runner",
    sha: "a".repeat(40),
  };

  await assert.rejects(
    publishGitHubRelease({ ...common, version: "v1.2.3" }),
    /stable releases require git-cliff release notes/,
  );
  await assert.rejects(
    publishGitHubRelease({
      ...common,
      version: "v1.2.3-rc.1",
      releaseNotes: "These notes must not be published.",
    }),
    /release candidates must not include git-cliff release notes/,
  );
});
