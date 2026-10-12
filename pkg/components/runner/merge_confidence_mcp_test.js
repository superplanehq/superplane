"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const { reportMergeCheck } = require("./merge_confidence_mcp");

test("reportMergeCheck posts the score and summary", async () => {
  const previousBaseURL = process.env.SUPERPLANE_BASE_URL;
  const previousToken = process.env.SUPERPLANE_MERGE_CONFIDENCE_TOKEN;
  const previousFetch = global.fetch;
  const calls = [];
  process.env.SUPERPLANE_BASE_URL = "https://superplane.example";
  process.env.SUPERPLANE_MERGE_CONFIDENCE_TOKEN = "runner-token";
  global.fetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, text: async () => '{"status":"reported"}' };
  };

  try {
    const result = await reportMergeCheck({
      check: " Risk ",
      score: 4,
      summary: "  Higher risk because the pull request raises the limit.  ",
    });
    assert.deepEqual(result, { status: "reported" });
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/merge-confidence/checks");
    assert.equal(calls[0].options.headers.Authorization, "Bearer runner-token");
    assert.deepEqual(JSON.parse(calls[0].options.body), {
      check: "risk",
      score: 4,
      summary: "Higher risk because the pull request raises the limit.",
    });
  } finally {
    global.fetch = previousFetch;
    if (previousBaseURL === undefined) delete process.env.SUPERPLANE_BASE_URL;
    else process.env.SUPERPLANE_BASE_URL = previousBaseURL;
    if (previousToken === undefined) delete process.env.SUPERPLANE_MERGE_CONFIDENCE_TOKEN;
    else process.env.SUPERPLANE_MERGE_CONFIDENCE_TOKEN = previousToken;
  }
});

test("reportMergeCheck returns the server error", async () => {
  const previousFetch = global.fetch;
  process.env.SUPERPLANE_BASE_URL = "https://superplane.example";
  process.env.SUPERPLANE_MERGE_CONFIDENCE_TOKEN = "runner-token";
  global.fetch = async () => ({
    ok: false,
    status: 400,
    text: async () => "merge confidence check is off: performance\n",
  });

  try {
    await assert.rejects(
      () => reportMergeCheck({ check: "performance", score: 5, summary: "No practice applies." }),
      /performance/,
    );
  } finally {
    global.fetch = previousFetch;
  }
});
