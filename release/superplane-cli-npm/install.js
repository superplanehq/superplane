#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");
const https = require("https");
const { pipeline } = require("stream/promises");

const PLATFORM_MAP = {
  "darwin-x64": "darwin-amd64",
  "darwin-arm64": "darwin-arm64",
  "linux-x64": "linux-amd64",
  "linux-arm64": "linux-arm64",
};

const DOWNLOAD_HOST =
  process.env.SUPERPLANE_CLI_DOWNLOAD_HOST || "https://install.superplane.com";

function resolveTarget() {
  const platform = os.platform();
  const arch = os.arch();
  const key = `${platform}-${arch}`;
  const target = PLATFORM_MAP[key];
  if (!target) {
    console.error(
      `@superplane/cli: unsupported platform ${key}. ` +
        `Supported: ${Object.keys(PLATFORM_MAP).join(", ")}.`
    );
    process.exit(1);
  }
  return target;
}

function get(url) {
  return new Promise((resolve, reject) => {
    const req = https.get(url, resolve);
    req.on("error", reject);
  });
}

async function download(url, dest) {
  let currentUrl = url;

  for (let redirectsLeft = 5; ; redirectsLeft--) {
    const res = await get(currentUrl);

    if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
      res.resume();
      if (redirectsLeft <= 0) {
        throw new Error("too many redirects");
      }
      currentUrl = new URL(res.headers.location, currentUrl).toString();
      continue;
    }

    if (res.statusCode !== 200) {
      res.resume();
      throw new Error(`HTTP ${res.statusCode} fetching ${currentUrl}`);
    }

    // pipeline propagates a mid-stream response error to the file stream and
    // rejects. res.pipe() does not, which let a truncated download look like a
    // successful install.
    await pipeline(res, fs.createWriteStream(dest));

    const expected = Number.parseInt(res.headers["content-length"], 10);
    if (Number.isFinite(expected) && fs.statSync(dest).size !== expected) {
      throw new Error(
        `incomplete download: expected ${expected} bytes, got ${fs.statSync(dest).size}`
      );
    }

    return;
  }
}

async function main() {
  const version = require("./package.json").version;
  const target = resolveTarget();
  const binDir = path.join(__dirname, "bin");
  const binPath = path.join(binDir, "superplane");

  fs.mkdirSync(binDir, { recursive: true });

  const url = `${DOWNLOAD_HOST}/v${version}/superplane-cli-${target}`;
  console.log(`@superplane/cli: downloading ${target} binary for v${version}`);
  console.log(`  ${url}`);

  const tmpPath = `${binPath}.download-${process.pid}`;

  try {
    await download(url, tmpPath);
    fs.chmodSync(tmpPath, 0o755);
    fs.renameSync(tmpPath, binPath);
  } catch (err) {
    fs.rmSync(tmpPath, { force: true });
    console.error("@superplane/cli: failed to download binary.");
    console.error(`  ${err.message}`);
    process.exit(1);
  }

  console.log("@superplane/cli: installed.");
}

main().catch((err) => {
  console.error("@superplane/cli: unexpected error.");
  console.error(err);
  process.exit(1);
});
