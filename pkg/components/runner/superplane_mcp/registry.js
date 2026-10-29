"use strict";

/**
 * Tool modules of the global runner MCP server. Every tool declares an MCP
 * definition and one handler. The backend catalog (runner_tools.go) decides
 * which tools a run may use.
 */

const MODULES = [require("./modules/owner")];

function allTools(modules = MODULES) {
  return modules.flatMap((module) => module.tools);
}

function findTool(name, modules = MODULES) {
  return allTools(modules).find((tool) => tool.definition.name === name);
}

module.exports = { allTools, findTool };
