package openrouter

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPinnedOpenCodeConfigurationLoader(t *testing.T) {
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("dependency check requires the local worker image with OpenCode 1.18.31")
	}
	// The JavaScript harness exercises the real runner and pinned OpenCode
	// against a local synthetic model service without external credentials.
	cmd := exec.Command("node", "-e", `
"use strict";
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const http = require("node:http");
const { spawnSync } = require("node:child_process");
const {
  runPrompt,
  openCodeProcessEnv,
  buildOpenCodeConfig,
} = require("./run.js");

const root = fs.mkdtempSync(path.join(os.tmpdir(), "managed-opencode-"));
const baseEnv = { PATH: process.env.PATH, HOME: path.join(root, "home") };
fs.mkdirSync(baseEnv.HOME);
const incompatible = JSON.stringify({
  permissions: { edit: "allow" },
  agents: { triaging: { permissions: { edit: "allow" } } },
  providers: { unwanted: { name: "Project provider" } },
});

function write(file, content) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, content);
}

function cli(args, env, cwd) {
  const result = spawnSync("opencode", ["--pure", ...args], {
    env,
    cwd,
    encoding: "utf8",
    timeout: 15000,
    maxBuffer: 4 * 1024 * 1024,
  });
  assert.ifError(result.error);
  return result;
}

async function main() {
  assert.equal(cli(["--version"], baseEnv, root).stdout.trim(), "1.18.31");
  const requests = [];
  let readFile = "";
  const server = http.createServer((req, res) => {
    let body = "";
    req.on("data", (chunk) => {
      body += chunk;
    });
    req.on("end", () => {
      const request = JSON.parse(body);
      requests.push(request);
      res.writeHead(200, { "Content-Type": "text/event-stream" });
      const shouldRead =
        readFile &&
        request.tools?.some((tool) => tool.function?.name === "read");
      const delta = shouldRead
        ? {
            tool_calls: [
              {
                index: 0,
                id: "read-fixture",
                type: "function",
                function: {
                  name: "read",
                  arguments: JSON.stringify({ filePath: readFile }),
                },
              },
            ],
          }
        : { content: "Plan complete." };
      const finishReason = shouldRead ? "tool_calls" : "stop";
      if (shouldRead) readFile = "";
      res.end(
        "data: " + JSON.stringify({ id: "synthetic", object: "chat.completion.chunk", created: 1, model: "fixture", choices: [{ index: 0, delta, finish_reason: finishReason }], usage: { prompt_tokens: 10, completion_tokens: 3, total_tokens: 13 } }) + "\n\ndata: [DONE]\n\n",
      );
    });
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  try {
    for (const hasConfig of [false, true]) {
      const repo = path.join(root, hasConfig ? "v2-repo" : "plain-repo");
      const task = path.join(root, hasConfig ? "v2-task" : "plain-task");
      fs.mkdirSync(repo, { recursive: true });
      const git = spawnSync("git", ["init", "--quiet", repo]);
      assert.equal(git.status, 0);
      write(path.join(repo, "AGENTS.md"), "Repository instruction marker.");
      write(path.join(repo, "CLAUDE.md"), "Disabled Claude marker.");
      write(path.join(root, "AGENTS.md"), "Outside repository marker.");
      write(
        path.join(repo, ".agents/skills/fixture/SKILL.md"),
        "---\nname: fixture\ndescription: Fixture skill marker.\n---\nFixture skill body.\n",
      );
      const cwd = hasConfig ? path.join(repo, "src") : repo;
      fs.mkdirSync(cwd, { recursive: true });
      if (hasConfig) {
        write(path.join(repo, "opencode.json"), incompatible);
        write(path.join(cwd, "AGENTS.md"), "Nested working directory marker.");
        write(path.join(cwd, "CONTEXT.md"), "Lower priority context marker.");
        write(
          path.join(cwd, "deep/AGENTS.md"),
          "Lazy nested instruction marker.",
        );
        readFile = path.join(cwd, "deep/file.txt");
        write(readFile, "Nested file content.");
      }
      write(path.join(task, "prompt_count"), "0\n");
      write(
        path.join(task, "prompt.txt"),
        "Return Plan complete. Do not use tools.",
      );
      const env = {
        ...baseEnv,
        SUPERPLANE_TASK_DIR: task,
        SUPERPLANE_RESULT_FILE: path.join(task, "result.json"),
        SUPERPLANE_PLANNING_SESSION_ID: "synthetic-plan",
        SUPERPLANE_PLANNING_SESSION_KIND: "work_order_analysis",
        CUSTOM_LLM_API_TYPE: "openai-compatible",
        CUSTOM_LLM_API_KEY: "synthetic-key",
        CUSTOM_LLM_BASE_URL: "http://127.0.0.1:" + server.address().port + "/v1",
        OPENCODE_CONFIG: "/inherited/invalid.json",
        OPENCODE_CONFIG_CONTENT: incompatible,
        OPENCODE_CONFIG_DIR: repo,
        OPENCODE_PERMISSION: '{"edit":"allow"}',
      };
      assert.equal(
        await runPrompt(path.join(task, "prompt.txt"), "fixture", {
          env,
          cwd,
          recordAgentMessage: async () => {},
        }),
        0,
      );
      const childEnv = openCodeProcessEnv(task, env);
      assert.equal(
        JSON.parse(fs.readFileSync(env.SUPERPLANE_RESULT_FILE, "utf8")).result,
        "Plan complete.",
      );
      assert.equal(childEnv.OPENCODE_CONFIG, path.join(task, "opencode.json"));
      const resolved = cli(["debug", "config"], childEnv, cwd);
      assert.equal(resolved.status, 0, resolved.stderr);
      const config = JSON.parse(resolved.stdout);
      assert.equal(config.permission.edit, "deny");
      assert.equal(config.permission.question, "deny");
      assert.equal(config.provider.custom.options.apiKey, "synthetic-key");
      assert.equal(
        config.provider.custom.options.baseURL,
        env.CUSTOM_LLM_BASE_URL,
      );
      assert.equal(config.permissions, undefined);
      assert.equal(config.agent.triaging, undefined);
      assert.equal(config.provider.unwanted, undefined);
      const context = JSON.stringify(
        requests.filter((request) => request.tools?.length).at(-1).messages,
      );
      assert.match(context, /Repository instruction marker/);
      assert.match(context, /Instructions from: .*analysis_protocol.md/);
      assert.doesNotMatch(
        context,
        /Disabled Claude marker|Outside repository marker|Lower priority context marker/,
      );
      if (hasConfig) {
        assert.match(context, /Nested working directory marker/);
        assert.match(context, /Lazy nested instruction marker/);
      }
      const skills = cli(["debug", "skill"], childEnv, cwd);
      assert.equal(skills.status, 0, skills.stderr);
      assert.match(skills.stdout, /Fixture skill marker/);
      write(
        childEnv.OPENCODE_CONFIG,
        JSON.stringify(
          buildOpenCodeConfig({
            taskDir: task,
            cwd,
            planning: true,
            env: {
              OPENROUTER_API_KEY: "synthetic-openrouter-key",
              OPENROUTER_BASE_URL: "https://models.example/v1",
            },
            models: ["openai/gpt-4o"],
          }),
        ),
      );
      const openrouterResolved = cli(["debug", "config"], childEnv, cwd);
      assert.equal(openrouterResolved.status, 0, openrouterResolved.stderr);
      const openrouter = JSON.parse(openrouterResolved.stdout).provider
        .openrouter;
      assert.equal(openrouter.options.apiKey, "synthetic-openrouter-key");
      assert.equal(openrouter.options.baseURL, "https://models.example/v1");
      if (!hasConfig) continue;

      // Negative control proves that the real V1 loader rejects these fields.
      const unisolated = { ...childEnv, OPENCODE_DISABLE_PROJECT_CONFIG: "0" };
      const failure = cli(["debug", "config"], unisolated, cwd);
      assert.notEqual(failure.status, 0);
      assert.match(failure.stderr, /V2 permissions are not supported/);
      assert.equal(
        fs.readFileSync(path.join(repo, "opencode.json"), "utf8"),
        incompatible,
      );

      // Cover the other discovery locations without repeating model requests.
      for (const file of ["opencode.jsonc", ".opencode/opencode.json"]) {
        fs.unlinkSync(path.join(repo, "opencode.json"));
        write(path.join(repo, file), incompatible);
        const isolated = cli(["debug", "config"], childEnv, cwd);
        assert.equal(isolated.status, 0, isolated.stderr);
        assert.equal(JSON.parse(isolated.stdout).agent.triaging, undefined);
        assert.equal(
          fs.readFileSync(path.join(repo, file), "utf8"),
          incompatible,
        );
        const rejected = cli(["debug", "config"], unisolated, cwd);
        assert.notEqual(rejected.status, 0);
        assert.match(rejected.stderr, /V2 permissions are not supported/);
        fs.unlinkSync(path.join(repo, file));
        write(path.join(repo, "opencode.json"), incompatible);
      }
      write(childEnv.OPENCODE_CONFIG, incompatible);
      const invalidManaged = cli(["debug", "config"], childEnv, cwd);
      assert.notEqual(invalidManaged.status, 0);
      assert.ok(invalidManaged.stderr.includes(childEnv.OPENCODE_CONFIG));
      assert.match(invalidManaged.stderr, /V2 permissions are not supported/);
    }
  } finally {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
  }
}

main()
  .finally(() => fs.rmSync(root, { recursive: true, force: true }))
  .catch((err) => {
    console.error(err);
    process.exitCode = 1;
  });
`)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
