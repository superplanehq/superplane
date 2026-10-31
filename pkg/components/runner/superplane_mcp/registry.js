"use strict";

/**
 * Tool modules of the global runner MCP server. Every tool declares an MCP
 * definition and one handler. The backend catalog (runner_tools.go) decides
 * which tools a run may use. A module can export prepare(client), which runs
 * once when the run may use one of its tools.
 */

const MODULES = [
  require("./modules/owner"),
  require("./modules/tasks"),
  require("./modules/artifacts"),
  require("./modules/plan"),
  require("./modules/evidence"),
  require("./modules/links"),
  require("./modules/merge"),
];

function allTools(modules = MODULES) {
  return modules.flatMap((module) => module.tools);
}

function findTool(name, modules = MODULES) {
  return allTools(modules).find((tool) => tool.definition.name === name);
}

function prepare(allowed, client, modules = MODULES) {
  for (const module of modules) {
    if (typeof module.prepare !== "function") continue;
    if (!module.tools.some((tool) => allowed.has(tool.definition.name))) continue;
    module.prepare(client);
  }
}

module.exports = { MODULES, allTools, findTool, prepare };
