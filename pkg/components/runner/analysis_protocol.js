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

function isCompactStatusText(text) {
  return /^\s*compactions remaining:\s*\d+\s*$/i.test(String(text || "").trim());
}

module.exports = {
  ANALYSIS_PROTOCOL_REVIEW,
  analysisProtocol,
  withoutEmbeddedAnalysisProtocol,
  withAnalysisContinuation,
  isCompactStatusText,
};
