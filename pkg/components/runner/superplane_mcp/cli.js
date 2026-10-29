"use strict";

/**
 * Shell commands for the global runner tools. Each command runs the same
 * handler as its MCP tool, with the same token. The backend checks the scope.
 */

const { createClient } = require("./client");
const registry = require("./registry");

function usage(tools = registry.allTools()) {
  return ["usage:", ...tools.map((tool) => `  ${tool.cli.usage}`)].join("\n");
}

function matches(argv) {
  return Boolean(registry.findCommand(Array.isArray(argv) ? argv : []));
}

function parseFlags(cli, args) {
  const flags = {};
  for (let index = cli.words.length; index < args.length; index += 2) {
    const flag = String(args[index] || "");
    const value = args[index + 1];
    const raw = flag.startsWith("--") ? flag.slice(2) : "";
    if (!cli.flags.includes(raw) || value === undefined || String(value).startsWith("--")) {
      throw new Error(`usage: ${cli.usage}`);
    }
    const name = (cli.alias && cli.alias[raw]) || raw;
    if ((cli.repeatable || []).includes(name)) {
      flags[name] = [...(flags[name] || []), value];
      continue;
    }
    flags[name] = value;
  }
  for (const name of cli.required || []) {
    if (!String(flags[name] || "").trim()) throw new Error(`usage: ${cli.usage}`);
  }
  return flags;
}

function parseCommand(argv) {
  const args = Array.isArray(argv) ? argv : [];
  const tool = registry.findCommand(args);
  if (!tool) throw new Error(usage());
  return { tool, input: tool.cli.toInput(parseFlags(tool.cli, args)) };
}

async function main(
  argv,
  env = process.env,
  fetchImpl = fetch,
  stdout = process.stdout,
  stderr = process.stderr,
) {
  if (Array.isArray(argv) && argv[0] === "task" && argv[1] === "help") {
    stdout.write(`${usage()}\n`);
    return 0;
  }
  try {
    const { tool, input } = parseCommand(argv);
    const result = await tool.handler(input, createClient(env, fetchImpl));
    stdout.write(`${JSON.stringify(result)}\n`);
  } catch (error) {
    stderr.write(`${error && error.message ? error.message : error}\n`);
    return 1;
  }
  return 0;
}

if (require.main === module) {
  main(process.argv.slice(2)).then((code) => process.exit(code));
}

module.exports = { main, matches, parseCommand, usage };
