"use strict";

const PUBLISH_LINK_TOOL = {
  name: "publish_link",
  description:
    "Attach an http or https link to this task. title is the label people see. key replaces the same link on a later run.",
  inputSchema: {
    type: "object",
    properties: {
      url: { type: "string" },
      title: { type: "string" },
      key: { type: "string" },
    },
    required: ["url"],
  },
};

function requiredEnv(name, env) {
  const value = String((env && env[name]) || "").trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
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

async function publishLink(input, env = process.env, fetchImpl = fetch) {
  const url = linkTarget(input && input.url);
  const title = String((input && input.title) || "").trim();
  const key = String((input && input.key) || "").trim();
  const baseURL = requiredEnv("SUPERPLANE_BASE_URL", env).replace(/\/$/, "");
  const token = requiredEnv("SUPERPLANE_ARTIFACT_TOKEN", env);
  const response = await fetchImpl(`${baseURL}/api/v1/runner/artifacts/links`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ url, title, key }),
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
  PUBLISH_LINK_TOOL,
  linkTarget,
  publishLink,
};
