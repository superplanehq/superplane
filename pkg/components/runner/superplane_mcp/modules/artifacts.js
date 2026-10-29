"use strict";

async function removeArtifact(input, client) {
  const key = String((input && input.key) || "").trim();
  const title = String((input && input.title) || "").trim();
  if (!key && !title) throw new Error("key or title is required");
  const body = {};
  if (key) body.key = key;
  if (title) body.title = title;
  return client.request("DELETE", "/api/v1/runner/task/artifacts", body);
}

const REMOVE_ARTIFACT = {
  definition: {
    name: "remove_artifact",
    description:
      "Remove one artifact from this run's task. On a planning run, this edits the session draft. Pass key or title. A title matches the newest artifact that has no key. A closed or archived task cannot change.",
    inputSchema: {
      type: "object",
      properties: {
        key: { type: "string", description: "Artifact key." },
        title: { type: "string", description: "Artifact title, when the artifact has no key." },
      },
    },
  },
  handler: removeArtifact,
};

module.exports = {
  tools: [REMOVE_ARTIFACT],
};
