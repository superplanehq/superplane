import { globalExternalsWithRegExp } from "./with-reg-exp.js";

/**
 * Create a `Plugin` for replacing modules with corresponding global variables.
 *
 * @param globals Object that maps between the two below:
 *
 * - From: Module path used in any `import` statements that should be replaced
 *   with a global variable.
 * - To: String for a global variable name, or any `ModuleInfo` object
 *   which also includes the global variable name.
 *
 * @example
 *
 * ```
 * const plugins = [globalExternals({ jquery: "$" })];
 * ```
 */
export const globalExternals = (globals) => {
  const normalizedGlobals = {
    modulePathFilter: new RegExp(`^(?:${Object.keys(globals).join("|")})$`),
    getModuleInfo: (modulePath) => globals[modulePath],
  };
  return globalExternalsWithRegExp(normalizedGlobals);
};
