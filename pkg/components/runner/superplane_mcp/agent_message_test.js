"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const { recordAgentMessage } = require("../agent_message");

test("recordAgentMessage ignores Claude compact status", async () => {
  const previousFetch = global.fetch;
  const calls = [];
  global.fetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, text: async () => '{"status":"shown"}' };
  };
  try {
    const result = await recordAgentMessage("Compactions remaining: 0");
    assert.deepEqual(result, { status: "ignored" });
    assert.equal(calls.length, 0);
  } finally {
    global.fetch = previousFetch;
  }
});

test("recordAgentMessage publishes the final reply", async () => {
  const previousBaseURL = process.env.SUPERPLANE_BASE_URL;
  const previousToken = process.env.SUPERPLANE_RUN_TOKEN;
  const previousActivityID = process.env.SUPERPLANE_ACTIVITY_ID;
  const previousFetch = global.fetch;
  const calls = [];
  process.env.SUPERPLANE_BASE_URL = "https://superplane.example";
  process.env.SUPERPLANE_RUN_TOKEN = "planning-token";
  process.env.SUPERPLANE_ACTIVITY_ID = "activity-1";
  global.fetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, text: async () => '{"status":"shown"}' };
  };
  try {
    const result = await recordAgentMessage("  I found the retry seam.  ");
    assert.deepEqual(result, { status: "shown" });
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, "https://superplane.example/api/v1/runner/planning-sessions/agent-messages");
    assert.equal(calls[0].options.headers.Authorization, "Bearer planning-token");
    assert.deepEqual(JSON.parse(calls[0].options.body), {
      text: "I found the retry seam.",
      activity_id: "activity-1",
    });
  } finally {
    global.fetch = previousFetch;
    if (previousBaseURL === undefined) delete process.env.SUPERPLANE_BASE_URL;
    else process.env.SUPERPLANE_BASE_URL = previousBaseURL;
    if (previousToken === undefined) delete process.env.SUPERPLANE_RUN_TOKEN;
    else process.env.SUPERPLANE_RUN_TOKEN = previousToken;
    if (previousActivityID === undefined) delete process.env.SUPERPLANE_ACTIVITY_ID;
    else process.env.SUPERPLANE_ACTIVITY_ID = previousActivityID;
  }
});
