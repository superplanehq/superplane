#!/usr/bin/env node

/* eslint-disable no-console */

const https = require("https");
const fs = require("fs");
const path = require("path");

async function main() {
  const version = process.argv[2];

  if (!version || version.length < 2) {
    console.error(
      "Usage: node release/make-github-release.js <version>\n\nExample:\n  node release/make-github-release.js v0.0.8"
    );
    process.exit(1);
  }

  const token = process.env.GITHUB_TOKEN;
  if (!token) {
    console.error("Error: GITHUB_TOKEN is required.");
    process.exit(1);
  }

  const repository = "superplanehq/superplane";
  const apiVersion = "2022-11-28";
  const userAgent = "superplane-release-script";

  const repoRoot = path.resolve(__dirname, "..");
  const buildRoot = path.join(repoRoot, `build/superplane-single-host-tarball-${version}`);
  const artifactPath = path.join(buildRoot, "superplane-single-host.tar.gz");
  const assetName = "superplane-single-host.tar.gz";

  if (!fs.existsSync(artifactPath)) {
    console.error(
      `Error: ${artifactPath} does not exist. Run release/superplane-single-host-tarball/build.sh ${version} first.`
    );
    process.exit(1);
  }

  console.log(`* Creating GitHub release for version ${version}`);

  const payload = JSON.stringify({
    tag_name: version,
    target_commitish: "main",
    name: version,
    body: `Release ${version}`,
    draft: false,
    prerelease: false,
    generate_release_notes: false,
  });

  const createReleaseResponse = await requestJson(
    {
      hostname: "api.github.com",
      path: `/repos/${repository}/releases`,
      method: "POST",
      headers: {
        Accept: "application/vnd.github+json",
        Authorization: `Bearer ${token}`,
        "User-Agent": userAgent,
        "X-GitHub-Api-Version": apiVersion,
        "Content-Type": "application/json",
        "Content-Length": Buffer.byteLength(payload),
      },
    },
    payload
  );

  if (createReleaseResponse.statusCode !== 201) {
    console.error(
      `Error: Failed to create release (HTTP ${createReleaseResponse.statusCode}). Response:`
    );
    console.error(createReleaseResponse.body);
    process.exit(1);
  }

  let release;
  try {
    release = JSON.parse(createReleaseResponse.body);
  } catch (err) {
    console.error("Error: Could not parse release response JSON.");
    console.error(createReleaseResponse.body);
    process.exit(1);
  }

  const releaseId = release && release.id;
  if (!releaseId) {
    console.error("Error: Release ID missing from GitHub response.");
    console.error(createReleaseResponse.body);
    process.exit(1);
  }

  console.log(`* Release created successfully (id: ${releaseId})`);
  console.log(`* Uploading asset ${assetName} from ${artifactPath}`);

  const assetBuffer = fs.readFileSync(artifactPath);

  const uploadResponse = await requestJson(
    {
      hostname: "uploads.github.com",
      path: `/repos/${repository}/releases/${releaseId}/assets?name=${encodeURIComponent(
        assetName
      )}`,
      method: "POST",
      headers: {
        Accept: "application/vnd.github+json",
        Authorization: `Bearer ${token}`,
        "User-Agent": userAgent,
        "X-GitHub-Api-Version": apiVersion,
        "Content-Type": "application/octet-stream",
        "Content-Length": assetBuffer.length,
      },
    },
    assetBuffer
  );

  if (uploadResponse.statusCode !== 201) {
    console.error(
      `Error: Failed to upload asset (HTTP ${uploadResponse.statusCode}). Response:`
    );
    console.error(uploadResponse.body);
    process.exit(1);
  }

  console.log(`* ${assetName} uploaded successfully`);
  console.log("Done.");
}

function requestJson(options, body) {
  return new Promise((resolve, reject) => {
    const req = https.request(options, (res) => {
      const chunks = [];

      res.on("data", (chunk) => {
        chunks.push(chunk);
      });

      res.on("end", () => {
        const joined = Buffer.concat(chunks);
        resolve({
          statusCode: res.statusCode,
          body: joined.toString("utf8"),
        });
      });
    });

    req.on("error", (err) => {
      reject(err);
    });

    if (body) {
      req.write(body);
    }

    req.end();
  });
}

main().catch((err) => {
  console.error("Unexpected error:", err);
  process.exit(1);
});
