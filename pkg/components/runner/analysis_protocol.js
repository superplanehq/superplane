"use strict";

const fs = require("fs");
const path = require("path");

const ANALYSIS_PROTOCOL = fs.readFileSync(path.join(__dirname, "analysis_protocol.md"), "utf8").trim();

function envFlagDefaultTrue(env, name) {
  const raw = String((env && env[name]) || "true")
    .trim()
    .toLowerCase();
  return raw !== "false" && raw !== "0" && raw !== "no" && raw !== "off";
}

function planningClarityEnabled(env = process.env) {
  return envFlagDefaultTrue(env, "SUPERPLANE_PLANNING_CLARITY");
}

function planningConfidenceEnabled(env = process.env) {
  return envFlagDefaultTrue(env, "SUPERPLANE_PLANNING_CONFIDENCE");
}

function analysisProtocol(env = process.env) {
  const clarity = planningClarityEnabled(env);
  const confidence = planningConfidenceEnabled(env);
  if (clarity && confidence) {
    return ANALYSIS_PROTOCOL;
  }
  let text = ANALYSIS_PROTOCOL;
  if (clarity && !confidence) {
    text = text
      .replace(
        "Publish a Clarity score and a Confidence score every turn.",
        "Publish a Clarity score every turn.",
      )
      .replace(
        "Call propose_clarity every turn with a 1 through 5 score and a short summary. Call propose_confidence every turn with a 1 through 5 score and a short summary. Write each summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not describe agent fit in the Clarity summary. Do not name missing decisions in the Confidence summary. Each summary is one chip, not the plan.",
        "Call propose_clarity every turn with a 1 through 5 score and a short summary. Write the summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not describe agent fit in the Clarity summary. Each summary is one chip, not the plan.",
      )
      .replace("propose_spec, propose_clarity, and propose_confidence again", "propose_spec and propose_clarity again")
      .replace("the spec and the scores", "the spec and the Clarity score");
  } else if (!clarity && confidence) {
    text = text
      .replace(
        "Publish a Clarity score and a Confidence score every turn.",
        "Publish a Confidence score every turn.",
      )
      .replace(
        "Call propose_clarity every turn with a 1 through 5 score and a short summary. Call propose_confidence every turn with a 1 through 5 score and a short summary. Write each summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not describe agent fit in the Clarity summary. Do not name missing decisions in the Confidence summary. Each summary is one chip, not the plan.",
        "Call propose_confidence every turn with a 1 through 5 score and a short summary. Write the summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not name missing decisions in the Confidence summary. Each summary is one chip, not the plan.",
      )
      .replace("propose_spec, propose_clarity, and propose_confidence again", "propose_spec and propose_confidence again")
      .replace("the spec and the scores", "the spec and the Confidence score");
  } else {
    text = text
      .replace(
        "Publish a Clarity score and a Confidence score every turn.",
        "Do not publish Clarity or Confidence scores.",
      )
      .replace(
        "Call propose_clarity every turn with a 1 through 5 score and a short summary. Call propose_confidence every turn with a 1 through 5 score and a short summary. Write each summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not describe agent fit in the Clarity summary. Do not name missing decisions in the Confidence summary. Each summary is one chip, not the plan.\n\n",
        "",
      )
      .replace("propose_spec, propose_clarity, and propose_confidence again", "propose_spec again")
      .replace("the spec and the scores", "the spec");
  }
  if (!clarity) {
    text = text.replace(/propose_clarity/g, "").replace(/  +/g, " ");
  }
  if (!confidence) {
    text = text.replace(/propose_confidence/g, "").replace(/  +/g, " ");
  }
  return text.replace(/\n{3,}/g, "\n\n").trim();
}

function withoutEmbeddedAnalysisProtocol(prompt, env = process.env) {
  const value = String(prompt || "");
  const current = analysisProtocol(env);
  if (value.includes(current)) {
    return value.replace(current, "").trimStart();
  }
  if (value.includes(ANALYSIS_PROTOCOL)) {
    return value.replace(ANALYSIS_PROTOCOL, "").trimStart();
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
  ANALYSIS_PROTOCOL,
  analysisProtocol,
  withoutEmbeddedAnalysisProtocol,
  withAnalysisContinuation,
  isCompactStatusText,
  planningClarityEnabled,
  planningConfidenceEnabled,
};
