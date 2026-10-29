"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");

const { linkTarget, publishLink } = require("./publish_link");

const env = {
  SUPERPLANE_BASE_URL: "https://app.example",
  SUPERPLANE_ARTIFACT_TOKEN: "token",
};

test("publishLink posts the url, title, and key", async () => {
  let request;
  const result = await publishLink(
    { url: "https://staging.example.com/pr-42", title: "Staging", key: "staging" },
    env,
    async (url, options) => {
      request = { url, options };
      return {
        ok: true,
        status: 201,
        text: async () =>
          JSON.stringify({
            artifact_id: "art-1",
            url: "https://staging.example.com/pr-42",
            title: "Staging",
            key: "staging",
          }),
      };
    },
  );

  assert.equal(request.url, "https://app.example/api/v1/runner/artifacts/links");
  assert.equal(request.options.headers.Authorization, "Bearer token");
  assert.deepEqual(JSON.parse(request.options.body), {
    url: "https://staging.example.com/pr-42",
    title: "Staging",
    key: "staging",
  });
  assert.equal(result.artifact_id, "art-1");
});

test("publishLink rejects a non-http url before the request", async () => {
  await assert.rejects(
    publishLink({ url: "javascript:alert(1)" }, env, async () => {
      throw new Error("fetch must not run");
    }),
    /http or https/,
  );
  assert.throws(() => linkTarget("https://"), /http or https/);
});
