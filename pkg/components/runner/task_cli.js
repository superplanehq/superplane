"use strict";

const fs = require("node:fs");

const { createClient } = require("./superplane_mcp/client");
const registry = require("./superplane_mcp/registry");

const labelFlags = ["name", "title", "key"];
const labelAlias = { name: "title" };

const COMMANDS = [
  {
    words: ["task", "owner", "suggest"],
    usage: "superplane task owner suggest [--path PATH]",
    flags: ["path"],
    repeat: ["path"],
    alias: { path: "paths" },
    tool: "suggest_owners",
  },
  {
    words: ["task", "owner", "assign"],
    usage: "superplane task owner assign --user USER [--reason TEXT] [--only-if-unassigned]",
    flags: ["user", "reason", "only-if-unassigned"],
    boolean: ["only-if-unassigned"],
    alias: { "only-if-unassigned": "only_if_unassigned" },
    required: ["user"],
    tool: "assign_owner",
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
    tool: "propose_update",
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
    words: ["task", "artifact", "add-link"],
    usage: "superplane task artifact add-link --url URL [--name NAME] [--key KEY]",
    flags: ["url", ...labelFlags],
    alias: labelAlias,
    required: ["url"],
    tool: "publish_link",
    line: (result) => result.url,
  },
  {
    words: ["task", "artifact", "add-markdown"],
    usage: "superplane task artifact add-markdown --file PATH [--name NAME] [--key KEY]",
    flags: ["file", ...labelFlags],
    alias: labelAlias,
    required: ["file"],
    line: (result) => result.title,
    run: publishMarkdownFile,
  },
  {
    words: ["task", "artifact", "add-file"],
    usage: "superplane task artifact add-file --file PATH [--poster PATH] [--name NAME]",
    flags: ["file", "poster", "name", "title"],
    alias: { file: "path", poster: "posterPath", name: "title" },
    required: ["path"],
    tool: "upload_artifact",
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
    words: ["task", "check", "add"],
    usage: "superplane task check add --name NAME",
    flags: ["name"],
    required: ["name"],
  },
  {
    words: ["task", "check", "remove"],
    usage: "superplane task check remove --name NAME",
    flags: ["name"],
    required: ["name"],
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

async function publishMarkdownFile(input, env, fetchImpl) {
  const { publishMarkdown } = require("./publish_markdown");
  return publishMarkdown({ body: readText(input.file).trim(), title: input.title, key: input.key }, env, fetchImpl);
}

function render(result) {
  if (result && Array.isArray(result.content) && result.structuredContent) return result.structuredContent;
  return result == null ? {} : result;
}

async function callTool(name, input, env, fetchImpl) {
  const tool = registry.findTool(name);
  if (!tool) throw new Error(`unknown tool: ${name}`);
  const client = createClient(env, fetchImpl);
  // A shell cannot return an image content block. Do not record an
  // inspection here. inspect_screenshot records it when the model sees
  // the image.
  if (name === "inspect_screenshot" || name === "inspect_attachment") client.recordInspection = false;
  return tool.handler(input, client);
}

async function main(argv, env = process.env, fetchImpl = fetch, stdout = process.stdout, stderr = process.stderr) {
  if (Array.isArray(argv) && argv[0] === "task" && argv[1] === "help") {
    stdout.write(`${usage()}\n`);
    return 0;
  }
  try {
    const { command, input } = parseCommand(argv);
    if (!command.tool && !command.run) {
      throw new Error(`superplane ${command.words.join(" ")} is not available yet`);
    }
    const toolInput = command.prepare ? command.prepare(input) : input;
    const result = command.run
      ? await command.run(toolInput, env, fetchImpl)
      : await callTool(command.tool, toolInput, env, fetchImpl);
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
