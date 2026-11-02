"use strict";

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

async function publishLink(input, client) {
  return client.request("POST", "/api/v1/runner/task/links", {
    url: linkTarget(input && input.url),
    title: String((input && input.title) || "").trim(),
    key: String((input && input.key) || "").trim(),
  });
}

const PUBLISH_LINK = {
  definition: {
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
  },
  handler: publishLink,
};

module.exports = {
  tools: [PUBLISH_LINK],
};
