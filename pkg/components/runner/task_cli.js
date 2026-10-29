"use strict";

const fs = require("node:fs");

const { publishLink } = require("./publish_link");
const { publishMarkdown } = require("./publish_markdown");

const COMMANDS = [
  {
    words: ["task", "artifact", "add"],
    usage: "superplane task artifact add (--url URL | --file PATH) [--name NAME] [--title TITLE] [--key KEY]",
    flags: ["url", "file", "name", "title", "key"],
    alias: { name: "title" },
    run: "publish",
  },
  {
    words: ["task", "artifact", "remove"],
    usage: "superplane task artifact remove --name NAME",
    flags: ["name", "title"],
    alias: { title: "name" },
    required: ["name"],
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
    words: ["task", "create"],
    usage: "superplane task create --title TITLE [--description TEXT]",
    flags: ["title", "description"],
    required: ["title"],
  },
  {
    words: ["task", "close"],
    usage: "superplane task close",
    flags: [],
  },
  {
    words: ["task", "list"],
    usage: "superplane task list [--state STATE] [--column COLUMN]",
    flags: ["state", "column"],
  },
  {
    words: ["task", "assign"],
    usage: "superplane task assign --assignee EMAIL",
    flags: ["assignee"],
    required: ["assignee"],
  },
  {
    words: ["task", "plan"],
    usage: "superplane task plan --file PATH",
    flags: ["file"],
    required: ["file"],
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
  for (let index = command.words.length; index < args.length; index += 1) {
    const flag = args[index];
    const value = args[index + 1];
    const name = flag.startsWith("--") ? flag.slice(2) : "";
    if (!command.flags.includes(name) || !value || value.startsWith("--")) {
      throw new Error(`usage: ${command.usage}`);
    }
    input[(command.alias && command.alias[name]) || name] = value;
    index += 1;
  }

  for (const name of command.required || []) {
    if (!String(input[name] || "").trim()) throw new Error(`usage: ${command.usage}`);
  }
  return { command, input };
}

async function main(argv, env = process.env, fetchImpl = fetch, stdout = process.stdout, stderr = process.stderr) {
  try {
    const { command, input } = parseCommand(argv);
    if (command.run !== "publish") {
      throw new Error(`superplane ${command.words.join(" ")} is not available yet`);
    }
    if (input.file && input.url) {
      throw new Error(`usage: ${command.usage}`);
    }
    if (input.file) {
      let body;
      try {
        body = fs.readFileSync(input.file, "utf8");
      } catch (_error) {
        throw new Error(`cannot read ${input.file}`);
      }
      const result = await publishMarkdown(
        { body, title: input.title, key: input.key },
        env,
        fetchImpl,
      );
      stdout.write(`${String(result.title || "").trim()}\n`);
    } else {
      const result = await publishLink(
        { url: input.url, title: input.title, key: input.key },
        env,
        fetchImpl,
      );
      stdout.write(`${String(result.url || "").trim()}\n`);
    }
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
