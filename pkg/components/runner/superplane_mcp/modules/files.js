"use strict";

/**
 * Task directory files for get_artifact and add_artifact. A file must be a
 * regular file inside SUPERPLANE_TASK_DIR. Symbolic links are rejected.
 */

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const MAX_ATTACHMENT_BYTES = 70 * 1024 * 1024;
const MAX_INSPECTABLE_SCREENSHOT_BYTES = 5 * 1024 * 1024;
const MAX_ARTIFACT_BYTES = 100 * 1024 * 1024;

const UPLOAD_CONTENT_TYPES = new Map([
  [".png", "image/png"],
  [".jpg", "image/jpeg"],
  [".jpeg", "image/jpeg"],
  [".webp", "image/webp"],
  [".webm", "video/webm"],
  [".mp4", "video/mp4"],
]);
const SCREENSHOT_CONTENT_TYPES = new Set(["image/png", "image/jpeg", "image/webp"]);

function taskDirectory(env) {
  const taskDir = String(env.SUPERPLANE_TASK_DIR || "").trim();
  if (!taskDir) throw new Error("SUPERPLANE_TASK_DIR is required");
  return path.resolve(taskDir);
}

// regularDirectory creates the directory when it is missing and returns its
// real path. A symbolic link cannot stand in for a task directory.
function regularDirectory(directory, label) {
  fs.mkdirSync(directory, { recursive: true });
  const info = fs.lstatSync(directory);
  if (info.isSymbolicLink() || !info.isDirectory()) {
    throw new Error(`${label} must be a regular directory`);
  }
  return fs.realpathSync(directory);
}

function isInside(file, root) {
  return file === root || file.startsWith(`${root}${path.sep}`);
}

/**
 * resolveTaskFile finds a file for a tool call.
 * - base: the directory for a relative path. The default is the task directory.
 * - root: the directory that must hold the file. The default is the task directory.
 * - rootError: the error when the file is outside root.
 */
function resolveTaskFile(inputPath, env, options = {}) {
  const candidate = String(inputPath || "").trim();
  if (!candidate) throw new Error("path is required");
  const taskDir = taskDirectory(env);
  const root = regularDirectory(options.root || taskDir, options.rootLabel || "task directory");
  const expanded = candidate
    .replace(/\$\{SUPERPLANE_TASK_DIR\}/g, taskDir)
    .replace(/\$SUPERPLANE_TASK_DIR/g, taskDir);
  const base = options.base === undefined ? taskDir : options.base;
  const absolute = path.isAbsolute(expanded) ? path.resolve(expanded) : path.resolve(base, expanded);
  let info;
  try {
    info = fs.lstatSync(absolute);
  } catch (_error) {
    throw new Error(`file not found: ${candidate}`);
  }
  if (info.isSymbolicLink() || !info.isFile()) {
    throw new Error(options.fileError || "path must be a regular file");
  }
  const resolved = fs.realpathSync(absolute);
  if (!isInside(resolved, root)) {
    throw new Error(options.rootError || "path must be inside the task directory");
  }
  if (info.size <= 0) throw new Error(options.emptyError || "file is empty");
  const realTaskDir = fs.realpathSync(taskDir);
  return {
    absolute: resolved,
    filename: path.basename(resolved),
    sizeBytes: info.size,
    root,
    taskPath: path.relative(realTaskDir, resolved),
    inEvidence: isInside(resolved, path.join(realTaskDir, "evidence")),
  };
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

function uploadContentType(filename) {
  return UPLOAD_CONTENT_TYPES.get(path.extname(filename).toLowerCase()) || "";
}

function fileSHA256(filePath) {
  return crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex");
}

function readHead(filePath, length) {
  const handle = fs.openSync(filePath, "r");
  try {
    const buffer = Buffer.alloc(length);
    const read = fs.readSync(handle, buffer, 0, length, 0);
    return buffer.subarray(0, read);
  } finally {
    fs.closeSync(handle);
  }
}

function inspectionsPath(env) {
  return path.join(taskDirectory(env), "visual-evidence-inspections.json");
}

function readInspections(env) {
  try {
    const parsed = JSON.parse(fs.readFileSync(inspectionsPath(env), "utf8"));
    return Array.isArray(parsed.inspections) ? parsed.inspections : [];
  } catch (_error) {
    return [];
  }
}

function writeJSONAtomically(destination, value) {
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  const temporary = `${destination}.${process.pid}.${crypto.randomUUID()}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o600 });
  fs.renameSync(temporary, destination);
}

// recordInspection keeps the size and hash of each image that the agent saw.
// add_artifact compares them before it uploads a screenshot or a poster.
function recordInspection(file, sha256, env) {
  const inspections = readInspections(env).filter((entry) => entry.path !== file.taskPath);
  inspections.push({ path: file.taskPath, sizeBytes: file.sizeBytes, sha256 });
  writeJSONAtomically(inspectionsPath(env), { inspections });
}

function requireInspectedScreenshot(file, contentType, env) {
  if (!SCREENSHOT_CONTENT_TYPES.has(contentType)) return;
  const inspection = readInspections(env).find((entry) => entry.path === file.taskPath);
  if (!inspection) throw new Error("call get_artifact before add_artifact for every screenshot");
  if (inspection.sizeBytes !== file.sizeBytes || inspection.sha256 !== fileSHA256(file.absolute)) {
    throw new Error("screenshot changed after get_artifact; read the final file again before add_artifact");
  }
}

// imageResult returns the image as an MCP image block. The base64 data stays
// out of structuredContent.
function imageResult(bytes, mimeType, metadata) {
  return {
    content: [
      { type: "text", text: JSON.stringify(metadata) },
      { type: "image", data: bytes.toString("base64"), mimeType },
    ],
    structuredContent: metadata,
  };
}

// Screenshots in evidence/ stay small, so the model can read them.
function checkImageSize(file) {
  if (file.inEvidence && file.sizeBytes > MAX_INSPECTABLE_SCREENSHOT_BYTES) {
    throw new Error(
      `screenshot exceeds ${MAX_INSPECTABLE_SCREENSHOT_BYTES} bytes; capture a smaller or more focused image`,
    );
  }
  if (file.sizeBytes > MAX_ATTACHMENT_BYTES) throw new Error(`image exceeds ${MAX_ATTACHMENT_BYTES} bytes`);
}

module.exports = {
  MAX_ARTIFACT_BYTES,
  MAX_ATTACHMENT_BYTES,
  MAX_INSPECTABLE_SCREENSHOT_BYTES,
  SCREENSHOT_CONTENT_TYPES,
  checkImageSize,
  fileSHA256,
  imageResult,
  readHead,
  recordInspection,
  regularDirectory,
  requireInspectedScreenshot,
  resolveTaskFile,
  sniffImageMime,
  taskDirectory,
  uploadContentType,
  writeJSONAtomically,
};
