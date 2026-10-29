"use strict";

const { resolveArtifactFile, uploadFile } = require("./task_artifact_mcp");

function artifactFile(filePath, env) {
  try {
    return resolveArtifactFile(filePath, env);
  } catch (error) {
    if (error && error.code === "ENOENT") {
      throw new Error(`cannot read ${filePath}`);
    }
    throw error;
  }
}

async function publishFile(input, env = process.env, fetchImpl = fetch) {
  const file = artifactFile(input && input.file, env);
  const title = String((input && input.title) || "").trim();
  const key = String((input && input.key) || "").trim();
  const posterPath = String((input && input.poster) || "").trim();
  if (posterPath && !file.contentType.startsWith("video/")) {
    throw new Error("poster is only supported for video");
  }
  if (posterPath) {
    const poster = artifactFile(posterPath, env);
    if (!poster.contentType.startsWith("image/")) {
      throw new Error("poster must be a PNG, JPEG, or WebP file");
    }
    const posterTitle = title ? `${title} poster` : "Video poster";
    const posterKey = key ? `${key}-poster` : "";
    await uploadFile(poster, posterTitle, env, fetchImpl, posterKey);
  }
  return uploadFile(file, title || file.filename, env, fetchImpl, key);
}

module.exports = {
  publishFile,
};
