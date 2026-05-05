const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const { getCliAssets } = require("./create-github-release");

test("getCliAssets includes install.sh and all staged binaries", () => {
  const cliDir = fs.mkdtempSync(path.join(os.tmpdir(), "superplane-cli-assets-"));

  try {
    fs.writeFileSync(path.join(cliDir, "superplane-cli-darwin-arm64"), "binary");
    fs.writeFileSync(path.join(cliDir, "superplane-cli-linux-amd64"), "binary");
    fs.writeFileSync(path.join(cliDir, "install.sh"), "#!/usr/bin/env sh\n");

    const assets = getCliAssets(cliDir);
    const names = assets.map((asset) => asset.assetName).sort();

    assert.deepEqual(names, ["install.sh", "superplane-cli-darwin-arm64", "superplane-cli-linux-amd64"]);
  } finally {
    fs.rmSync(cliDir, { recursive: true, force: true });
  }
});

test("create-github-release exits when install.sh is missing", () => {
  const cliDir = fs.mkdtempSync(path.join(os.tmpdir(), "superplane-cli-assets-missing-install-"));
  const script = path.join(process.cwd(), "release/create-github-release.js");

  try {
    fs.writeFileSync(path.join(cliDir, "superplane-cli-linux-amd64"), "binary");

    const child = spawnSync(
      process.execPath,
      [
        "-e",
        `
          const { getCliAssets } = require(${JSON.stringify(script)});
          getCliAssets(${JSON.stringify(cliDir)});
        `,
      ],
      { encoding: "utf8" }
    );

    assert.equal(child.status, 1);
    assert.match(child.stderr, /install\.sh/);
  } finally {
    fs.rmSync(cliDir, { recursive: true, force: true });
  }
});
