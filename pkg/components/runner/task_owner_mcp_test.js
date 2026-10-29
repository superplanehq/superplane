"use strict";

const assert = require("node:assert/strict");
const { spawn, spawnSync, execFile } = require("node:child_process");
const { promisify } = require("node:util");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const readline = require("node:readline");
const test = require("node:test");
const claude = require("./claude/run");
const codex = require("./codex/run");
const openrouter = require("./openrouter/run");

test("task MCP without an owner capability exposes no owner or publication tools", () => {
  const result = spawnSync(
    process.execPath,
    [path.join(__dirname, "task_artifact_mcp.js")],
    {
      env: {
        ...process.env,
        SUPERPLANE_OWNER_TOKEN: "",
        SUPERPLANE_ARTIFACT_TOKEN: "",
        SUPERPLANE_RUN_TOKEN: "unrelated",
        SUPERPLANE_VISUAL_EVIDENCE: "",
        SUPERPLANE_PUBLISH_LINKS: "",
      },
      input: `${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "tools/list" })}\n${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/call", params: { name: "assign_owner", arguments: { user: "member", reason: "Owns pkg." } } })}\n`,
      encoding: "utf8",
    },
  );
  assert.equal(result.status, 0, result.stderr);
  const replies = result.stdout
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));
  assert.deepEqual(replies[0].result.tools, []);
  assert.equal(replies[1].result.isError, true);
});

// Execute the selected MCP command from the actual adapter configuration.
// The HTTP fixture owns only the transport response, not tool behavior.
for (const adapter of ["claude", "codex", "openrouter", "custom"]) {
  for (const mode of ["owner", "planning", "artifacts", "links", "merge"]) {
    test(`${adapter} ${mode} configuration delivers callable owner tools`, async (t) => {
      const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "owner-mcp-"));
      t.after(() => fs.rmSync(taskDir, { recursive: true, force: true }));
      const requests = [];
      const server = http.createServer(async (req, res) => {
        let body = "";
        for await (const chunk of req) body += chunk;
        requests.push({
          path: req.url,
          auth: req.headers.authorization,
          body: JSON.parse(body),
        });
        res.setHeader("Content-Type", "application/json");
        res.end(
          JSON.stringify(
            req.url.endsWith("/owner")
              ? { status: "assigned" }
              : { members: [{ user_id: "member" }] },
          ),
        );
      });
      await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
      t.after(() => server.close());
      const env = {
        ...process.env,
        SUPERPLANE_TASK_DIR: taskDir,
        SUPERPLANE_BASE_URL: `http://127.0.0.1:${server.address().port}`,
        SUPERPLANE_OWNER_TOKEN: "owner-secret",
        SUPERPLANE_RUN_TOKEN: "",
        SUPERPLANE_ARTIFACT_TOKEN: "",
        SUPERPLANE_MERGE_CONFIDENCE_TOKEN: "",
        SUPERPLANE_VISUAL_EVIDENCE: "",
        SUPERPLANE_PUBLISH_LINKS: "",
        SUPERPLANE_PLANNING_SESSION_KIND: "",
        SUPERPLANE_PLANNING_SESSION_ID: "",
      };
      if (mode === "planning")
        Object.assign(env, {
          SUPERPLANE_OWNER_TOKEN: "",
          SUPERPLANE_RUN_TOKEN: "planning-secret",
          SUPERPLANE_PLANNING_SESSION_ID: "session",
          SUPERPLANE_PLANNING_SESSION_KIND: "work_order_analysis",
        });
      if (mode === "artifacts")
        Object.assign(env, {
          SUPERPLANE_ARTIFACT_TOKEN: "artifact-secret",
          SUPERPLANE_VISUAL_EVIDENCE: "yes",
        });
      if (mode === "links")
        Object.assign(env, {
          SUPERPLANE_ARTIFACT_TOKEN: "artifact-secret",
          SUPERPLANE_PUBLISH_LINKS: "yes",
        });
      if (mode === "merge")
        env.SUPERPLANE_MERGE_CONFIDENCE_TOKEN = "merge-secret";
      let command;
      if (adapter === "claude") {
        const config = JSON.parse(
          fs.readFileSync(claude.writeClaudeMCPConfig(taskDir, env, true)),
        );
        command = [
          config.mcpServers.superplane.command,
          ...config.mcpServers.superplane.args,
        ];
        assert.match(
          claude.allowedClaudeTools(env),
          /mcp__superplane__assign_owner/,
        );
      } else if (adapter === "codex") {
        const args = codex.codexExecArgs(
          env,
          "model",
          codex.superplaneMCPScript(taskDir, env),
        );
        const setting = args.find((arg) =>
          arg.startsWith("mcp_servers.superplane.args="),
        );
        assert.ok(setting, "owner-only runs must enable MCP");
        command = [
          "node",
          ...JSON.parse(setting.split("=").slice(1).join("=")),
        ];
        const forwarded = args.find((arg) =>
          arg.startsWith("mcp_servers.superplane.env_vars="),
        );
        assert.match(forwarded, /SUPERPLANE_OWNER_TOKEN/);
        assert.ok(!args.some((arg) => arg.includes("owner-secret")));
      } else {
        if (adapter === "custom")
          Object.assign(env, {
            CUSTOM_LLM_BASE_URL: "https://provider.example",
            CUSTOM_LLM_API_KEY: "custom-key",
            CUSTOM_LLM_API_TYPE: "openai-compatible",
          });
        command = openrouter.buildOpenCodeConfig({
          taskDir,
          env,
          planning: mode === "planning",
          models: ["model"],
        }).mcp.superplane.command;
      }
      // Ship the same modules required by the selected server into the task.
      for (const file of [
        "planning_session_mcp.js",
        "task_artifact_mcp.js",
        "merge_confidence_mcp.js",
        "task_owner_tools.js",
        "publish_link.js",
        "analysis_protocol.js",
        "analysis_protocol_review.md",
        "attachment_limit.js",
      ]) {
        fs.copyFileSync(path.join(__dirname, file), path.join(taskDir, file));
      }
      const child = spawn(command[0], command.slice(1), { env });
      t.after(() => child.kill());
      const lines = readline.createInterface({ input: child.stdout });
      const pending = new Map();
      lines.on("line", (line) => {
        const reply = JSON.parse(line);
        pending.get(reply.id)?.(reply);
      });
      let id = 0;
      async function call(method, params) {
        const requestID = ++id;
        const reply = new Promise((resolve, reject) => {
          const timeout = setTimeout(
            () => reject(new Error("MCP response timed out")),
            3000,
          );
          pending.set(requestID, (value) => {
            clearTimeout(timeout);
            resolve(value);
          });
        });
        child.stdin.write(
          `${JSON.stringify({ jsonrpc: "2.0", id: requestID, method, params })}\n`,
        );
        return reply;
      }
      const listed = (await call("tools/list")).result.tools.map(
        (tool) => tool.name,
      );
      assert.ok(listed.includes("suggest_owners"));
      assert.ok(listed.includes("assign_owner"));
      if (mode === "owner")
        assert.deepEqual(listed.sort(), ["assign_owner", "suggest_owners"]);
      assert.equal(listed.includes("propose_update"), mode === "planning");
      assert.equal(listed.includes("report_merge_check"), mode === "merge");
      const suggestion = await call("tools/call", {
        name: "suggest_owners",
        arguments: { paths: [" pkg/ "] },
      });
      assert.equal(
        suggestion.result.structuredContent.members[0].user_id,
        "member",
      );
      const assigned = await call("tools/call", {
        name: "assign_owner",
        arguments: {
          user: " member ",
          reason: " Owns pkg. ",
          task_id: "unrelated",
        },
      });
      assert.equal(assigned.result.structuredContent.status, "assigned");
      const prefix =
        mode === "planning"
          ? "/api/v1/runner/planning-sessions"
          : "/api/v1/runner/tasks";
      assert.deepEqual(requests, [
        {
          path: `${prefix}/suggest-owners`,
          auth: `Bearer ${mode === "planning" ? "planning-secret" : "owner-secret"}`,
          body: { paths: ["pkg/"] },
        },
        {
          path: `${prefix}/owner`,
          auth: `Bearer ${mode === "planning" ? "planning-secret" : "owner-secret"}`,
          body: { user: "member", reason: "Owns pkg." },
        },
      ]);
    });
  }
}

for (const adapter of ["claude", "codex", "openrouter"]) {
  test(`${adapter} CLI starts SuperPlane MCP for an owner-only run`, async (t) => {
    const taskDir = fs.mkdtempSync(path.join(os.tmpdir(), "owner-cli-"));
    t.after(() => fs.rmSync(taskDir, { recursive: true, force: true }));
    const cli = adapter === "openrouter" ? "opencode" : adapter;
    fs.writeFileSync(
      path.join(taskDir, cli),
      `#!/usr/bin/env node
const fs = require("node:fs");
fs.writeFileSync(process.env.CAPTURE_ARGS, JSON.stringify(process.argv.slice(2)));
process.stdout.write(JSON.stringify({type:"result", result:"", is_error:false}) + "\\n");
`,
      { mode: 0o755 },
    );
    const prompt = path.join(taskDir, "prompt.txt");
    fs.writeFileSync(prompt, "Implement the task.");
    fs.writeFileSync(path.join(taskDir, "prompt_count"), "0");
    const capture = path.join(taskDir, "args.json");
    await promisify(execFile)(
      process.execPath,
      [path.join(__dirname, adapter, "run.js"), prompt],
      {
        cwd: taskDir,
        env: {
          ...process.env,
          PATH: `${taskDir}:${process.env.PATH}`,
          CAPTURE_ARGS: capture,
          SUPERPLANE_TASK_DIR: taskDir,
          SUPERPLANE_RESULT_FILE: path.join(taskDir, "result.json"),
          SUPERPLANE_OWNER_TOKEN: "owner-secret",
          SUPERPLANE_RUN_TOKEN: "",
          SUPERPLANE_ARTIFACT_TOKEN: "",
          SUPERPLANE_MERGE_CONFIDENCE_TOKEN: "",
          SUPERPLANE_VISUAL_EVIDENCE: "",
          SUPERPLANE_PUBLISH_LINKS: "",
          SUPERPLANE_PLANNING_SESSION_KIND: "",
          SUPERPLANE_PLANNING_SESSION_ID: "",
          OPENAI_API_KEY: "stub-key",
          CODEX_API_KEY: "stub-key",
        },
        timeout: 10000,
      },
    );
    const args = JSON.parse(fs.readFileSync(capture));
    let script;
    if (adapter === "claude") {
      const configFile = args[args.indexOf("--mcp-config") + 1];
      script = JSON.parse(fs.readFileSync(configFile)).mcpServers.superplane
        .args[0];
    } else if (adapter === "codex") {
      const setting = args.find((arg) =>
        arg.startsWith("mcp_servers.superplane.args="),
      );
      assert.ok(setting);
      script = JSON.parse(setting.split("=").slice(1).join("="))[0];
      assert.ok(!args.some((arg) => arg.includes("owner-secret")));
    } else {
      script = JSON.parse(fs.readFileSync(path.join(taskDir, "opencode.json")))
        .mcp.superplane.command[1];
    }
    assert.equal(path.basename(script), "task_artifact_mcp.js");
  });
}
