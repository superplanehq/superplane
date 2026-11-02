"use strict";

const fs = require("node:fs");

function surveyQuestions(input) {
  const raw = input && Array.isArray(input.questions) ? input.questions : [];
  return raw
    .map((question) => ({
      prompt: String((question && question.prompt) || "").trim(),
      options: Array.isArray(question && question.options)
        ? question.options.map((option) => String(option || "").trim()).filter(Boolean)
        : [],
      ...(question && question.recommended_option
        ? { recommended_option: String(question.recommended_option) }
        : {}),
    }))
    .filter((question) => question.prompt);
}

function scoreField(input, name) {
  const raw = input && input[name];
  if (!raw || typeof raw !== "object") throw new Error(`${name} is required`);
  const score = Number(raw.score);
  if (!Number.isFinite(score)) throw new Error(`${name} score is required`);
  const summary = String(raw.summary || "").trim();
  if (!summary) throw new Error(`${name} summary is required`);
  return { score, summary };
}

function unwrapMarkdown(value) {
  const text = String(value || "").trim();
  if (!text.startsWith('"')) return text;
  try {
    const decoded = JSON.parse(text);
    if (typeof decoded === "string" && decoded.trim()) return decoded;
  } catch (_err) {
    // The value is not a JSON string.
  }
  return text;
}

// The planning exit graph reads the spec from this file.
function writeSpecOutput(spec, env) {
  try {
    fs.writeFileSync(String(env.SUPERPLANE_ANALYSIS_SPEC_FILE || "/tmp/spec.md"), spec);
  } catch (_err) {
    // The backend already saved the spec.
  }
}

async function proposePlan(input, client) {
  const body = {};
  if (input && input.scores) {
    body.scores = {
      clarity: scoreField(input.scores, "clarity"),
      complexity: scoreField(input.scores, "complexity"),
      verifiability: scoreField(input.scores, "verifiability"),
    };
  }
  const spec = unwrapMarkdown(input && input.spec);
  if (spec) body.spec = spec;
  const questions = surveyQuestions(input && input.survey);
  if (questions.length > 0) body.survey = { questions };
  if (!body.scores && !body.spec && !body.survey) {
    throw new Error("scores, spec, or survey is required");
  }
  const result = await client.request("PUT", "/api/v1/runner/task/plan", body);
  if (body.spec) writeSpecOutput(body.spec, client.env);
  return result;
}

const scoreSchema = {
  type: "object",
  properties: {
    score: { type: "number", description: "Integer from 1 through 3." },
    summary: { type: "string", description: "One sentence." },
  },
  required: ["score", "summary"],
};

const PROPOSE_PLAN = {
  definition: {
    name: "propose_plan",
    description:
      "Publish scores, the specification, and an optional survey in one call. Pass scores as clarity, complexity, and verifiability. Each score is an integer from 1 through 3 with a one-sentence summary. Pass spec as the full markdown body. Pass survey only when you ask a question. The first plan turn must include scores.",
    inputSchema: {
      type: "object",
      properties: {
        scores: {
          type: "object",
          properties: { clarity: scoreSchema, complexity: scoreSchema, verifiability: scoreSchema },
          required: ["clarity", "complexity", "verifiability"],
        },
        spec: { type: "string", description: "Full specification markdown." },
        survey: {
          type: "object",
          properties: {
            questions: {
              type: "array",
              minItems: 1,
              items: {
                type: "object",
                properties: {
                  prompt: { type: "string" },
                  options: { type: "array", items: { type: "string" } },
                  recommended_option: {
                    type: "string",
                    description: "Exact text of the recommended option. Omit when context is not enough.",
                  },
                },
                required: ["prompt", "options"],
              },
            },
          },
          required: ["questions"],
        },
      },
    },
  },
  handler: proposePlan,
  aliases: [{ name: "propose_update" }],
};

module.exports = {
  tools: [PROPOSE_PLAN],
};
