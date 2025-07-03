"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.pnpPlugin = void 0;
const tslib_1 = require("tslib");
const fs = tslib_1.__importStar(require("fs"));
const path_1 = tslib_1.__importDefault(require("path"));
const matchAll = /()/;
const defaultExtensions = [`.tsx`, `.ts`, `.jsx`, `.mjs`, `.cjs`, `.js`, `.css`, `.json`];
// Reference: https://github.com/evanw/esbuild/blob/537195ae84bee1510fac14235906d588084c39cd/pkg/api/api_impl.go#L366-L388
function parseExternals(externals) {
    return externals.map(external => {
        // ESBuild's validation pass runs before this function is called so there's no need to assert that there's a single wildcard
        const wildcardIdx = external.indexOf(`*`);
        if (wildcardIdx !== -1)
            return { prefix: external.slice(0, wildcardIdx), suffix: external.slice(wildcardIdx + 1) };
        return external;
    });
}
function isExternal(path, externals) {
    for (const external of externals) {
        if (typeof external === `object`) {
            // Reference: https://github.com/evanw/esbuild/blob/537195ae84bee1510fac14235906d588084c39cd/internal/resolver/resolver.go#L372-L381
            if (path.length >= external.prefix.length + external.suffix.length
                && path.startsWith(external.prefix)
                && path.endsWith(external.suffix)) {
                return true;
            }
        }
        else {
            if (path === external)
                return true;
            // If the module "foo" has been marked as external, we also want to treat
            // paths into that module such as "foo/bar" as external too.
            // References:
            // - https://github.com/evanw/esbuild/blob/537195ae84bee1510fac14235906d588084c39cd/internal/resolver/resolver.go#L651-L652
            // - https://github.com/evanw/esbuild/blob/537195ae84bee1510fac14235906d588084c39cd/internal/resolver/resolver.go#L1688-L1691
            if (!external.startsWith(`/`) && !external.startsWith(`./`) && !external.startsWith(`../`) && external !== `.` && external !== `..`) {
                if (path.startsWith(`${external}/`)) {
                    return true;
                }
            }
        }
    }
    return false;
}
async function defaultOnLoad(args) {
    return {
        contents: await fs.promises.readFile(args.path),
        loader: `default`,
        // For regular imports in the `file` namespace, resolveDir is the directory the
        // file being resolved lives in. For all other virtual modules, this defaults to
        // empty string: ""
        // A sensible value for pnp imports is the same as the `file` namespace, as pnp
        // still resolves to files on disk (in the cache).
        resolveDir: path_1.default.dirname(args.path),
    };
}
async function defaultOnResolve(args, { resolvedPath, error, watchFiles }) {
    const problems = error ? [{ text: error.message }] : [];
    // Sometimes dynamic resolve calls might be wrapped in a try / catch,
    // but ESBuild neither skips them nor does it provide a way for us to tell.
    // Because of that, we downgrade all errors to warnings in these situations.
    // Issue: https://github.com/evanw/esbuild/issues/1127
    let mergeWith;
    switch (args.kind) {
        case `require-call`:
        case `require-resolve`:
        case `dynamic-import`:
            {
                mergeWith = { warnings: problems };
            }
            break;
        default:
            {
                mergeWith = { errors: problems };
            }
            break;
    }
    if (resolvedPath !== null) {
        return { namespace: `pnp`, path: resolvedPath, watchFiles };
    }
    else {
        return { external: true, ...mergeWith, watchFiles };
    }
}
function pnpPlugin({ baseDir = process.cwd(), extensions = defaultExtensions, filter = matchAll, onResolve = defaultOnResolve, onLoad = defaultOnLoad, } = {}) {
    return {
        name: `@yarnpkg/esbuild-plugin-pnp`,
        setup(build) {
            var _a, _b;
            const { findPnpApi } = require(`module`);
            if (typeof findPnpApi === `undefined`)
                return;
            const externals = parseExternals((_a = build.initialOptions.external) !== null && _a !== void 0 ? _a : []);
            const platform = (_b = build.initialOptions.platform) !== null && _b !== void 0 ? _b : `browser`;
            const isPlatformNode = platform === `node`;
            // Reference: https://github.com/evanw/esbuild/blob/537195ae84bee1510fac14235906d588084c39cd/internal/resolver/resolver.go#L238-L253
            const conditionsDefault = new Set(build.initialOptions.conditions);
            conditionsDefault.add(`default`);
            if (platform === `browser` || platform === `node`)
                conditionsDefault.add(platform);
            const conditionsImport = new Set(conditionsDefault);
            conditionsImport.add(`import`);
            const conditionsRequire = new Set(conditionsDefault);
            conditionsRequire.add(`require`);
            build.onResolve({ filter }, args => {
                var _a, _b;
                if (isExternal(args.path, externals))
                    return { external: true };
                // Reference: https://github.com/evanw/esbuild/blob/537195ae84bee1510fac14235906d588084c39cd/internal/resolver/resolver.go#L1495-L1502
                let conditions = conditionsDefault;
                if (args.kind === `dynamic-import` || args.kind === `import-statement`)
                    conditions = conditionsImport;
                else if (args.kind === `require-call` || args.kind === `require-resolve`)
                    conditions = conditionsRequire;
                // The entry point resolution uses an empty string
                const effectiveImporter = args.resolveDir
                    ? `${args.resolveDir}/`
                    : args.importer
                        ? args.importer
                        : `${baseDir}/`;
                const pnpApi = findPnpApi(effectiveImporter);
                if (!pnpApi)
                    // Path isn't controlled by PnP so delegate to the next resolver in the chain
                    return undefined;
                let path = null;
                let error;
                try {
                    path = pnpApi.resolveRequest(args.path, effectiveImporter, {
                        conditions,
                        considerBuiltins: isPlatformNode,
                        extensions,
                    });
                }
                catch (e) {
                    error = e;
                }
                const watchFiles = [pnpApi.resolveRequest(`pnpapi`, null)];
                if (path) {
                    const locator = pnpApi.findPackageLocator(path);
                    if (locator) {
                        const info = pnpApi.getPackageInformation(locator);
                        if ((info === null || info === void 0 ? void 0 : info.linkType) === `SOFT`) {
                            watchFiles.push((_b = (_a = pnpApi.resolveVirtual) === null || _a === void 0 ? void 0 : _a.call(pnpApi, path)) !== null && _b !== void 0 ? _b : path);
                        }
                    }
                }
                return onResolve(args, { resolvedPath: path, error, watchFiles });
            });
            // We register on the build to prevent ESBuild from reading the files
            // itself, since it wouldn't know how to access the files from within
            // the zip archives.
            if (build.onLoad !== null) {
                build.onLoad({ filter }, onLoad);
            }
        },
    };
}
exports.pnpPlugin = pnpPlugin;
