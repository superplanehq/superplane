"use strict";

function requiredEnv(name, env) {
  const value = String((env && env[name]) || "").trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

async function publishMarkdown(input, env = process.env, fetchImpl = fetch) {
  const body = String((input && input.body) || "").trim();
  if (!body) throw new Error("file is empty");
  const title = String((input && input.title) || "").trim();
  const key = String((input && input.key) || "").trim();
  const baseURL = requiredEnv("SUPERPLANE_BASE_URL", env).replace(/\/$/, "");
  const token = requiredEnv("SUPERPLANE_ARTIFACT_TOKEN", env);
  const response = await fetchImpl(`${baseURL}/api/v1/runner/artifacts/markdown`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ body, title, key }),
  });
  const text = await response.text();
  let result = {};
  try {
    result = text ? JSON.parse(text) : {};
  } catch (_error) {
    result = { message: text };
  }
  if (!response.ok) {
    throw new Error(result.message || result.error || text || `HTTP ${response.status}`);
  }
  return result;
}

module.exports = {
  publishMarkdown,
};
