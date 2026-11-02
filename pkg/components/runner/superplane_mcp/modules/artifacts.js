"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const files = require("./files");
const { recordArtifacts } = require("./evidence");

const ARTIFACT_TYPES = ["markdown", "link", "file"];
const MAX_ARTIFACT_KEY_BYTES = 512;
// The backend stores a runner key as "<task id>:<key>" in a 512-byte column.
const MAX_RUNNER_ARTIFACT_KEY_BYTES = MAX_ARTIFACT_KEY_BYTES - 37;
const MAX_ARTIFACT_TITLE_LENGTH = 1024;

function exceedsCodePoints(value, max) {
  let count = 0;
  for (let index = 0; index < value.length; ) {
    count += 1;
    if (count > max) return true;
    const code = value.codePointAt(index);
    index += code > 0xffff ? 2 : 1;
  }
  return false;
}

function text(input, name) {
  return String((input && input[name]) || "").trim();
}

function artifactType(input, { required }) {
  const type = text(input, "type").toLowerCase();
  if (!type && !required) return "";
  if (!ARTIFACT_TYPES.includes(type)) throw new Error("type must be markdown, link, or file");
  return type;
}

function titleAndKey(input) {
  const title = text(input, "title");
  const key = text(input, "key");
  if (Buffer.byteLength(key) > MAX_RUNNER_ARTIFACT_KEY_BYTES) {
    throw new Error(`key must be at most ${MAX_RUNNER_ARTIFACT_KEY_BYTES} bytes`);
  }
  return { title, key };
}

function linkTarget(value) {
  const url = String(value || "").trim();
  if (!url) throw new Error("url is required");
  let parsed;
  try {
    parsed = new URL(url);
  } catch (_error) {
    throw new Error("url must be http or https");
  }
  if ((parsed.protocol !== "http:" && parsed.protocol !== "https:") || !parsed.host) {
    throw new Error("url must be http or https");
  }
  return url;
}

// Path rules for each tool name. get_artifact and add_artifact accept any
// file in the task directory. The previous names keep their directories.
const TASK_DIRECTORY_RULES = {};

function attachmentRules(env) {
  const attachments = path.join(files.taskDirectory(env), "attachments");
  return {
    root: attachments,
    base: attachments,
    rootLabel: "task attachments directory",
    rootError: "attachment path must be inside the task attachments directory",
    fileError: "attachment path must be a regular file",
    emptyError: "attachment file is empty",
  };
}

function evidenceRules(env) {
  return {
    root: path.join(files.taskDirectory(env), "evidence"),
    base: process.cwd(),
    rootLabel: "task evidence directory",
    rootError: "artifact path must be inside the task evidence directory",
    fileError: "artifact path must be a regular file",
    emptyError: "artifact file is empty",
  };
}

/**
 * readTaskFile returns one task file. An image returns as an MCP image, and
 * SuperPlane records its size and hash for add_artifact.
 * - sniffOnly: detect an image by its bytes only.
 * - imageError: reject a file that is not an image.
 */
function readTaskFile(rawPath, client, rules, { sniffOnly = false, imageError = "" } = {}) {
  const env = client.env;
  const file = files.resolveTaskFile(rawPath, env, rules);
  const byExtension = files.uploadContentType(file.filename);
  let mimeType = files.sniffImageMime(files.readHead(file.absolute, 16));
  if (!mimeType && !sniffOnly && files.SCREENSHOT_CONTENT_TYPES.has(byExtension)) mimeType = byExtension;
  if (!mimeType && imageError) throw new Error(imageError);

  const metadata = { path: path.relative(file.root, file.absolute), filename: file.filename, mimeType, sizeBytes: file.sizeBytes };
  if (!mimeType) {
    if (file.sizeBytes > files.MAX_ARTIFACT_BYTES) throw new Error(`file exceeds ${files.MAX_ARTIFACT_BYTES} bytes`);
    return { ...metadata, mimeType: byExtension || "application/octet-stream", sha256: files.fileSHA256(file.absolute) };
  }
  files.checkImageSize(file);
  const bytes = fs.readFileSync(file.absolute);
  metadata.sha256 = crypto.createHash("sha256").update(bytes).digest("hex");
  if (client.recordInspection !== false) files.recordInspection(file, metadata.sha256, env);
  return files.imageResult(bytes, mimeType, metadata);
}

function uploadableFile(rawPath, env, rules) {
  const file = files.resolveTaskFile(rawPath, env, rules);
  if (file.sizeBytes > files.MAX_ARTIFACT_BYTES) throw new Error(`artifact exceeds ${files.MAX_ARTIFACT_BYTES} bytes`);
  const contentType = files.uploadContentType(file.filename);
  if (!contentType) throw new Error("artifact file type is not supported; use PNG, JPEG, WebP, WebM, or MP4");
  return { ...file, contentType };
}

function uploadFile(file, title, key, client) {
  const headers = {
    "Content-Type": file.contentType,
    "Content-Length": String(file.sizeBytes),
    "Content-Disposition": `attachment; filename*=UTF-8''${encodeURIComponent(file.filename)}`,
  };
  if (title) headers["X-SuperPlane-Artifact-Title"] = encodeURIComponent(title);
  if (key) headers["X-SuperPlane-Artifact-Key"] = encodeURIComponent(key);
  return client.upload("/api/v1/runner/task/artifacts", headers, fs.createReadStream(file.absolute));
}

function markdownLabel(value) {
  return String(value || "Evidence")
    .replace(/\\/g, "\\\\")
    .replace(/\[/g, "\\[")
    .replace(/\]/g, "\\]");
}

async function uploadVideoWithPoster(file, poster, title, key, client) {
  const env = client.env;
  if (!files.SCREENSHOT_CONTENT_TYPES.has(poster.contentType)) {
    throw new Error("poster must be a PNG, JPEG, or WebP file");
  }
  files.requireInspectedScreenshot(poster, poster.contentType, env);

  const posterResult = await uploadFile(poster, title ? `${title} poster` : "Video poster", key ? `${key}:poster` : "", client);
  let videoResult;
  try {
    videoResult = await uploadFile(file, title, key, client);
  } catch (error) {
    recordArtifacts([posterResult], env, { partialReason: `Video upload failed: ${error.message}` });
    throw error;
  }
  const label = markdownLabel(title || file.filename);
  const linkedPoster = {
    ...posterResult,
    video_file_id: videoResult.file_id,
    markdown: `[![${label}](${posterResult.public_url})](${videoResult.public_url})`,
  };
  const linkedVideo = {
    ...videoResult,
    poster_file_id: posterResult.file_id,
    markdown: `[Watch video: ${label}](${videoResult.public_url})`,
  };
  recordArtifacts([linkedPoster, linkedVideo], env, { complete: true });
  return linkedVideo;
}

async function addFile(input, client, rules) {
  const env = client.env;
  const file = uploadableFile(input && input.path, env, rules);
  const { title, key } = titleAndKey(input);
  const posterPath = text(input, "posterPath");
  if (posterPath && !file.contentType.startsWith("video/")) {
    throw new Error("posterPath is only supported for video artifacts");
  }
  files.requireInspectedScreenshot(file, file.contentType, env);
  if (posterPath) {
    return uploadVideoWithPoster(file, uploadableFile(posterPath, env, rules), title, key, client);
  }
  const result = await uploadFile(file, title, key, client);
  recordArtifacts([result], env);
  return result;
}

async function addLink(input, client) {
  const url = linkTarget(input && input.url);
  const { title, key } = titleAndKey(input);
  return client.request("POST", "/api/v1/runner/task/links", { url, title, key });
}

async function addMarkdown(input, client) {
  const body = String((input && input.body) || "").trim();
  if (!body) throw new Error("body is required");
  const { title, key } = titleAndKey(input);
  return client.request("POST", "/api/v1/runner/task/markdown", { body, title, key });
}

async function addArtifact(input, client) {
  const type = artifactType(input, { required: true });
  if (exceedsCodePoints(text(input, "title"), MAX_ARTIFACT_TITLE_LENGTH)) {
    throw new Error("title must be at most 1024 characters");
  }
  switch (type) {
    case "markdown":
      return addMarkdown(input, client);
    case "link":
      return addLink(input, client);
    default:
      return addFile(input, client, TASK_DIRECTORY_RULES);
  }
}

async function listArtifacts(input, client) {
  const type = artifactType(input, { required: false });
  const query = type ? `?type=${encodeURIComponent(type)}` : "";
  return client.request("GET", `/api/v1/runner/task/artifacts${query}`);
}

async function getArtifact(input, client) {
  const id = text(input, "id");
  const key = text(input, "key");
  const filePath = text(input, "path");
  if ([id, key, filePath].filter(Boolean).length !== 1) throw new Error("give one of id, key, or path");
  if (filePath) return readTaskFile(filePath, client, TASK_DIRECTORY_RULES);
  const params = new URLSearchParams(id ? { id } : { key });
  return client.request("GET", `/api/v1/runner/task/artifact?${params.toString()}`);
}

async function removeArtifact(input, client) {
  const key = text(input, "key");
  const title = text(input, "title");
  if (!key && !title) throw new Error("key or title is required");
  if (Buffer.byteLength(key) > MAX_ARTIFACT_KEY_BYTES) throw new Error("key must be at most 512 bytes");
  if (exceedsCodePoints(title, MAX_ARTIFACT_TITLE_LENGTH)) throw new Error("title must be at most 1024 characters");
  const body = {};
  if (key) body.key = key;
  if (title) body.title = title;
  return client.request("DELETE", "/api/v1/runner/task/artifacts", body);
}

function inspectAttachment(input, client) {
  return readTaskFile(input && input.path, client, attachmentRules(client.env), {
    sniffOnly: true,
    imageError: "attachment must be a PNG, JPEG, GIF, or WebP file",
  });
}

function inspectScreenshot(input, client) {
  const rules = evidenceRules(client.env);
  const file = uploadableFile(input && input.path, client.env, rules);
  if (!files.SCREENSHOT_CONTENT_TYPES.has(file.contentType)) {
    throw new Error("screenshot must be a PNG, JPEG, or WebP file");
  }
  return readTaskFile(input.path, client, rules);
}

async function uploadArtifact(input, client) {
  return addFile({ ...input, key: "" }, client, evidenceRules(client.env));
}

async function publishLink(input, client) {
  return addLink(input, client);
}

const LIST_ARTIFACTS = {
  definition: {
    name: "list_artifacts",
    description:
      "List the artifacts on this run's task, newest first, without their content. On a planning run, this lists the session draft. Each artifact has id, key, type, title, url, and created_at.",
    inputSchema: {
      type: "object",
      properties: {
        type: { type: "string", enum: ARTIFACT_TYPES, description: "Optional. Keep only markdown, link, or file artifacts." },
      },
    },
  },
  handler: listArtifacts,
};

const GET_ARTIFACT = {
  definition: {
    name: "get_artifact",
    description:
      "Read one artifact of this run's task, or one file in the task directory. Give id or key for an artifact from list_artifacts, or path for a task file. A PNG, JPEG, GIF, or WebP file returns as an image that you can see. Use it for every image that a person attached to the task, and for every screenshot before you attach it with add_artifact. Screenshots in evidence/ are at most 5 MiB. Other images are at most 70 MiB.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "Artifact id from list_artifacts." },
        key: { type: "string", description: "Artifact key from list_artifacts." },
        path: {
          type: "string",
          description: "A file in the task directory, for example attachments/mock.png or evidence/checkout.png. A relative path starts at the task directory.",
        },
      },
    },
  },
  handler: getArtifact,
  aliases: [
    { name: "inspect_attachment", handler: inspectAttachment },
    { name: "inspect_screenshot", handler: inspectScreenshot },
  ],
};

const ADD_ARTIFACT = {
  definition: {
    name: "add_artifact",
    description:
      "Attach a markdown note, a link, or a file to this run's task. On a planning run, this edits the session draft. type is markdown (pass body), link (pass an http or https url), or file (pass path to a PNG, JPEG, WebP, WebM, or MP4 file in the task directory, at most 100 MiB). For a video, posterPath is an image that the task shows in place of the video. title is the label that people see. key replaces the artifact with the same key. Read each screenshot and poster with get_artifact first. A file result has a markdown snippet for a pull request description.",
    inputSchema: {
      type: "object",
      properties: {
        type: { type: "string", enum: ARTIFACT_TYPES },
        body: { type: "string", description: "Markdown, for type markdown." },
        url: { type: "string", description: "An http or https URL, for type link." },
        path: { type: "string", description: "A file in the task directory, for type file." },
        posterPath: { type: "string", description: "Optional. For a video: a PNG, JPEG, or WebP image that you read with get_artifact." },
        title: { type: "string", maxLength: 1024, description: "Optional. The label that people see. For a file, the default is the filename." },
        key: { type: "string", description: "Optional. A stable id. An artifact with the same key is replaced." },
      },
      required: ["type"],
    },
  },
  handler: addArtifact,
  aliases: [
    { name: "upload_artifact", handler: uploadArtifact },
    { name: "publish_link", handler: publishLink },
  ],
};

const REMOVE_ARTIFACT = {
  definition: {
    name: "remove_artifact",
    description:
      "Remove one artifact from this run's task. On a planning run, this edits the session draft. Pass key or title. A title matches the newest artifact that has no key. A closed or archived task cannot change.",
    inputSchema: {
      type: "object",
      properties: {
        key: { type: "string", maxLength: 512, description: "Artifact key. At most 512 bytes." },
        title: {
          type: "string",
          maxLength: 1024,
          description: "Artifact title, when the artifact has no key. At most 1024 characters.",
        },
      },
    },
  },
  handler: removeArtifact,
};

module.exports = {
  tools: [LIST_ARTIFACTS, GET_ARTIFACT, ADD_ARTIFACT, REMOVE_ARTIFACT],
};
