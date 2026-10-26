#!/usr/bin/env node

/* eslint-disable no-console */

const crypto = require("node:crypto");
const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const http = require("node:http");
const https = require("node:https");
const path = require("node:path");

const API_VERSION = "2022-11-28";
const USER_AGENT = "superplane-component-release";
const VERSION_PATTERN =
  /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*))*))?$/;
const SHA_PATTERN = /^[0-9a-f]{40}$/;
const RELEASE_NOTES_NAME = "release-notes.md";
const GIT_CLIFF_IMAGE =
  "ghcr.io/orhun/git-cliff/git-cliff:2.14.0@sha256:0f72b15def11c5836c9dfb2e4da8b9b52b4bc2c8c2f46c47701b7e68f3998ff7";
const COMPONENTS = {
  superplane: {
    initialChangelogRef: "v0.30.0",
    tagPrefix: "",
    releaseName: "SuperPlane",
    releaseScopes: ["superplane", "release/superplane"],
    releasePaths: [],
    digestNames: {
      image: "SuperPlane image",
      chart: "SuperPlane chart",
    },
  },
  cli: {
    initialChangelogRef: "v0.30.0",
    tagPrefix: "cli/",
    releaseName: "SuperPlane CLI",
    releaseScopes: ["cli", "release/cli"],
    releasePaths: [
      ".semaphore/*cli-release.yml",
      "cliff.toml",
      "cmd/cli/**",
      "pkg/cli/**",
      "release/cli/**",
      "release/superplane-cli-npm/**",
      "release/component-release.*",
    ],
  },
  runner: {
    initialChangelogRef: "v0.30.0",
    tagPrefix: "runner/",
    releaseName: "Runner",
    releaseScopes: ["runner", "runners", "release/runner"],
    releasePaths: [
      ".semaphore/*runner-release.yml",
      "cliff.toml",
      "cmd/runner/**",
      "pkg/runners/**",
      "release/component-release.*",
      "release/runner/**",
    ],
  },
  "fleet-manager": {
    initialChangelogRef: "v0.30.0",
    tagPrefix: "fleet-manager/",
    releaseName: "Fleet Manager",
    releaseScopes: [
      "fleet-manager",
      "fleets",
      "release/fleet-manager",
      "release/fleet-manager-chart",
    ],
    releasePaths: [
      ".semaphore/*fleet-manager-release.yml",
      "cliff.toml",
      "cmd/fleetmanager/**",
      "pkg/fleets/**",
      "release/component-release.*",
      "release/fleet-manager/**",
      "release/fleet-manager-chart/**",
    ],
    digestNames: {
      image: "Fleet Manager image",
      chart: "Fleet Manager chart",
    },
  },
};

function parseVersion(version) {
  if (!VERSION_PATTERN.test(version || "")) {
    throw new Error(
      "version must be vMAJOR.MINOR.PATCH with an optional SemVer prerelease and no build metadata",
    );
  }
  return version;
}

function componentConfig(component) {
  const config = COMPONENTS[component];
  if (!config) {
    throw new Error(`unsupported component: ${component}`);
  }
  return config;
}

function componentTag(component, version) {
  return `${componentConfig(component).tagPrefix}${parseVersion(version)}`;
}

function validateSHA(sha) {
  if (!SHA_PATTERN.test(sha || "")) {
    throw new Error("source SHA must be a full lowercase 40-character Git SHA");
  }
  return sha;
}

function isPrerelease(version) {
  return parseVersion(version).includes("-");
}

function validateFinalization(candidateVersion, stableVersion) {
  const candidate = parseVersion(candidateVersion);
  const stable = parseVersion(stableVersion);

  if (!candidate.includes("-")) {
    throw new Error("candidate version must be a prerelease");
  }
  if (stable.includes("-")) {
    throw new Error("stable version must not be a prerelease");
  }

  const expectedStableVersion = candidate.slice(0, candidate.indexOf("-"));
  if (stable !== expectedStableVersion) {
    throw new Error(`stable version must be ${expectedStableVersion}`);
  }
  return stable;
}

function sha256File(filePath) {
  const hash = crypto.createHash("sha256");
  const file = fs.openSync(filePath, "r");
  const chunk = Buffer.alloc(8 * 1024 * 1024);
  try {
    let bytesRead = 0;
    do {
      bytesRead = fs.readSync(file, chunk, 0, chunk.length, null);
      if (bytesRead > 0) {
        hash.update(chunk.subarray(0, bytesRead));
      }
    } while (bytesRead > 0);
  } finally {
    fs.closeSync(file);
  }
  return hash.digest("hex");
}

function releaseNotesMetadata(version, directory) {
  const filePath = path.join(directory, RELEASE_NOTES_NAME);
  if (isPrerelease(version)) {
    if (fs.existsSync(filePath)) {
      throw new Error(
        "release candidates must not include git-cliff release notes",
      );
    }
    return null;
  }
  if (!fs.existsSync(filePath)) {
    throw new Error("stable releases require git-cliff release notes");
  }
  const stat = fs.statSync(filePath);
  if (!stat.isFile() || stat.size === 0) {
    throw new Error("git-cliff release notes must be a non-empty file");
  }
  return {
    name: RELEASE_NOTES_NAME,
    sha256: sha256File(filePath),
    size: stat.size,
  };
}

function gitOutput(args, repository) {
  return execFileSync("git", args, {
    cwd: repository,
    encoding: "utf8",
  }).trim();
}

function previousStableTag(component, sha, repository) {
  const config = componentConfig(component);
  const tagPrefix = config.tagPrefix;
  const componentTags = gitOutput(
    [
      "tag",
      "--merged",
      sha,
      "--list",
      `${tagPrefix}v*`,
      "--sort=-version:refname",
    ],
    repository,
  ).split("\n");
  const stablePattern = new RegExp(
    `^${tagPrefix.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}v\\d+\\.\\d+\\.\\d+$`,
  );
  const previousComponentTag = componentTags.find((tag) =>
    stablePattern.test(tag),
  );
  if (previousComponentTag) {
    return previousComponentTag;
  }

  gitOutput(
    ["merge-base", "--is-ancestor", config.initialChangelogRef, sha],
    repository,
  );
  return config.initialChangelogRef;
}

function conventionalCommitScope(subject) {
  const match = /^[a-z][a-z0-9-]*(?:\(([^)]+)\))?!?:/.exec(subject);
  return match ? match[1] : null;
}

function commitsScopedToOtherComponents(component, range, repository) {
  const scopeOwners = new Map();
  for (const [componentName, config] of Object.entries(COMPONENTS)) {
    for (const scope of config.releaseScopes) {
      scopeOwners.set(scope, componentName);
    }
  }

  const log = gitOutput(["log", "--format=%H%x09%s", range], repository);
  if (!log) {
    return [];
  }

  const commits = [];
  for (const line of log.split("\n")) {
    const separator = line.indexOf("\t");
    if (separator === -1) {
      continue;
    }
    const sha = line.slice(0, separator);
    const subject = line.slice(separator + 1);
    const owner = scopeOwners.get(conventionalCommitScope(subject));
    if (owner && owner !== component) {
      commits.push(sha);
    }
  }
  return commits;
}

function generateReleaseNotes({
  component,
  version,
  sha,
  output,
  repository,
  includeGitHubAuthors = false,
  githubToken,
  dockerExec = execFileSync,
}) {
  const config = componentConfig(component);
  parseVersion(version);
  validateSHA(sha);
  if (isPrerelease(version)) {
    console.log(`* Skipping git-cliff notes for prerelease ${version}`);
    return false;
  }

  const sourceRepository =
    repository || gitOutput(["rev-parse", "--show-toplevel"]);
  const previousTag = previousStableTag(component, sha, sourceRepository);
  const tag = componentTag(component, version);
  const tagPattern = `^${config.tagPrefix}v[0-9]+\\.[0-9]+\\.[0-9]+$`;
  const args = [
    "run",
    "--rm",
    "--volume",
    `${sourceRepository}:/app:ro`,
    "--entrypoint",
    "git-cliff",
  ];
  const dockerOptions = {
    encoding: "utf8",
    maxBuffer: 10 * 1024 * 1024,
  };
  if (includeGitHubAuthors) {
    if (!githubToken?.trim()) {
      throw new Error("GITHUB_TOKEN is required for GitHub author attribution");
    }
    args.push("--env", "GITHUB_TOKEN");
    dockerOptions.env = { ...process.env, GITHUB_TOKEN: githubToken };
  }
  args.push(
    GIT_CLIFF_IMAGE,
    "--config",
    "cliff.toml",
    "--no-exec",
    "--strip",
    "header",
    "--tag",
    tag,
    "--tag-pattern",
    tagPattern,
  );
  if (!includeGitHubAuthors) {
    args.push("--offline");
  }
  for (const releasePath of config.releasePaths) {
    args.push("--include-path", releasePath);
  }
  const range = `${previousTag}..${sha}`;
  for (const commit of commitsScopedToOtherComponents(
    component,
    range,
    sourceRepository,
  )) {
    args.push("--skip-commit", commit);
  }
  args.push(range);

  const notes = dockerExec("docker", args, dockerOptions).trim();
  if (!notes) {
    throw new Error("git-cliff generated empty release notes");
  }
  fs.writeFileSync(output, `${notes}\n`, { flag: "wx" });
  return true;
}

function createManifest({
  component,
  version,
  sha,
  workflowID,
  pipelineID,
  directory,
  assetNames,
}) {
  const tag = componentTag(component, version);
  validateSHA(sha);
  if (!workflowID || !pipelineID) {
    throw new Error("Semaphore workflow and pipeline IDs are required");
  }
  if (!Array.isArray(assetNames) || assetNames.length === 0) {
    throw new Error("manifest must contain at least one asset");
  }
  if (new Set(assetNames).size !== assetNames.length) {
    throw new Error("manifest asset names must be unique");
  }

  const assets = assetNames.map((name) => {
    if (path.basename(name) !== name) {
      throw new Error(`asset name must not contain a path: ${name}`);
    }
    const filePath = path.join(directory, name);
    const stat = fs.statSync(filePath);
    if (!stat.isFile()) {
      throw new Error(`asset is not a regular file: ${name}`);
    }
    return {
      name,
      sha256: sha256File(filePath),
      size: stat.size,
    };
  });

  return {
    schemaVersion: 2,
    component,
    version,
    tag,
    sourceSha: sha,
    semaphoreWorkflowId: workflowID,
    preparationPipelineId: pipelineID,
    releaseNotes: releaseNotesMetadata(version, directory),
    assets,
  };
}

function verifyManifest(
  manifest,
  { component, version, sha, workflowID, directory, assetNames },
) {
  if (manifest.schemaVersion !== 2) {
    throw new Error(`unsupported manifest schema: ${manifest.schemaVersion}`);
  }
  if (
    manifest.component !== component ||
    manifest.version !== parseVersion(version) ||
    manifest.tag !== componentTag(component, version) ||
    manifest.sourceSha !== validateSHA(sha) ||
    manifest.semaphoreWorkflowId !== workflowID ||
    !manifest.preparationPipelineId
  ) {
    throw new Error("manifest identity does not match this release workflow");
  }
  if (!Array.isArray(manifest.assets) || manifest.assets.length === 0) {
    throw new Error("manifest does not contain assets");
  }

  const expectedReleaseNotes = releaseNotesMetadata(version, directory);
  if (
    JSON.stringify(manifest.releaseNotes) !==
    JSON.stringify(expectedReleaseNotes)
  ) {
    throw new Error("release notes checksum does not match manifest");
  }
  const recordedNames = manifest.assets.map((asset) => asset.name).sort();
  const expectedNames = [...assetNames].sort();
  if (
    recordedNames.length !== expectedNames.length ||
    recordedNames.some((name, index) => name !== expectedNames[index])
  ) {
    throw new Error("manifest filenames do not match the release contract");
  }

  for (const asset of manifest.assets) {
    const filePath = path.join(directory, asset.name);
    const stat = fs.statSync(filePath);
    if (stat.size !== asset.size || sha256File(filePath) !== asset.sha256) {
      throw new Error(`asset checksum does not match manifest: ${asset.name}`);
    }
  }
  return manifest;
}

function request({ apiURL, method, pathname, token, body, headers = {} }) {
  const url = new URL(pathname, apiURL);
  const transport = url.protocol === "http:" ? http : https;
  const payload = body === undefined ? undefined : Buffer.from(body);
  return new Promise((resolve, reject) => {
    const requestHeaders = {
      Accept: "application/vnd.github+json",
      "User-Agent": USER_AGENT,
      "X-GitHub-Api-Version": API_VERSION,
      ...headers,
    };
    if (token) {
      requestHeaders.Authorization = `Bearer ${token}`;
    }
    if (payload) {
      requestHeaders["Content-Length"] = payload.length;
    }
    const outgoing = transport.request(
      url,
      { method, headers: requestHeaders },
      (response) => {
        const chunks = [];
        response.on("data", (chunk) => chunks.push(chunk));
        response.on("end", () =>
          resolve({
            statusCode: response.statusCode,
            headers: response.headers,
            body: Buffer.concat(chunks),
          }),
        );
      },
    );
    outgoing.on("error", reject);
    if (payload) {
      outgoing.write(payload);
    }
    outgoing.end();
  });
}

function parseJSON(response, action) {
  try {
    return JSON.parse(response.body.toString("utf8"));
  } catch (error) {
    throw new Error(`${action} returned invalid JSON: ${error.message}`);
  }
}

async function verifyPublishedPrerelease({
  apiURL,
  repository,
  token,
  component,
  version,
  sha,
}) {
  const tag = componentTag(component, version);
  validateSHA(sha);
  if (!isPrerelease(version)) {
    throw new Error(`release ${tag} is not a prerelease`);
  }

  const response = await request({
    apiURL,
    method: "GET",
    pathname: `/repos/${repository}/releases/tags/${encodeURIComponent(tag)}`,
    token,
  });
  if (response.statusCode !== 200) {
    throw new Error(
      `get GitHub release ${tag} failed with HTTP ${response.statusCode}`,
    );
  }

  const release = parseJSON(response, `get GitHub release ${tag}`);
  if (
    release.draft !== false ||
    release.prerelease !== true ||
    release.tag_name !== tag
  ) {
    throw new Error(`GitHub release ${tag} is not a published prerelease`);
  }
  if (release.target_commitish !== sha) {
    throw new Error(`GitHub release ${tag} does not match source SHA ${sha}`);
  }
  return release;
}

function nextPage(linkHeader) {
  if (!linkHeader) {
    return undefined;
  }
  for (const link of linkHeader.split(",")) {
    const match = link.match(/<([^>]+)>;\s*rel="([^"]+)"/);
    if (match && match[2] === "next") {
      return match[1];
    }
  }
  return undefined;
}

async function findDraftByTag({ apiURL, repository, token, tag }) {
  let page = `/repos/${repository}/releases?per_page=100`;
  const visitedPages = new Set();

  while (page) {
    if (visitedPages.has(page)) {
      throw new Error("GitHub releases pagination contains a cycle");
    }
    visitedPages.add(page);

    const response = await request({
      apiURL,
      method: "GET",
      pathname: page,
      token,
    });
    if (response.statusCode !== 200) {
      throw new Error(
        `list GitHub releases failed with HTTP ${response.statusCode}: ${response.body}`,
      );
    }
    const release = parseJSON(response, "list GitHub releases").find(
      (candidate) => candidate.tag_name === tag,
    );
    if (release) {
      return release;
    }
    page = nextPage(response.headers.link);
  }
  return undefined;
}

async function findOrCreateDraft({
  apiURL,
  repository,
  token,
  component,
  version,
  sha,
}) {
  const config = componentConfig(component);
  const tag = componentTag(component, version);
  validateSHA(sha);
  const tagPath = `/repos/${repository}/releases/tags/${encodeURIComponent(tag)}`;
  const lookup = await request({
    apiURL,
    method: "GET",
    pathname: tagPath,
    token,
  });
  let release;
  if (lookup.statusCode === 404) {
    release = await findDraftByTag({
      apiURL,
      repository,
      token,
      tag,
    });
    if (!release) {
      const payload = JSON.stringify({
        tag_name: tag,
        target_commitish: sha,
        name: `${config.releaseName} ${version}`,
        body: `Source commit: ${sha}`,
        draft: true,
        prerelease: version.includes("-"),
        generate_release_notes: false,
        make_latest: "false",
      });
      const created = await request({
        apiURL,
        method: "POST",
        pathname: `/repos/${repository}/releases`,
        token,
        body: payload,
        headers: { "Content-Type": "application/json" },
      });
      if (created.statusCode !== 201) {
        throw new Error(
          `create GitHub draft failed with HTTP ${created.statusCode}: ${created.body}`,
        );
      }
      release = parseJSON(created, "create GitHub draft");
    }
  } else if (lookup.statusCode === 200) {
    release = parseJSON(lookup, "get GitHub release");
  } else {
    throw new Error(
      `get GitHub release failed with HTTP ${lookup.statusCode}: ${lookup.body}`,
    );
  }

  if (!release.draft) {
    throw new Error(`GitHub release ${tag} already exists and is not a draft`);
  }
  if (release.tag_name !== tag || release.target_commitish !== sha) {
    throw new Error(`GitHub draft ${tag} does not match source SHA ${sha}`);
  }
  if (
    release.prerelease !== undefined &&
    release.prerelease !== isPrerelease(version)
  ) {
    throw new Error(`GitHub draft ${tag} has the wrong prerelease setting`);
  }
  return release;
}

async function downloadAsset({ apiURL, repository, token, assetID }) {
  let response = await request({
    apiURL,
    method: "GET",
    pathname: `/repos/${repository}/releases/assets/${assetID}`,
    token,
    headers: { Accept: "application/octet-stream" },
  });
  if (
    [301, 302, 303, 307, 308].includes(response.statusCode) &&
    response.headers.location
  ) {
    response = await request({
      apiURL: response.headers.location,
      method: "GET",
      pathname: "",
      headers: { Accept: "application/octet-stream" },
    });
  }
  if (response.statusCode !== 200) {
    throw new Error(
      `download GitHub asset failed with HTTP ${response.statusCode}`,
    );
  }
  return response.body;
}

function uploadAPIURL(apiURL) {
  const url = new URL(apiURL);
  if (url.hostname === "api.github.com") {
    url.hostname = "uploads.github.com";
  }
  return url.toString();
}

async function stageGitHubRelease(options) {
  const release = await findOrCreateDraft(options);
  const existingAssets = new Map(
    (release.assets || []).map((asset) => [asset.name, asset]),
  );

  for (const assetName of options.assetNames) {
    const filePath = path.join(options.directory, assetName);
    const local = fs.readFileSync(filePath);
    const existing = existingAssets.get(assetName);
    if (existing) {
      const remote = await downloadAsset({
        apiURL: options.apiURL,
        repository: options.repository,
        token: options.token,
        assetID: existing.id,
      });
      if (
        !crypto.timingSafeEqual(
          crypto.createHash("sha256").update(local).digest(),
          crypto.createHash("sha256").update(remote).digest(),
        )
      ) {
        throw new Error(
          `existing GitHub asset ${assetName} does not match prepared content`,
        );
      }
      console.log(`* Reusing matching GitHub asset ${assetName}`);
      continue;
    }

    const uploaded = await request({
      apiURL: uploadAPIURL(options.apiURL),
      method: "POST",
      pathname: `/repos/${options.repository}/releases/${release.id}/assets?name=${encodeURIComponent(assetName)}`,
      token: options.token,
      body: local,
      headers: { "Content-Type": "application/octet-stream" },
    });
    if (uploaded.statusCode !== 201) {
      throw new Error(
        `upload GitHub asset ${assetName} failed with HTTP ${uploaded.statusCode}: ${uploaded.body}`,
      );
    }
  }
  return release;
}

async function publishGitHubRelease({
  apiURL,
  repository,
  token,
  releaseID,
  component,
  version,
  sha,
  imageDigest,
  chartDigest,
  releaseNotes,
}) {
  componentConfig(component);
  const prerelease = isPrerelease(version);
  validateSHA(sha);
  if (prerelease && releaseNotes) {
    throw new Error(
      "release candidates must not include git-cliff release notes",
    );
  }
  if (!prerelease && !releaseNotes?.trim()) {
    throw new Error("stable releases require git-cliff release notes");
  }

  const config = componentConfig(component);
  let body = `Source commit: ${sha}`;
  if (imageDigest) {
    if (!config.digestNames?.image) {
      throw new Error(`${component} releases do not publish an image digest`);
    }
    body += `\n\n${config.digestNames.image} digest: \`${imageDigest}\``;
  }
  if (chartDigest) {
    if (!config.digestNames?.chart) {
      throw new Error(`${component} releases do not publish a chart digest`);
    }
    body += `\n\n${config.digestNames.chart} digest: \`${chartDigest}\``;
  }
  if (!prerelease) {
    body += `\n\n${releaseNotes.trim()}`;
  }
  const response = await request({
    apiURL,
    method: "PATCH",
    pathname: `/repos/${repository}/releases/${releaseID}`,
    token,
    body: JSON.stringify({
      draft: false,
      prerelease,
      make_latest: prerelease ? "false" : "true",
      body,
    }),
    headers: { "Content-Type": "application/json" },
  });
  if (response.statusCode !== 200) {
    throw new Error(
      `publish GitHub release failed with HTTP ${response.statusCode}: ${response.body}`,
    );
  }
  return parseJSON(response, "publish GitHub release");
}

function option(name) {
  const index = process.argv.indexOf(`--${name}`);
  if (index === -1 || !process.argv[index + 1]) {
    throw new Error(`--${name} is required`);
  }
  return process.argv[index + 1];
}

function optionalOption(name) {
  const index = process.argv.indexOf(`--${name}`);
  return index === -1 ? undefined : process.argv[index + 1];
}

function assetNamesOption() {
  return option("assets").split(",").filter(Boolean);
}

async function main() {
  const command = process.argv[2];
  if (command === "validate-version") {
    console.log(parseVersion(process.argv[3]));
    return;
  }
  if (command === "tag") {
    console.log(componentTag(process.argv[3], process.argv[4]));
    return;
  }
  if (command === "validate-finalization") {
    console.log(validateFinalization(process.argv[3], process.argv[4]));
    return;
  }
  if (command === "verify-published-prerelease") {
    const release = await verifyPublishedPrerelease({
      apiURL: process.env.GITHUB_API_URL || "https://api.github.com",
      repository: process.env.GITHUB_REPOSITORY || "superplanehq/superplane",
      token: process.env.GITHUB_TOKEN,
      component: option("component"),
      version: option("version"),
      sha: option("sha"),
    });
    console.log(`Published release candidate ${release.tag_name} verified.`);
    return;
  }
  if (command === "generate-release-notes") {
    const includeGitHubAuthors = process.argv.includes("--github-authors");
    generateReleaseNotes({
      component: option("component"),
      version: option("version"),
      sha: option("sha"),
      output: option("output"),
      includeGitHubAuthors,
      githubToken: includeGitHubAuthors
        ? process.env.GITHUB_TOKEN
        : undefined,
    });
    return;
  }
  if (command === "create-manifest") {
    const output = option("output");
    const manifest = createManifest({
      component: option("component"),
      version: option("version"),
      sha: option("sha"),
      workflowID: option("workflow-id"),
      pipelineID: option("pipeline-id"),
      directory: option("directory"),
      assetNames: assetNamesOption(),
    });
    fs.writeFileSync(output, `${JSON.stringify(manifest, null, 2)}\n`, {
      flag: "wx",
    });
    return;
  }
  if (command === "verify-manifest") {
    const manifest = JSON.parse(fs.readFileSync(option("manifest"), "utf8"));
    verifyManifest(manifest, {
      component: option("component"),
      version: option("version"),
      sha: option("sha"),
      workflowID: option("workflow-id"),
      directory: option("directory"),
      assetNames: assetNamesOption(),
    });
    console.log("Release manifest verified.");
    return;
  }
  if (command === "stage-github-release") {
    if (!process.env.GITHUB_TOKEN) {
      throw new Error("GITHUB_TOKEN is required");
    }
    const release = await stageGitHubRelease({
      apiURL: process.env.GITHUB_API_URL || "https://api.github.com",
      repository: process.env.GITHUB_REPOSITORY || "superplanehq/superplane",
      token: process.env.GITHUB_TOKEN,
      component: option("component"),
      version: option("version"),
      sha: option("sha"),
      directory: option("directory"),
      assetNames: assetNamesOption(),
    });
    const releaseIDFile = optionalOption("release-id-file");
    if (releaseIDFile) {
      fs.writeFileSync(releaseIDFile, `${release.id}\n`, { flag: "wx" });
    } else {
      console.log(release.id);
    }
    return;
  }
  if (command === "publish-github-release") {
    if (!process.env.GITHUB_TOKEN) {
      throw new Error("GITHUB_TOKEN is required");
    }
    const releaseNotesPath = optionalOption("release-notes");
    const release = await publishGitHubRelease({
      apiURL: process.env.GITHUB_API_URL || "https://api.github.com",
      repository: process.env.GITHUB_REPOSITORY || "superplanehq/superplane",
      token: process.env.GITHUB_TOKEN,
      releaseID: option("release-id"),
      component: option("component"),
      version: option("version"),
      sha: option("sha"),
      imageDigest:
        process.env.SUPERPLANE_IMAGE_DIGEST ||
        process.env.FLEET_MANAGER_IMAGE_DIGEST,
      chartDigest:
        process.env.SUPERPLANE_CHART_DIGEST ||
        process.env.FLEET_MANAGER_CHART_DIGEST,
      releaseNotes: releaseNotesPath
        ? fs.readFileSync(releaseNotesPath, "utf8")
        : undefined,
    });
    console.log(release.html_url || release.id);
    return;
  }
  throw new Error(`unknown command: ${command || "(missing)"}`);
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`Error: ${error.message}`);
    process.exit(1);
  });
}

module.exports = {
  componentTag,
  createManifest,
  generateReleaseNotes,
  parseVersion,
  publishGitHubRelease,
  stageGitHubRelease,
  validateFinalization,
  verifyManifest,
  verifyPublishedPrerelease,
};
