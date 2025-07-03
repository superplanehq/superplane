import type * as esbuild from "esbuild";
import type { ModuleInfo } from "./module-info";
import { normalizeModuleInfo } from "./module-info.js";
import { createContents } from "./on-load.js";

const PLUGIN_NAME = "global-externals";

/**
 * Object that contains a filter and a mapping function for replacing modules
 * with corresponding global variables.
 */
export type GlobalsMapper<T extends string = string> = {
  /**
   * Regular expression that match module paths to be processed by this plugin.
   *
   * @see <https://esbuild.github.io/plugins/>
   */
  modulePathFilter: RegExp;

  /**
   * Function that returns either a corresponding global variable name or a
   * `ModuleInfo` object for the module at `modulePath`.
   */
  getModuleInfo: (modulePath: T) => string | ModuleInfo;
};

/**
 * Create a `Plugin` for replacing modules with corresponding global variables.
 *
 * @param globals See type declaration.
 */
export const globalExternalsWithRegExp = <T extends string>(
  globals: GlobalsMapper<T>
): esbuild.Plugin => {
  const { modulePathFilter, getModuleInfo } = globals;

  return {
    name: PLUGIN_NAME,
    setup(build) {
      build.onResolve({ filter: modulePathFilter }, (args) => ({
        path: args.path,
        namespace: PLUGIN_NAME,
      }));

      build.onLoad({ filter: /.*/, namespace: PLUGIN_NAME }, (args) => {
        // eslint-disable-next-line total-functions/no-unsafe-type-assertion
        const modulePath = args.path as T;
        const moduleInfo = normalizeModuleInfo(getModuleInfo(modulePath));
        return { contents: createContents(moduleInfo) };
      });
    },
  };
};
