"use strict";

const fs = require("fs");
const path = require("path");

const ANALYSIS_PROTOCOL = fs.readFileSync(path.join(__dirname, "analysis_protocol.md"), "utf8").trim();
const ANALYSIS_PROTOCOL_REVIEW = fs.readFileSync(path.join(__dirname, "analysis_protocol_review.md"), "utf8").trim();

function envFlagDefaultTrue(env, name) {
  const raw = String((env && env[name]) || "true")
    .trim()
    .toLowerCase();
  return raw !== "false" && raw !== "0" && raw !== "no" && raw !== "off";
}

function envFlagDefaultFalse(env, name) {
  const raw = String((env && env[name]) || "")
    .trim()
    .toLowerCase();
  return raw === "true" || raw === "1" || raw === "yes" || raw === "on";
}

function planningClarityEnabled(env = process.env) {
  return envFlagDefaultTrue(env, "SUPERPLANE_PLANNING_CLARITY");
}

function planningConfidenceEnabled(env = process.env) {
  return envFlagDefaultTrue(env, "SUPERPLANE_PLANNING_CONFIDENCE");
}

function planningReviewEnabled(env = process.env) {
  return envFlagDefaultFalse(env, "SUPERPLANE_PLANNING_REVIEW");
}

function analysisProtocol(env = process.env) {
  if (planningReviewEnabled(env)) {
    return ANALYSIS_PROTOCOL_REVIEW;
  }
  const clarity = planningClarityEnabled(env);
  const confidence = planningConfidenceEnabled(env);
  if (clarity && confidence) {
    return ANALYSIS_PROTOCOL;
  }
  let text = ANALYSIS_PROTOCOL;
  if (clarity && !confidence) {
    text = text
      .replace(
        "Publish a Clarity score and a Confidence score on the first message. Also publish them on a turn that also asks a question. Do not publish scores on an answer turn. If the user asks to update only one score, publish that score only. When every required score is 5, publish each required score on that plan turn.",
        "Publish a Clarity score on the first message. Also publish it on a turn that also asks a question. Do not publish that score on an answer turn. When every required score is 5, publish each required score on that plan turn.",
      )
      .replace(
        "Call propose_clarity and propose_confidence on the first message and on a plan turn that also asks a question. Do not call them on an answer turn. If the user asks to update only one score, call that score tool only. When every required score is 5, call each required score tool on that plan turn, even if one score did not change. Write each summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not describe agent fit in the Clarity summary. Do not name missing decisions in the Confidence summary. Each summary is one chip, not the plan. Otherwise an unchanged score stays on the card. On an answer turn, the last sentence says the user can ask to update the plan or the scores, or confirm a decision. On the first plan turn, include that same sentence after the finding. Do not add that sentence on a turn that already updates the plan.",
        "Call propose_clarity on the first message and on a plan turn that also asks a question. Do not call it on an answer turn. When every required score is 5, call that score tool on that plan turn, even if the score did not change. Write the summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not describe agent fit in the Clarity summary. Each summary is one chip, not the plan. Otherwise an unchanged score stays on the card. On an answer turn, the last sentence says the user can ask to update the plan or the scores, or confirm a decision. On the first plan turn, include that same sentence after the finding. Do not add that sentence on a turn that already updates the plan.",
      )
      .replace("the spec and the scores", "the spec and the Clarity score");
  } else if (!clarity && confidence) {
    text = text
      .replace(
        "Publish a Clarity score and a Confidence score on the first message. Also publish them on a turn that also asks a question. Do not publish scores on an answer turn. If the user asks to update only one score, publish that score only. When every required score is 5, publish each required score on that plan turn.",
        "Publish a Confidence score on the first message. Also publish it on a turn that also asks a question. Do not publish that score on an answer turn. When every required score is 5, publish each required score on that plan turn.",
      )
      .replace(
        "Call propose_clarity and propose_confidence on the first message and on a plan turn that also asks a question. Do not call them on an answer turn. If the user asks to update only one score, call that score tool only. When every required score is 5, call each required score tool on that plan turn, even if one score did not change. Write each summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not describe agent fit in the Clarity summary. Do not name missing decisions in the Confidence summary. Each summary is one chip, not the plan. Otherwise an unchanged score stays on the card. On an answer turn, the last sentence says the user can ask to update the plan or the scores, or confirm a decision. On the first plan turn, include that same sentence after the finding. Do not add that sentence on a turn that already updates the plan.",
        "Call propose_confidence on the first message and on a plan turn that also asks a question. Do not call it on an answer turn. When every required score is 5, call that score tool on that plan turn, even if the score did not change. Write the summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not name missing decisions in the Confidence summary. Each summary is one chip, not the plan. Otherwise an unchanged score stays on the card. On an answer turn, the last sentence says the user can ask to update the plan or the scores, or confirm a decision. On the first plan turn, include that same sentence after the finding. Do not add that sentence on a turn that already updates the plan.",
      )
      .replace("the spec and the scores", "the spec and the Confidence score");
  } else {
    text = text
      .replace(
        "Publish a Clarity score and a Confidence score on the first message. Also publish them on a turn that also asks a question. Do not publish scores on an answer turn. If the user asks to update only one score, publish that score only. When every required score is 5, publish each required score on that plan turn.",
        "Do not publish Clarity or Confidence scores.",
      )
      .replace(
        "Answer, then update the plan and publish any score that changed. If the user asks to update only a score, publish that score and do not rewrite the plan. When every required score is 5, publish each required score on that plan turn.",
        "Answer, then update the plan.",
      )
      .replace(
        "Call propose_clarity and propose_confidence on the first message and on a plan turn that also asks a question. Do not call them on an answer turn. If the user asks to update only one score, call that score tool only. When every required score is 5, call each required score tool on that plan turn, even if one score did not change. Write each summary the way the task prompt asks. Do not write a test or an acceptance check in a summary. Do not describe agent fit in the Clarity summary. Do not name missing decisions in the Confidence summary. Each summary is one chip, not the plan. Otherwise an unchanged score stays on the card. On an answer turn, the last sentence says the user can ask to update the plan or the scores, or confirm a decision. On the first plan turn, include that same sentence after the finding. Do not add that sentence on a turn that already updates the plan.\n\n",
        "On an answer turn, the last sentence says the user can ask to update the plan or the scores, or confirm a decision. On the first plan turn, include that same sentence after the finding. Do not add that sentence on a turn that already updates the plan.\n\n",
      )
      .replace(
        "On a plan turn, call the required score tools before survey. Call propose_spec before survey when you write a plan.",
        "On a plan turn, call propose_spec before survey when you write a plan.",
      )
      .replaceAll("Publish each required score on that turn. ", "")
      .replaceAll(
        "On a plan turn that also asks, publish each required score first. Write the plan when the task prompt allows it. Then call survey. Then stop.",
        "On a plan turn that also asks, write the plan first. Then call survey. Then stop.",
      )
      .replaceAll(
        "If a later message is unclear and scores exist, leave the plan unchanged. If no score is published yet, this turn is a plan turn.",
        "If a later message is unclear, leave the plan unchanged.",
      )
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
  ANALYSIS_PROTOCOL,
  ANALYSIS_PROTOCOL_REVIEW,
  analysisProtocol,
  withoutEmbeddedAnalysisProtocol,
  withAnalysisContinuation,
  isCompactStatusText,
  planningClarityEnabled,
  planningConfidenceEnabled,
  planningReviewEnabled,
};
