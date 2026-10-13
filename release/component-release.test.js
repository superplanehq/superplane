#!/usr/bin/env node

const assert = require("node:assert/strict");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const {
  componentTag,
  createManifest,
  parseVersion,
  publishGitHubRelease,
  stageGitHubRelease,
  verifyManifest,
} = require("./component-release");
const { createJWT } = require("./github-app-token");

test("creates a short-lived signed GitHub App JWT", () => {
  const { privateKey, publicKey } = require("node:crypto").generateKeyPairSync(
    "rsa",
    {
      modulusLength: 2048,
    },
  );
  const jwt = createJWT("12345", privateKey, 2_000_000_000);
  const [header, payload, signature] = jwt.split(".");
  assert.equal(
    require("node:crypto").verify(
      "RSA-SHA256",
      Buffer.from(`${header}.${payload}`),
      publicKey,
      Buffer.from(signature, "base64url"),
    ),
    true,
  );
  assert.deepEqual(JSON.parse(Buffer.from(payload, "base64url")), {
    iat: 1_999_999_940,
    exp: 2_000_000_540,
    iss: "12345",
  });
});

test("accepts stable and prerelease versions", () => {
  assert.equal(parseVersion("v1.2.3"), "v1.2.3");
  assert.equal(parseVersion("v1.2.3-rc.1"), "v1.2.3-rc.1");
});

test("rejects build metadata and malformed prereleases", () => {
  for (const version of ["1.2.3", "v1.2", "v1.2.3+build.1", "v1.2.3-"]) {
    assert.throws(() => parseVersion(version), /version/);
  }
});

test("maps each component to its independent tag", () => {
  assert.equal(componentTag("runner", "v1.2.3"), "runner/v1.2.3");
  assert.equal(
    componentTag("fleet-manager", "v2.0.0-rc.1"),
    "fleet-manager/v2.0.0-rc.1",
  );
  assert.throws(() => componentTag("server", "v1.2.3"), /component/);
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
