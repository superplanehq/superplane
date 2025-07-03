import { normalizeModuleInfo } from "./module-info.js";
import { createContents } from "./on-load.js";
const PLUGIN_NAME = "global-externals";

/**
 * Create a `Plugin` for replacing modules with corresponding global variables.
 *
 * @param globals See type declaration.
 */
export const globalExternalsWithRegExp = (globals) => {
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
        const modulePath = args.path;
        const moduleInfo = normalizeModuleInfo(getModuleInfo(modulePath));
        return { contents: createContents(moduleInfo) };
      });
    },
  };
};
