"use strict";

const fs = require("fs");
const path = require("path");

const ANALYSIS_PROTOCOL_REVIEW = fs.readFileSync(path.join(__dirname, "analysis_protocol_review.md"), "utf8").trim();

function analysisProtocol() {
  return ANALYSIS_PROTOCOL_REVIEW;
}

function withoutEmbeddedAnalysisProtocol(prompt) {
  const value = String(prompt || "");
  if (value.includes(ANALYSIS_PROTOCOL_REVIEW)) {
    return value.replace(ANALYSIS_PROTOCOL_REVIEW, "").trimStart();
  }
  return value;
}

const TASK_OWNER_RULES_FILE = path.join(".superplane", "skills", "task-owner.md");
const MAX_TASK_OWNER_RULES_BYTES = 8 * 1024;

// Repository owner rules replace the default owner rule. The runner reads
// the file so the rules apply even when the agent does not look for them.
// A symlink is skipped so the file cannot point outside the repository.
function taskOwnerRules(roots) {
  for (const root of roots) {
    if (!root) {
      continue;
    }
    const file = path.join(root, TASK_OWNER_RULES_FILE);
    let fd;
    try {
      if (!fs.lstatSync(file).isFile()) {
        continue;
      }
      const realRoot = fs.realpathSync(root);
      if (!fs.realpathSync(file).startsWith(`${realRoot}${path.sep}`)) {
        continue;
      }
      fd = fs.openSync(file, "r");
      const buffer = Buffer.alloc(MAX_TASK_OWNER_RULES_BYTES);
      const read = fs.readSync(fd, buffer, 0, MAX_TASK_OWNER_RULES_BYTES, 0);
      const rules = buffer.toString("utf8", 0, read).trim();
      if (rules) {
        return rules;
      }
    } catch (_err) {
      // The repository has no owner rules at this root.
    } finally {
      if (fd !== undefined) {
        fs.closeSync(fd);
      }
    }
  }
  return "";
}

function readContinuation(taskDir) {
  try {
    return fs.readFileSync(path.join(taskDir, "analysis_continuation.md"), "utf8").trim();
  } catch (_err) {
    return "";
  }
}

function withAnalysisContinuation(taskDir, promptCount, prompt, cwd = process.cwd()) {
  if (!taskDir || Number(promptCount) > 0) {
    return prompt;
  }
  const parts = [];
  const rules = taskOwnerRules([cwd, path.join(taskDir, "repo")]);
  if (rules) {
    parts.push(`# Repository owner rules\n\n${rules}`);
  }
  const continuation = readContinuation(taskDir);
  if (continuation) {
    parts.push(continuation);
  }
  if (parts.length === 0) {
    return prompt;
  }
  return `${parts.join("\n\n")}\n\n${prompt}`;
}

function isCompactStatusText(text) {
  return /^\s*compactions remaining:\s*\d+\s*$/i.test(String(text || "").trim());
}

module.exports = {
  ANALYSIS_PROTOCOL_REVIEW,
  analysisProtocol,
  withoutEmbeddedAnalysisProtocol,
  withAnalysisContinuation,
  taskOwnerRules,
  isCompactStatusText,
};
