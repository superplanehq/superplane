"use strict";

const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const path = require("node:path");
const test = require("node:test");

const { linkTarget, main, publishLink } = require("./publish_link");

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

test("publish-link command prints the attached url", async () => {
  let stdout = "";
  const code = await main(
    ["publish-link", "--url", "https://staging.example.com/pr-42", "--title", "Staging", "--key", "staging"],
    env,
    async () => ({
      ok: true,
      status: 201,
      text: async () => JSON.stringify({ url: "https://staging.example.com/pr-42" }),
    }),
    { write: (chunk) => { stdout += chunk; } },
    { write() {} },
  );
  assert.equal(code, 0);
  assert.equal(stdout, "https://staging.example.com/pr-42\n");
});

test("publish-link command requires a url", () => {
  const result = spawnSync(process.execPath, [path.join(__dirname, "publish_link.js"), "publish-link"], {
    encoding: "utf8",
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /url is required/);
});
