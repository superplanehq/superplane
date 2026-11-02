"use strict";

const fs = require("node:fs");
const path = require("node:path");

const { createClient } = require("./superplane_mcp/client");
const registry = require("./superplane_mcp/registry");

const labelFlags = ["name", "title", "key"];
const labelAlias = { name: "title" };

const COMMANDS = [
  {
    words: ["task", "members"],
    usage: "superplane task members",
    flags: [],
    tool: "list_members",
  },
  {
    words: ["task", "owner", "suggest"],
    usage: "superplane task owner suggest [--path PATH]",
    flags: ["path"],
    repeat: ["path"],
    alias: { path: "paths" },
    tool: "suggest_task_owner",
  },
  {
    words: ["task", "owner", "assign"],
    usage: "superplane task owner assign --user USER [--reason TEXT] [--only-if-unassigned]",
    flags: ["user", "reason", "only-if-unassigned"],
    boolean: ["only-if-unassigned"],
    alias: { "only-if-unassigned": "only_if_unassigned" },
    required: ["user"],
    tool: "assign_task_owner",
  },
  {
    words: ["task", "list"],
    usage: "superplane task list [--state STATE] [--limit N]",
    flags: ["state", "limit"],
    repeat: ["state"],
    tool: "list_tasks",
  },
  {
    words: ["task", "create"],
    usage: "superplane task create --title TITLE [--description TEXT]",
    flags: ["title", "description"],
    required: ["title"],
    tool: "create_task",
  },
  {
    words: ["task", "close"],
    usage: "superplane task close [--result completed|rejected|failed]",
    flags: ["result"],
    tool: "close_task",
  },
  {
    words: ["task", "plan"],
    usage: "superplane task plan [--file PATH] [--scores PATH] [--survey PATH]",
    flags: ["file", "scores", "survey"],
    tool: "propose_plan",
    prepare: planInput,
  },
  {
    words: ["task", "attachment", "inspect"],
    usage: "superplane task attachment inspect --path PATH",
    flags: ["path"],
    required: ["path"],
    tool: "inspect_attachment",
  },
  {
    words: ["task", "evidence", "inspect"],
    usage: "superplane task evidence inspect --file PATH",
    flags: ["file"],
    alias: { file: "path" },
    required: ["path"],
    tool: "inspect_screenshot",
  },
  {
    words: ["task", "evidence", "unavailable"],
    usage: "superplane task evidence unavailable --reason TEXT --attempts PATH",
    flags: ["reason", "attempts"],
    required: ["reason", "attempts"],
    tool: "report_visual_evidence_unavailable",
    prepare: evidenceUnavailableInput,
  },
  {
    words: ["task", "artifact", "list"],
    usage: "superplane task artifact list [--type markdown|link|file]",
    flags: ["type"],
    tool: "list_artifacts",
  },
  {
    words: ["task", "artifact", "get"],
    usage: "superplane task artifact get (--id ID | --key KEY | --path PATH)",
    flags: ["id", "key", "path"],
    tool: "get_artifact",
    prepare: shellPaths,
  },
  {
    words: ["task", "artifact", "add-link"],
    usage: "superplane task artifact add-link --url URL [--name NAME] [--key KEY]",
    flags: ["url", ...labelFlags],
    alias: labelAlias,
    required: ["url"],
    tool: "add_artifact",
    prepare: (input) => ({ ...input, type: "link" }),
    line: (result) => result.url,
  },
  {
    words: ["task", "artifact", "add-markdown"],
    usage: "superplane task artifact add-markdown --file PATH [--name NAME] [--key KEY]",
    flags: ["file", ...labelFlags],
    alias: labelAlias,
    required: ["file"],
    tool: "add_artifact",
    prepare: markdownInput,
    line: (result) => result.title,
  },
  {
    words: ["task", "artifact", "add-file"],
    usage: "superplane task artifact add-file --file PATH [--poster PATH] [--name NAME] [--key KEY]",
    flags: ["file", "poster", ...labelFlags],
    alias: { file: "path", poster: "posterPath", ...labelAlias },
    required: ["path"],
    tool: "add_artifact",
    prepare: (input) => ({ ...shellPaths(input), type: "file" }),
    line: (result) => result.title,
  },
  {
    words: ["task", "artifact", "remove"],
    usage: "superplane task artifact remove [--name NAME] [--key KEY]",
    flags: ["name", "title", "key"],
    alias: { name: "title" },
    tool: "remove_artifact",
  },
  {
    words: ["task", "check", "report"],
    usage: "superplane task check report --check ID --score N --summary TEXT",
    flags: ["check", "score", "summary"],
    required: ["check", "score", "summary"],
    tool: "report_merge_check",
  },
  {
    words: ["task", "check", "list"],
    usage: "superplane task check list --task TASK",
    flags: ["task"],
    required: ["task"],
    tool: "list_checks",
  },
  {
    words: ["task", "check", "add"],
    usage:
      "superplane task check add --key KEY --name NAME --score N --max-score N --level positive|caution|critical --reason TEXT",
    flags: ["key", "name", "score", "max-score", "level", "reason"],
    alias: { "max-score": "max_score" },
    required: ["key", "name", "score", "max_score", "level", "reason"],
    tool: "add_checks",
    prepare: (input) => ({ checks: [input] }),
  },
  {
    words: ["task", "check", "remove"],
    usage: "superplane task check remove --key KEY [--key KEY]",
    flags: ["key"],
    repeat: ["key"],
    alias: { key: "keys" },
    required: ["keys"],
    tool: "remove_checks",
  },
  {
    words: ["task", "move"],
    usage: "superplane task move --column COLUMN",
    flags: ["column"],
    required: ["column"],
  },
];

function usage() {
  return ["usage:", ...COMMANDS.map((command) => `  ${command.usage}`)].join("\n");
}

function findCommand(args) {
  return COMMANDS.find((command) => command.words.every((word, index) => args[index] === word));
}

function parseCommand(argv) {
  const args = Array.isArray(argv) ? argv : [];
  const command = findCommand(args);
  if (!command) throw new Error(usage());

  const input = {};
  const repeat = new Set(command.repeat || []);
  const boolean = new Set(command.boolean || []);
  for (let index = command.words.length; index < args.length; index += 1) {
    const flag = args[index];
    const name = flag.startsWith("--") ? flag.slice(2) : "";
    if (!command.flags.includes(name)) throw new Error(`usage: ${command.usage}`);
    const stored = (command.alias && command.alias[name]) || name;
    if (boolean.has(name)) {
      input[stored] = true;
      continue;
    }
    const value = args[index + 1];
    if (!value || value.startsWith("--")) throw new Error(`usage: ${command.usage}`);
    if (repeat.has(name)) {
      const current = input[stored];
      input[stored] = current ? [...current, value] : [value];
    } else {
      input[stored] = value;
    }
    index += 1;
  }

  for (const name of command.required || []) {
    const value = input[name];
    if (Array.isArray(value) ? value.length === 0 : !String(value || "").trim()) {
      throw new Error(`usage: ${command.usage}`);
    }
  }
  return { command, input };
}

function readText(file) {
  try {
    return fs.readFileSync(file, "utf8");
  } catch (_error) {
    throw new Error(`cannot read ${file}`);
  }
}

function readJSON(file) {
  let text;
  try {
    text = fs.readFileSync(file, "utf8");
  } catch (_error) {
    throw new Error(`cannot read ${file}`);
  }
  try {
    return JSON.parse(text);
  } catch (_error) {
    throw new Error(`${file} must be JSON`);
  }
}

// A shell path is relative to the current directory, not the task directory.
function shellPaths(input) {
  const resolved = { ...input };
  for (const name of ["path", "posterPath"]) {
    if (resolved[name]) resolved[name] = path.resolve(resolved[name]);
  }
  return resolved;
}

function planInput(input) {
  const body = {};
  if (input.file) body.spec = readText(input.file);
  if (input.scores) body.scores = readJSON(input.scores);
  if (input.survey) body.survey = readJSON(input.survey);
  if (!body.spec && !body.scores && !body.survey) {
    throw new Error("usage: superplane task plan [--file PATH] [--scores PATH] [--survey PATH]");
  }
  return body;
}

function evidenceUnavailableInput(input) {
  return { reason: input.reason, attempts: readJSON(input.attempts) };
}

function markdownInput(input) {
  return { type: "markdown", body: readText(input.file), title: input.title, key: input.key };
}

function render(result) {
  if (result && Array.isArray(result.content) && result.structuredContent) return result.structuredContent;
  return result == null ? {} : result;
}

async function callTool(name, input, env, fetchImpl) {
  const resolved = registry.resolve(name, registry.allTools());
  if (!resolved) throw new Error(`unknown tool: ${name}`);
  const client = createClient(env, fetchImpl);
  // A shell cannot return an image content block. Do not record an
  // inspection here. get_artifact records it when the model sees the image.
  if (resolved.tool.definition.name === "get_artifact") client.recordInspection = false;
  return resolved.handler(input, client);
}

async function main(argv, env = process.env, fetchImpl = fetch, stdout = process.stdout, stderr = process.stderr) {
  if (Array.isArray(argv) && argv[0] === "task" && ["help", "--help", "-h"].includes(argv[1])) {
    stdout.write(`${usage()}\n`);
    return 0;
  }
  try {
    const { command, input } = parseCommand(argv);
    if (!command.tool) {
      throw new Error(`superplane ${command.words.join(" ")} is not available yet`);
    }
    const toolInput = command.prepare ? command.prepare(input) : input;
    const result = await callTool(command.tool, toolInput, env, fetchImpl);
    const shown = command.line ? command.line(result) : JSON.stringify(render(result));
    stdout.write(`${String(shown ?? "").trim()}\n`);
  } catch (error) {
    stderr.write(`${error && error.message ? error.message : error}\n`);
    return 1;
  }
  return 0;
}

if (require.main === module) {
  main(process.argv.slice(2)).then((code) => process.exit(code));
}

module.exports = {
  main,
  parseCommand,
};
