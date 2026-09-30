import { describe, expect, it } from "bun:test";

import { normalizeTerminalOutput, shellScriptHeadline, shellScriptLineCount } from "@/lib/shellScript";

const cloneScript = [
  "set -euo pipefail",
  'if [ -z "${REPO_URL:-}" ]; then',
  '  echo "This workspace has no repository to analyze." >&2',
  "  exit 1",
  "fi",
  'git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"',
  "rm -rf repo",
  'git clone --depth 1 --branch "${BASE:-main}" "${REPO_URL}" repo',
].join("\n");

describe("shellScriptHeadline", () => {
  it("names the clone command instead of the setup prologue", () => {
    expect(shellScriptHeadline(cloneScript)).toBe('git clone --depth 1 --branch "${BASE:-main}" "${REPO_URL}" repo');
  });

  it("keeps a single command line", () => {
    expect(shellScriptHeadline("git status")).toBe("git status");
  });

  it("skips a leading cd in a chain", () => {
    expect(shellScriptHeadline("cd /tmp/opencode && curl -fL bun.zip")).toBe("curl -fL bun.zip");
  });

  it("keeps a setup command when it is the whole script", () => {
    expect(shellScriptHeadline("rm -rf repo")).toBe("rm -rf repo");
  });

  it("skips a shebang and comments", () => {
    expect(shellScriptHeadline("#!/bin/bash\n# refresh\ngit status")).toBe("git status");
  });

  it("returns an empty string for a blank script", () => {
    expect(shellScriptHeadline("  \n")).toBe("");
  });
});

describe("shellScriptLineCount", () => {
  it("counts non-empty lines", () => {
    expect(shellScriptLineCount(cloneScript)).toBe(8);
    expect(shellScriptLineCount("git status\n\n")).toBe(1);
  });
});

describe("normalizeTerminalOutput", () => {
  it("keeps the last git progress update on each line", () => {
    const raw = [
      "Cloning into 'repo'...",
      "remote: Enumerating objects: 100\rremote: Enumerating objects: 9364, done.",
      "remote: Counting objects:  50%\rremote: Counting objects: 100% (9364/9364), done.",
      "Receiving objects:  10%\rReceiving objects: 100% (9364/9364), 14.47 MiB | 15.16 MiB/s, done.",
      "Resolving deltas: 100% (1891/1891), done.",
    ].join("\n");

    expect(normalizeTerminalOutput(raw)).toBe(
      [
        "Cloning into 'repo'...",
        "remote: Enumerating objects: 9364, done.",
        "remote: Counting objects: 100% (9364/9364), done.",
        "Receiving objects: 100% (9364/9364), 14.47 MiB | 15.16 MiB/s, done.",
        "Resolving deltas: 100% (1891/1891), done.",
      ].join("\n"),
    );
  });

  it("strips ANSI color codes", () => {
    expect(normalizeTerminalOutput("\u001b[32mdone\u001b[0m")).toBe("done");
  });

  it("strips terminal title sequences", () => {
    expect(normalizeTerminalOutput("\u001b]0;clone\u0007done")).toBe("done");
  });

  it("collapses long blank runs and drops trailing blank lines", () => {
    expect(normalizeTerminalOutput("one\n\n\n\n\ntwo\n\n")).toBe("one\n\ntwo");
  });

  it("returns an empty string when there is no output", () => {
    expect(normalizeTerminalOutput("")).toBe("");
    expect(normalizeTerminalOutput("\n\n")).toBe("");
  });
});
