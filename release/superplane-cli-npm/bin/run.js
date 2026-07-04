#!/usr/bin/env node
"use strict";

const path = require("path");
const { spawnSync } = require("child_process");

const bin = path.join(__dirname, "superplane");
const result = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  if (result.error.code === "ENOENT") {
    console.error(
      "@superplane/cli: binary not found at " +
        bin +
        ". Did the postinstall step fail? Try `npm rebuild @superplane/cli`."
    );
    process.exit(127);
  }
  throw result.error;
}

if (result.signal) {
  process.kill(process.pid, result.signal);
} else {
  process.exit(result.status === null ? 1 : result.status);
}
