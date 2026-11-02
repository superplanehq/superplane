"use strict";

/**
 * Tool modules of the global runner MCP server. Every tool declares an MCP
 * definition and one handler. The backend catalog (runner_tools.go) decides
 * which tools a run may use. A module can export prepare(client), which runs
 * once when the run may use one of its tools.
 *
 * A tool can declare aliases: previous tool names that stay callable but are
 * not listed. An alias can declare its own handler when it takes the previous
 * input.
 */

const MODULES = [
  require("./modules/members"),
  require("./modules/tasks"),
  require("./modules/checks"),
  require("./modules/artifacts"),
  require("./modules/plan"),
  require("./modules/evidence"),
  require("./modules/merge"),
];

function allTools(modules = MODULES) {
  return modules.flatMap((module) => module.tools);
}

function toolNames(tool) {
  return [tool.definition.name, ...(tool.aliases || []).map((alias) => alias.name)];
}

function allToolNames(modules = MODULES) {
  return allTools(modules).flatMap(toolNames);
}

// isAllowed reports whether a run may use the tool under its name or an alias.
function isAllowed(tool, allowed) {
  return toolNames(tool).some((name) => allowed.has(name));
}

function resolve(name, tools) {
  for (const tool of tools) {
    if (tool.definition.name === name) return { tool, handler: tool.handler };
    const alias = (tool.aliases || []).find((candidate) => candidate.name === name);
    if (alias) return { tool, handler: alias.handler || tool.handler };
  }
  return undefined;
}

function findTool(name, modules = MODULES) {
  const resolved = resolve(name, allTools(modules));
  return resolved && resolved.tool;
}

function prepare(allowed, client, modules = MODULES) {
  for (const module of modules) {
    if (typeof module.prepare !== "function") continue;
    if (!module.tools.some((tool) => isAllowed(tool, allowed))) continue;
    module.prepare(client);
  }
}

module.exports = { MODULES, allTools, allToolNames, findTool, isAllowed, prepare, resolve };
