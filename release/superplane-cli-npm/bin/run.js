#!/usr/bin/env node
"use strict";

const path = require("path");
const { spawnSync } = require("child_process");

const bin = path.join(__dirname, "superplane");
const result = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  const code = result.error.code;
  if (code === "ENOENT" || code === "EACCES" || code === "ENOEXEC") {
    console.error(
      "@superplane/cli: could not run the binary at " +
        bin +
        " (" + code + "). Did the postinstall step fail? " +
        "Try `npm rebuild @superplane/cli`."
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
