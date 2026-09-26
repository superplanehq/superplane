"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const {
  FOLLOW_UP_CMD_INDEX_BASE,
  MAX_ATTACHMENT_BYTES,
  interpretWaitResponse,
  isAllowedSignedDownloadURL,
  materializeFollowUpAttachments,
  nextAction,
  persistAnalysisContinuation,
  runLoop,
  runPromptFile,
  safeWaitRequest,
  signedFileURLs,
} = require("./follow_up_loop.js");

const CLOUDFLARE_502 = {
  error: "An error occurred with your request. Please try again.",
  retryable: true,
  retry_after: 60,
};

test("waits while SuperPlane has no event", () => {
  assert.deepEqual(nextAction({ status: "pending" }), { type: "wait" });
  assert.deepEqual(nextAction({}), { type: "wait" });
});

test("exits when the session ends", () => {
  assert.deepEqual(nextAction({ status: "ended" }), { type: "exit", code: 0 });
});

test("turns a user message into the next prompt", () => {
  assert.deepEqual(nextAction({ status: "message", text: " Add a Size field " }), {
    type: "prompt",
    text: "Add a Size field",
  });
});

test("ignores an empty user message", () => {
  assert.deepEqual(nextAction({ status: "message", text: "   " }), { type: "wait" });
});

test("runLoop runs the user prompt then exits on ended", async () => {
  const prompts = [];
  const results = [{ status: "pending" }, { status: "message", text: "Add color" }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async (text) => {
      prompts.push(text);
      return 0;
    },
    sleep: async () => {},
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(prompts, ["Add color"]);
});

test("persistAnalysisContinuation writes a wait continuation for the next rewind", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-continuation-"));
  persistAnalysisContinuation(dir, {
    continuation: "Continue this SuperPlane analysis session.",
  });
  assert.equal(
    fs.readFileSync(path.join(dir, "analysis_continuation.md"), "utf8"),
    "Continue this SuperPlane analysis session.\n",
  );
  persistAnalysisContinuation(dir, { status: "message", text: "ok" });
  assert.equal(fs.existsSync(path.join(dir, "analysis_continuation.md")), false);
});

test("runLoop writes wait continuation before the follow-up prompt", async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-loop-continuation-"));
  const results = [
    {
      status: "message",
      text: "Narrow the spec",
      continuation: "Continue this SuperPlane analysis session.",
    },
    { status: "ended" },
  ];
  await runLoop({
    waitOnce: async () => results.shift(),
    taskDir: dir,
    runPrompt: async () => 0,
    sleep: async () => {},
    writeLiveLogRecord: () => {},
  });
  assert.equal(
    fs.readFileSync(path.join(dir, "analysis_continuation.md"), "utf8"),
    "Continue this SuperPlane analysis session.\n",
  );
});

test("interpretWaitResponse treats a Cloudflare 502 as idle pending", () => {
  const got = interpretWaitResponse(502, CLOUDFLARE_502);
  assert.deepEqual(got, { status: "pending" });
});

test("interpretWaitResponse retries 503, 504, and 429 as idle pending", () => {
  assert.deepEqual(interpretWaitResponse(503, {}), { status: "pending" });
  assert.deepEqual(interpretWaitResponse(504, {}), { status: "pending" });
  assert.deepEqual(interpretWaitResponse(429, { retry_after: 12 }), { status: "pending" });
});

test("interpretWaitResponse treats a Cloudflare error body as idle pending", () => {
  const got = interpretWaitResponse(500, { cloudflare_error: true, retry_after: 30 });
  assert.deepEqual(got, { status: "pending" });
});

test("interpretWaitResponse ignores retry_after on a transient wait", () => {
  const got = interpretWaitResponse(502, { retryable: true, retry_after: 120 });
  assert.deepEqual(got, { status: "pending" });
});

test("interpretWaitResponse ends on 409", () => {
  assert.deepEqual(interpretWaitResponse(409, { message: "conflict" }), { status: "ended" });
});

test("interpretWaitResponse throws on 401", () => {
  assert.throws(() => interpretWaitResponse(401, { message: "unauthorized" }), /unauthorized/);
});

test("interpretWaitResponse retries 404 and 403 as idle pending", () => {
  assert.deepEqual(interpretWaitResponse(404, { message: "ngrok" }), { status: "pending" });
  assert.deepEqual(interpretWaitResponse(404, {}, "ERR_NGROK_3200"), { status: "pending" });
  assert.deepEqual(interpretWaitResponse(403, { message: "forbidden" }), { status: "pending" });
});

test("interpretWaitResponse ends when the planning session is gone", () => {
  assert.deepEqual(interpretWaitResponse(404, { message: "planning session not found" }), { status: "ended" });
  assert.deepEqual(interpretWaitResponse(404, {}, "planning session not found\n"), { status: "ended" });
});

test("interpretWaitResponse throws on 400", () => {
  assert.throws(() => interpretWaitResponse(400, { message: "invalid" }), /invalid/);
});

test("interpretWaitResponse retries a generic 500 as idle pending", () => {
  assert.deepEqual(interpretWaitResponse(500, { message: "oops" }), { status: "pending" });
});

test("runLoop sleeps 1s with no log after a Cloudflare 502, then runs the next message", async () => {
  const sleeps = [];
  const logs = [];
  const prompts = [];
  const results = [interpretWaitResponse(502, CLOUDFLARE_502), { status: "message", text: "hello" }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async (text) => {
      prompts.push(text);
      return 0;
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.deepEqual(logs, []);
  assert.deepEqual(prompts, ["hello"]);
});

test("runLoop sleeps 1s with no log when a transient wait has no retry_after", async () => {
  const sleeps = [];
  const logs = [];
  const results = [{ status: "pending", transient: true }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.deepEqual(logs, []);
});

test("runLoop ignores a large retry_after and stays silent", async () => {
  const sleeps = [];
  const logs = [];
  const results = [{ status: "pending", retry_after: 99999, transient: true }, { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.deepEqual(logs, []);
});

test("runLoop backs off silently on idle pending and empty-message waits", async () => {
  const sleeps = [];
  const logs = [];
  const results = [
    { status: "pending" },
    { status: "message", text: "   " },
    { status: "ended" },
  ];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000, 1000]);
  assert.deepEqual(logs, []);
});

test("safeWaitRequest treats a fetch throw as unreachable pending", async () => {
  const got = await safeWaitRequest(async () => {
    throw new TypeError("fetch failed");
  });
  assert.deepEqual(got, { status: "pending", unreachable: true });
});

test("runLoop ends cleanly after consecutive unreachable waits", async () => {
  const logs = [];
  const sleeps = [];
  let waits = 0;
  const code = await runLoop({
    waitOnce: async () => {
      waits += 1;
      if (waits > 10) {
        throw new Error("loop did not exit after unreachable waits");
      }
      return { status: "pending", unreachable: true };
    },
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
    maxUnreachableWaits: 3,
  });
  assert.equal(code, 0);
  assert.equal(waits, 3);
  assert.equal(sleeps.length, 2);
  assert.equal(logs.length, 1);
  assert.match(logs[0], /stopped waiting after 3 consecutive unreachable SuperPlane contacts/);
  assert.doesNotMatch(logs[0], /fail/i);
});

test("safeWaitRequest treats an abort as unreachable pending", async () => {
  const got = await safeWaitRequest(async () => {
    const err = new Error("This operation was aborted");
    err.name = "AbortError";
    throw err;
  });
  assert.deepEqual(got, { status: "pending", unreachable: true });
});

test("safeWaitRequest keeps a delivered user message", async () => {
  const got = await safeWaitRequest(async () => ({
    status: 200,
    text: async () => JSON.stringify({ status: "message", text: "hello" }),
  }));
  assert.deepEqual(got, { status: "message", text: "hello" });
});

test("safeWaitRequest treats 404 as pending", async () => {
  const got = await safeWaitRequest(async () => ({
    status: 404,
    text: async () => "ERR_NGROK_3200",
  }));
  assert.deepEqual(got, { status: "pending" });
});

test("safeWaitRequest still throws on 401", async () => {
  await assert.rejects(
    () =>
      safeWaitRequest(async () => ({
        status: 401,
        text: async () => JSON.stringify({ message: "unauthorized" }),
      })),
    /unauthorized/,
  );
});

test("runLoop runs a user message after a dropped wait", async () => {
  const prompts = [];
  const logs = [];
  const results = [
    await safeWaitRequest(async () => {
      throw new TypeError("fetch failed");
    }),
    { status: "message", text: "hello" },
    { status: "ended" },
  ];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async (text) => {
      prompts.push(text);
      return 0;
    },
    sleep: async () => {},
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(prompts, ["hello"]);
  assert.deepEqual(logs, []);
});

test("runLoop stays alive when waitOnce returns pending after a fetch throw", async () => {
  const sleeps = [];
  const logs = [];
  const results = [await safeWaitRequest(async () => { throw new TypeError("fetch failed"); }), { status: "ended" }];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => {
      throw new Error("prompt must not run");
    },
    sleep: async (ms) => {
      sleeps.push(ms);
    },
    log: (msg) => logs.push(msg),
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.deepEqual(sleeps, [1000]);
  assert.deepEqual(logs, []);
});

test("runLoop emits cmd_start then cmd_end for each follow-up prompt", async () => {
  const records = [];
  const nowValues = [5_000, 5_250, 6_000, 6_400];
  const results = [
    { status: "message", text: "Add color" },
    { status: "message", text: "Use the existing form" },
    { status: "ended" },
  ];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => 0,
    writeLiveLogRecord: (rec) => records.push(rec),
    now: () => nowValues.shift(),
  });
  assert.equal(code, 0);
  assert.deepEqual(records, [
    {
      type: "cmd_start",
      index: FOLLOW_UP_CMD_INDEX_BASE,
      text: "Add color",
      kind: "prompt",
      preview: "Add color",
      started_at: 5_000,
    },
    {
      type: "cmd_end",
      index: FOLLOW_UP_CMD_INDEX_BASE,
      status: "passed",
      duration_ms: 250,
    },
    {
      type: "cmd_start",
      index: FOLLOW_UP_CMD_INDEX_BASE + 1,
      text: "Use the existing form",
      kind: "prompt",
      preview: "Use the existing form",
      started_at: 6_000,
    },
    {
      type: "cmd_end",
      index: FOLLOW_UP_CMD_INDEX_BASE + 1,
      status: "passed",
      duration_ms: 400,
    },
  ]);
});

test("runLoop marks a failed follow-up cmd_end and keeps waiting", async () => {
  const records = [];
  const results = [{ status: "message", text: "hello" }, { status: "ended" }];
  const logs = [];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    runPrompt: async () => 2,
    writeLiveLogRecord: (rec) => records.push(rec),
    now: () => 1_000,
    log: (msg) => logs.push(msg),
  });
  assert.equal(code, 0);
  assert.equal(records[1].type, "cmd_end");
  assert.equal(records[1].status, "failed");
  assert.match(logs[0], /follow-up prompt failed with exit 2/);
});

test("runPromptFile forwards extra argv to run.js", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-loop-"));
  const argvFile = path.join(taskDir, "argv.json");
  fs.writeFileSync(
    path.join(taskDir, "run.js"),
    `require("fs").writeFileSync(${JSON.stringify(argvFile)}, JSON.stringify({argv: process.argv.slice(2), rewind: process.env.SUPERPLANE_ANALYSIS_REWIND}));\n`,
  );
  const promptFile = path.join(taskDir, "prompt.txt");
  fs.writeFileSync(promptFile, "hello\n");

  const code = await runPromptFile(taskDir, promptFile, "openai/gpt-4.1", ["64"]);
  assert.equal(code, 0);
  const recorded = JSON.parse(fs.readFileSync(argvFile, "utf8"));
  assert.deepEqual(recorded.argv, [promptFile, "openai/gpt-4.1", "64"]);
  assert.equal(recorded.rewind, "yes");
});

const PNG_BYTES = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00]);
const FILE_ID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
const APP_ENV = { SUPERPLANE_BASE_URL: "https://app.example" };

function gcsSignedURL(fileID = FILE_ID) {
  return `https://storage.googleapis.com/bucket/orgs/x/workspaces/y/tasks/z/${fileID}?sp_file=1`;
}

function hmacSignedURL(fileID = FILE_ID) {
  return `https://app.example/api/v1/public/files/${fileID}?expires=1&sig=abc&sp_file=1`;
}

function pngFetch(expectedURL) {
  return async (url) => {
    assert.equal(url, expectedURL);
    return {
      ok: true,
      status: 200,
      headers: { get: (name) => (String(name).toLowerCase() === "content-type" ? "image/png" : null) },
      arrayBuffer: async () => PNG_BYTES,
    };
  };
}

test("signedFileURLs keeps HMAC and object-storage image URLs", () => {
  const gcs = gcsSignedURL();
  const hmac = hmacSignedURL();
  const text = `See ![shot.png](${gcs}) and ![other](${hmac})`;
  assert.deepEqual(signedFileURLs(text), [gcs, hmac]);
});

test("signedFileURLs ignores unsigned URLs", () => {
  assert.deepEqual(
    signedFileURLs("See https://example.test/shot.png and https://app.example/api/v1/public/files/not-a-uuid?sp_file=1"),
    [],
  );
});

test("isAllowedSignedDownloadURL rejects HMAC URLs on an unexpected host", () => {
  const forged = `https://127.0.0.1/api/v1/public/files/${FILE_ID}?sp_file=1`;
  assert.equal(isAllowedSignedDownloadURL(forged, APP_ENV), false);
  assert.equal(isAllowedSignedDownloadURL(hmacSignedURL(), APP_ENV), true);
  assert.equal(isAllowedSignedDownloadURL(gcsSignedURL(), APP_ENV), true);
});

test("materializeFollowUpAttachments downloads signed images and rewrites the prompt", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-attachments-"));
  const signed = gcsSignedURL();
  const rewritten = await materializeFollowUpAttachments(
    taskDir,
    `See ![shot.png](${signed})`,
    pngFetch(signed),
  );
  assert.doesNotMatch(rewritten, /sp_file=1/);
  assert.match(rewritten, /inspect_attachment/);
  const saved = fs.readdirSync(path.join(taskDir, "attachments"));
  assert.equal(saved.length, 1);
  assert.match(saved[0], /^01-aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa\.png$/);
  assert.match(rewritten, new RegExp(saved[0].replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
  assert.equal(
    fs.readFileSync(path.join(taskDir, "attachments", saved[0])).compare(PNG_BYTES),
    0,
  );
});

test("materializeFollowUpAttachments continues numbering after existing files", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-attachments-index-"));
  const attachments = path.join(taskDir, "attachments");
  fs.mkdirSync(attachments);
  fs.writeFileSync(path.join(attachments, "01-existing.png"), "old");
  const signed = hmacSignedURL();
  const rewritten = await materializeFollowUpAttachments(
    taskDir,
    `See ![shot.png](${signed})`,
    pngFetch(signed),
    APP_ENV,
  );
  const saved = fs.readdirSync(attachments).sort();
  assert.deepEqual(saved, ["01-existing.png", `02-${FILE_ID}.png`]);
  assert.match(rewritten, /02-aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa\.png/);
  assert.doesNotMatch(rewritten, /sp_file=1/);
});

test("materializeFollowUpAttachments notes failed downloads and strips signed URLs", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-attachments-fail-"));
  const signed = gcsSignedURL();
  const rewritten = await materializeFollowUpAttachments(
    taskDir,
    `See ![shot.png](${signed})`,
    async () => ({ ok: false, status: 500 }),
  );
  assert.doesNotMatch(rewritten, /sp_file=1/);
  assert.doesNotMatch(rewritten, /inspect_attachment/);
  assert.match(rewritten, /SuperPlane could not download 1 user image/);
  assert.equal(fs.existsSync(path.join(taskDir, "attachments")), true);
  assert.deepEqual(fs.readdirSync(path.join(taskDir, "attachments")), []);
});

test("materializeFollowUpAttachments does not fetch HMAC URLs on an unexpected host", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-attachments-ssrf-"));
  const forged = `https://127.0.0.1/api/v1/public/files/${FILE_ID}?sp_file=1`;
  let calls = 0;
  const rewritten = await materializeFollowUpAttachments(
    taskDir,
    `See ![shot.png](${forged})`,
    async () => {
      calls += 1;
      throw new Error("must not fetch");
    },
    APP_ENV,
  );
  assert.equal(calls, 0);
  assert.doesNotMatch(rewritten, /127\.0\.0\.1/);
  assert.doesNotMatch(rewritten, /sp_file=1/);
  assert.match(rewritten, /SuperPlane could not download 1 user image/);
});

test("materializeFollowUpAttachments does not follow a redirect to an unexpected host", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-attachments-redirect-"));
  const signed = gcsSignedURL();
  const calls = [];
  const rewritten = await materializeFollowUpAttachments(
    taskDir,
    `See ![shot.png](${signed})`,
    async (url) => {
      calls.push(url);
      if (url === signed) {
        return {
          ok: false,
          status: 302,
          headers: { get: (name) => (String(name).toLowerCase() === "location" ? "http://127.0.0.1/secret" : null) },
        };
      }
      throw new Error("must not follow");
    },
  );
  assert.deepEqual(calls, [signed]);
  assert.match(rewritten, /SuperPlane could not download 1 user image/);
  assert.doesNotMatch(rewritten, /sp_file=1/);
});

test("materializeFollowUpAttachments rejects an oversized Content-Length", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-attachments-size-"));
  const signed = gcsSignedURL();
  let arrayBufferCalls = 0;
  const rewritten = await materializeFollowUpAttachments(
    taskDir,
    `See ![shot.png](${signed})`,
    async () => ({
      ok: true,
      status: 200,
      headers: {
        get: (name) => (String(name).toLowerCase() === "content-length" ? String(MAX_ATTACHMENT_BYTES + 1) : null),
      },
      arrayBuffer: async () => {
        arrayBufferCalls += 1;
        return PNG_BYTES;
      },
    }),
  );
  assert.equal(arrayBufferCalls, 0);
  assert.match(rewritten, /SuperPlane could not download 1 user image/);
});

test("runLoop downloads follow-up images before the prompt", async () => {
  const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "follow-up-loop-images-"));
  const signed = gcsSignedURL();
  const prompts = [];
  const results = [
    { status: "message", text: `Look at ![dialog](${signed})` },
    { status: "ended" },
  ];
  const code = await runLoop({
    waitOnce: async () => results.shift(),
    taskDir,
    fetch: pngFetch(signed),
    runPrompt: async (text) => {
      prompts.push(text);
      return 0;
    },
    sleep: async () => {},
    writeLiveLogRecord: () => {},
  });
  assert.equal(code, 0);
  assert.equal(prompts.length, 1);
  assert.doesNotMatch(prompts[0], /sp_file=1/);
  assert.match(prompts[0], /inspect_attachment/);
  const saved = fs.readdirSync(path.join(taskDir, "attachments"));
  assert.equal(saved.length, 1);
  assert.match(prompts[0], new RegExp(saved[0].replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
});
