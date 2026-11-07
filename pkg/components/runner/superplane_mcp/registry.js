"use strict";

/**
 * Tool modules of the global runner MCP server. Every tool declares an MCP
 * definition and one handler. The backend catalog (runner_tools.go) decides
 * which tools a run may use. A module can export prepare(client), which runs
 * once when the run may use one of its tools.
 */

const MODULES = [
  require("./modules/members"),
  require("./modules/tasks"),
  require("./modules/checks"),
  require("./modules/artifacts"),
  require("./modules/plan"),
  require("./modules/evidence"),
  require("./modules/merge"),
  require("./modules/pull_requests"),
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

// stop ends work that a tool runs outside this process group, for example a
// git push in its own session. A module can export stop() for that work.
function stop(modules = MODULES) {
  for (const module of modules) {
    if (typeof module.stop === "function") module.stop();
  }
}

// stopOnExit stops that work when this process ends, on a termination signal,
// or when the agent closes the input stream.
function stopOnExit(input, modules = MODULES) {
  const shutdown = (code) => () => {
    stop(modules);
    process.exit(code);
  };
  if (input) input.on("end", shutdown(0));
  for (const signal of ["SIGTERM", "SIGINT", "SIGHUP"]) process.once(signal, shutdown(1));
  process.once("exit", () => stop(modules));
}

module.exports = { MODULES, allTools, findTool, prepare, stop, stopOnExit };
