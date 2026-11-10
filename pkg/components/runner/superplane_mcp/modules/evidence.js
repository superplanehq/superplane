"use strict";

const fs = require("node:fs");
const path = require("node:path");

const { taskDirectory, writeJSONAtomically } = require("./files");

const ATTEMPT_TYPES = new Set(["preview", "playwright", "upload"]);
const REQUIRED_ATTEMPT_TYPES = ["preview", "playwright"];

function manifestPath(env) {
  return path.join(taskDirectory(env), "visual-evidence.json");
}

function readManifest(env) {
  try {
    const parsed = JSON.parse(fs.readFileSync(manifestPath(env), "utf8"));
    const result = {
      status: String(parsed.status || "not_applicable"),
      reason: String(parsed.reason || ""),
      artifacts: Array.isArray(parsed.artifacts) ? parsed.artifacts : [],
    };
    if (Array.isArray(parsed.attempts)) result.attempts = parsed.attempts;
    return result;
  } catch (_error) {
    return { status: "not_applicable", reason: "", artifacts: [] };
  }
}

function writeManifest(manifest, env) {
  writeJSONAtomically(manifestPath(env), manifest);
}

// recordArtifacts adds uploaded files to the evidence manifest. The visual
// evidence app reads the manifest after the run. A partial manifest stays
// partial unless the upload completes a video and its poster. A keyed upload
// keeps its artifact_id and gets a new file_id, so match on both.
function recordArtifacts(results, env, { partialReason = "", complete = false } = {}) {
  const manifest = readManifest(env);
  const fileIDs = new Set(results.map((item) => item.file_id));
  const artifactIDs = new Set(results.map((item) => item.artifact_id).filter(Boolean));
  const artifacts = manifest.artifacts.filter(
    (item) => !fileIDs.has(item.file_id) && !artifactIDs.has(item.artifact_id),
  );
  artifacts.push(...results);
  if (partialReason) {
    writeManifest({ status: "partial", reason: partialReason, artifacts }, env);
    return;
  }
  const partial = manifest.status === "partial" && !complete;
  writeManifest({ status: partial ? "partial" : "captured", reason: partial ? manifest.reason : "", artifacts }, env);
}

function normalizeEvidenceAttempts(value) {
  if (!Array.isArray(value) || value.length < 2) {
    throw new Error("attempts is required and must include one preview and one playwright attempt");
  }
  const attempts = value.map((attempt, index) => {
    if (!attempt || typeof attempt !== "object" || Array.isArray(attempt)) {
      throw new Error(`attempts[${index}] must be an object`);
    }
    const type = String(attempt.type || "").trim();
    const command = String(attempt.command || "").trim();
    const outcome = String(attempt.outcome || "").trim();
    if (!ATTEMPT_TYPES.has(type)) throw new Error(`attempts[${index}].type must be preview, playwright, or upload`);
    if (!command) throw new Error(`attempts[${index}].command is required`);
    if (!outcome) throw new Error(`attempts[${index}].outcome is required`);
    return { type, command, outcome };
  });
  const attemptTypes = new Set(attempts.map((attempt) => attempt.type));
  if (REQUIRED_ATTEMPT_TYPES.some((type) => !attemptTypes.has(type))) {
    throw new Error("attempts must include one preview and one playwright attempt");
  }
  return attempts;
}

function reportVisualEvidenceUnavailable(input, client) {
  const env = client.env;
  const reason = String((input && input.reason) || "").trim();
  if (!reason) throw new Error("reason is required");
  const attempts = normalizeEvidenceAttempts(input && input.attempts);
  const manifest = readManifest(env);
  let status = "unavailable";
  if (manifest.artifacts.length > 0) status = manifest.status === "partial" ? "partial" : "captured";
  const result = { status, reason, attempts, artifacts: manifest.artifacts };
  writeManifest(result, env);
  return result;
}

// prepare writes the evidence manifest at startup, so a run that captures
// nothing still leaves a readable file.
function prepare(client) {
  writeManifest(readManifest(client.env), client.env);
}

const REPORT_VISUAL_EVIDENCE_UNAVAILABLE = {
  definition: {
    name: "report_visual_evidence_unavailable",
    description:
      "Record why required visual evidence could not be produced after trying an isolated preview and a Playwright capture.",
    inputSchema: {
      type: "object",
      properties: {
        reason: { type: "string" },
        attempts: {
          type: "array",
          description: "Concrete preview and Playwright attempts. Add an upload attempt when upload fails.",
          items: {
            type: "object",
            properties: {
              type: { type: "string", enum: ["preview", "playwright", "upload"] },
              command: { type: "string", minLength: 1 },
              outcome: { type: "string", minLength: 1 },
            },
            required: ["type", "command", "outcome"],
            additionalProperties: false,
          },
          minItems: 2,
        },
      },
      required: ["reason", "attempts"],
    },
  },
  handler: reportVisualEvidenceUnavailable,
};

module.exports = {
  readManifest,
  recordArtifacts,
  prepare,
  tools: [REPORT_VISUAL_EVIDENCE_UNAVAILABLE],
};
