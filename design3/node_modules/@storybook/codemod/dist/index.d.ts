export { packageNames, default as updateOrganisationName } from './transforms/update-organisation-name.js';
export { default as updateAddonInfo } from './transforms/update-addon-info.js';

declare function listCodemods(): string[];
declare function runCodemod(codemod: any, { glob, logger, dryRun, rename, parser }: any): Promise<void>;

export { listCodemods, runCodemod };
