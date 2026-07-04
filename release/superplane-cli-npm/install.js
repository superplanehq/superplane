#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");
const https = require("https");

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

function download(url, dest) {
  return new Promise((resolve, reject) => {
    function attempt(currentUrl, redirectsLeft) {
      const req = https.get(currentUrl, (res) => {
        if (
          res.statusCode >= 300 &&
          res.statusCode < 400 &&
          res.headers.location
        ) {
          res.resume();
          if (redirectsLeft <= 0) {
            reject(new Error("too many redirects"));
            return;
          }
          attempt(res.headers.location, redirectsLeft - 1);
          return;
        }
        if (res.statusCode !== 200) {
          res.resume();
          reject(new Error(`HTTP ${res.statusCode} fetching ${currentUrl}`));
          return;
        }
        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on("finish", () => file.close((err) => (err ? reject(err) : resolve())));
        file.on("error", reject);
      });
      req.on("error", reject);
    }
    attempt(url, 5);
  });
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

  try {
    await download(url, binPath);
    fs.chmodSync(binPath, 0o755);
  } catch (err) {
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
