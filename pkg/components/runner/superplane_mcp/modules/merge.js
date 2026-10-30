"use strict";

async function reportMergeCheck(input, client) {
  const check = String((input && input.check) || "")
    .trim()
    .toLowerCase();
  if (!check) throw new Error("check is required");
  const score = Number(input && input.score);
  if (!Number.isFinite(score)) throw new Error("score is required");
  const summary = String((input && input.summary) || "").trim();
  if (!summary) throw new Error("summary is required");
  return client.request("POST", "/api/v1/runner/task/merge-checks", { check, score, summary });
}

const REPORT_MERGE_CHECK = {
  definition: {
    name: "report_merge_check",
    description:
      "Publish one merge confidence check for this pull request. Call this once for the check in the prompt. check is the id from the Merge check line. score is an integer from 1 through 5. summary is one sentence.",
    inputSchema: {
      type: "object",
      properties: {
        check: {
          type: "string",
          description: "The id from the Merge check line, such as risk or api-latency.",
        },
        score: { type: "number", description: "Integer from 1 through 5." },
        summary: { type: "string", description: "One sentence. Do not repeat the score." },
      },
      required: ["check", "score", "summary"],
    },
  },
  handler: reportMergeCheck,
};

module.exports = {
  tools: [REPORT_MERGE_CHECK],
};
