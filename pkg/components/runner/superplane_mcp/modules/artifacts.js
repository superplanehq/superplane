"use strict";

const MAX_ARTIFACT_KEY_BYTES = 512;
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

async function removeArtifact(input, client) {
  const key = String((input && input.key) || "").trim();
  const title = String((input && input.title) || "").trim();
  if (!key && !title) throw new Error("key or title is required");
  if (Buffer.byteLength(key) > MAX_ARTIFACT_KEY_BYTES) throw new Error("key must be at most 512 bytes");
  if (exceedsCodePoints(title, MAX_ARTIFACT_TITLE_LENGTH)) throw new Error("title must be at most 1024 characters");
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
  tools: [REMOVE_ARTIFACT],
};
