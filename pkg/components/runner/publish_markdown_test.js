"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");

const { publishMarkdown } = require("./publish_markdown");

const env = {
  SUPERPLANE_BASE_URL: "https://app.example",
  SUPERPLANE_RUNNER_TOKEN: "token",
};

test("publishMarkdown posts the body, title, and key", async () => {
  let request;
  const result = await publishMarkdown(
    {
      body: "Email: demo@operately.dev\n\nPassword: preview-demo-1\n",
      title: "Preview login",
      key: "preview-login",
    },
    env,
    async (url, options) => {
      request = { url, options };
      return {
        ok: true,
        status: 201,
        text: async () =>
          JSON.stringify({
            artifact_id: "art-1",
            title: "Preview login",
            key: "preview-login",
          }),
      };
    },
  );

  assert.equal(request.url, "https://app.example/api/v1/runner/task/markdown");
  assert.equal(request.options.headers.Authorization, "Bearer token");
  assert.deepEqual(JSON.parse(request.options.body), {
    body: "Email: demo@operately.dev\n\nPassword: preview-demo-1",
    title: "Preview login",
    key: "preview-login",
  });
  assert.equal(result.artifact_id, "art-1");
});

test("publishMarkdown rejects an empty file before the request", async () => {
  await assert.rejects(
    publishMarkdown({ body: "  \n" }, env, async () => {
      throw new Error("fetch must not run");
    }),
    /file is empty/,
  );
});
