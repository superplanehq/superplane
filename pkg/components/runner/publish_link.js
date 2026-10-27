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

function usage() {
  return "usage: superplane publish-link --url URL [--title TITLE] [--key KEY]";
}

function parseArgs(argv) {
  const args = Array.isArray(argv) ? argv : [];
  if (args[0] !== "publish-link") {
    throw new Error(usage());
  }
  const input = {};
  for (let index = 1; index < args.length; index += 1) {
    const flag = args[index];
    const value = args[index + 1];
    if (!value || value.startsWith("--")) {
      throw new Error(usage());
    }
    if (flag === "--url") input.url = value;
    else if (flag === "--title") input.title = value;
    else if (flag === "--key") input.key = value;
    else throw new Error(usage());
    index += 1;
  }
  return input;
}

async function main(argv, env = process.env, fetchImpl = fetch, stdout = process.stdout, stderr = process.stderr) {
  try {
    const result = await publishLink(parseArgs(argv), env, fetchImpl);
    const url = String(result.url || "").trim();
    stdout.write(`${url}\n`);
  } catch (error) {
    stderr.write(`${error && error.message ? error.message : error}\n`);
    return 1;
  }
  return 0;
}

if (require.main === module) {
  main(process.argv.slice(2)).then((code) => process.exit(code));
}

module.exports = {
  PUBLISH_LINK_TOOL,
  linkTarget,
  main,
  parseArgs,
  publishLink,
};
