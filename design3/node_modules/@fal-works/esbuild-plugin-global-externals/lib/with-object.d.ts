import type * as esbuild from "esbuild";
import type { ModuleInfo } from "./module-info";

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
export declare const globalExternals: <T extends string>(
  globals: Record<T, string | ModuleInfo>
) => esbuild.Plugin;

//# sourceMappingURL=with-object.d.ts.map
