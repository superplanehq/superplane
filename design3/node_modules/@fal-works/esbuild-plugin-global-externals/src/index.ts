export type { ModuleType, ModuleInfo } from "./module-info";

import { globalExternals } from "./with-object.js";
export { globalExternals };
export default globalExternals;

export { globalExternalsWithRegExp } from "./with-reg-exp.js";
export type { GlobalsMapper } from "./with-reg-exp";
