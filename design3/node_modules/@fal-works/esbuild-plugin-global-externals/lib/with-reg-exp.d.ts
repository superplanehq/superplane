import type * as esbuild from "esbuild";
import type { ModuleInfo } from "./module-info";

/**
 * Object that contains a filter and a mapping function for replacing modules
 * with corresponding global variables.
 */
export declare type GlobalsMapper<T extends string = string> = {
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
export declare const globalExternalsWithRegExp: <T extends string>(
  globals: GlobalsMapper<T>
) => esbuild.Plugin;

//# sourceMappingURL=with-reg-exp.d.ts.map
