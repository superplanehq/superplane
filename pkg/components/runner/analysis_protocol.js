"use strict";

const fs = require("fs");
const path = require("path");

const ANALYSIS_PROTOCOL = fs.readFileSync(path.join(__dirname, "analysis_protocol.md"), "utf8").trim();

function analysisProtocol() {
  return ANALYSIS_PROTOCOL;
}

function analysisProtocolForPrompt(prompt) {
  return String(prompt || "").includes(ANALYSIS_PROTOCOL) ? "" : ANALYSIS_PROTOCOL;
}

function withAnalysisContinuation(taskDir, promptCount, prompt) {
  if (!taskDir || Number(promptCount) > 0) {
    return prompt;
  }
  const file = require("path").join(taskDir, "analysis_continuation.md");
  try {
    const extra = require("fs").readFileSync(file, "utf8").trim();
    if (!extra) {
      return prompt;
    }
    return `${extra}\n\n${prompt}`;
  } catch (_err) {
    return prompt;
  }
}

module.exports = { ANALYSIS_PROTOCOL, analysisProtocol, analysisProtocolForPrompt, withAnalysisContinuation };
