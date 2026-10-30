"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const MAX_ATTACHMENT_BYTES = 70 * 1024 * 1024;

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

async function proposeUpdate(input, client) {
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

function sniffImageMime(bytes) {
  if (bytes.length >= 8 && bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4e && bytes[3] === 0x47) {
    return "image/png";
  }
  if (bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) {
    return "image/jpeg";
  }
  if (bytes.length >= 6) {
    const header = bytes.toString("ascii", 0, 6);
    if (header === "GIF87a" || header === "GIF89a") return "image/gif";
  }
  if (bytes.length >= 12 && bytes.toString("ascii", 0, 4) === "RIFF" && bytes.toString("ascii", 8, 12) === "WEBP") {
    return "image/webp";
  }
  return "";
}

function resolveAttachmentFile(inputPath, env) {
  const candidate = String(inputPath || "").trim();
  if (!candidate) throw new Error("path is required");
  const taskDirValue = String(env.SUPERPLANE_TASK_DIR || "").trim();
  if (!taskDirValue) throw new Error("SUPERPLANE_TASK_DIR is required");
  const taskDir = path.resolve(taskDirValue);
  const attachmentsDir = path.join(taskDir, "attachments");
  fs.mkdirSync(attachmentsDir, { recursive: true });
  const attachmentsInfo = fs.lstatSync(attachmentsDir);
  if (attachmentsInfo.isSymbolicLink() || !attachmentsInfo.isDirectory()) {
    throw new Error("task attachments directory must be a regular directory");
  }
  const attachmentsRoot = fs.realpathSync(attachmentsDir);
  const expanded = candidate
    .replace(/\$\{SUPERPLANE_TASK_DIR\}/g, taskDir)
    .replace(/\$SUPERPLANE_TASK_DIR/g, taskDir);
  const absolute = path.isAbsolute(expanded) ? path.resolve(expanded) : path.resolve(attachmentsDir, expanded);
  const fileInfo = fs.lstatSync(absolute);
  if (fileInfo.isSymbolicLink() || !fileInfo.isFile()) {
    throw new Error("attachment path must be a regular file");
  }
  const resolved = fs.realpathSync(absolute);
  if (resolved !== attachmentsRoot && !resolved.startsWith(`${attachmentsRoot}${path.sep}`)) {
    throw new Error("attachment path must be inside the task attachments directory");
  }
  if (fileInfo.size <= 0) throw new Error("attachment file is empty");
  if (fileInfo.size > MAX_ATTACHMENT_BYTES) {
    throw new Error(`attachment exceeds ${MAX_ATTACHMENT_BYTES} bytes`);
  }
  return { absolute: resolved, filename: path.basename(resolved), root: attachmentsRoot };
}

function inspectAttachment(input, client) {
  const file = resolveAttachmentFile(input && input.path, client.env);
  const bytes = fs.readFileSync(file.absolute);
  const mimeType = sniffImageMime(bytes);
  if (!mimeType) throw new Error("attachment must be a PNG, JPEG, GIF, or WebP file");
  const metadata = {
    path: path.relative(file.root, file.absolute),
    filename: file.filename,
    mimeType,
    sizeBytes: bytes.length,
    sha256: crypto.createHash("sha256").update(bytes).digest("hex"),
  };
  return {
    content: [
      { type: "text", text: JSON.stringify(metadata) },
      { type: "image", data: bytes.toString("base64"), mimeType },
    ],
    structuredContent: metadata,
  };
}

const scoreSchema = {
  type: "object",
  properties: {
    score: { type: "number", description: "Integer from 1 through 3." },
    summary: { type: "string", description: "One sentence." },
  },
  required: ["score", "summary"],
};

const PROPOSE_UPDATE = {
  definition: {
    name: "propose_update",
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
  handler: proposeUpdate,
};

const INSPECT_ATTACHMENT = {
  definition: {
    name: "inspect_attachment",
    description:
      "Inspect a user image from the task attachments directory. Returns the image so you can see it. Call this for every PNG, JPEG, GIF, or WebP user image. Do not use OCR or the file command.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" } },
      required: ["path"],
    },
  },
  handler: inspectAttachment,
};

module.exports = {
  MAX_ATTACHMENT_BYTES,
  tools: [PROPOSE_UPDATE, INSPECT_ATTACHMENT],
};
