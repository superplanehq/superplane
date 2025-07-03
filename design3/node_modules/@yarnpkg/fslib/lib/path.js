"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toFilename = exports.convertPath = exports.ppath = exports.npath = exports.Filename = exports.PortablePath = void 0;
const tslib_1 = require("tslib");
const path_1 = tslib_1.__importDefault(require("path"));
var PathType;
(function (PathType) {
    PathType[PathType["File"] = 0] = "File";
    PathType[PathType["Portable"] = 1] = "Portable";
    PathType[PathType["Native"] = 2] = "Native";
})(PathType || (PathType = {}));
exports.PortablePath = {
    root: `/`,
    dot: `.`,
    parent: `..`,
};
exports.Filename = {
    nodeModules: `node_modules`,
    manifest: `package.json`,
    lockfile: `yarn.lock`,
    virtual: `__virtual__`,
    /**
     * @deprecated
     */
    pnpJs: `.pnp.js`,
    pnpCjs: `.pnp.cjs`,
    rc: `.yarnrc.yml`,
};
exports.npath = Object.create(path_1.default);
exports.ppath = Object.create(path_1.default.posix);
exports.npath.cwd = () => process.cwd();
exports.ppath.cwd = () => toPortablePath(process.cwd());
exports.ppath.resolve = (...segments) => {
    if (segments.length > 0 && exports.ppath.isAbsolute(segments[0])) {
        return path_1.default.posix.resolve(...segments);
    }
    else {
        return path_1.default.posix.resolve(exports.ppath.cwd(), ...segments);
    }
};
const contains = function (pathUtils, from, to) {
    from = pathUtils.normalize(from);
    to = pathUtils.normalize(to);
    if (from === to)
        return `.`;
    if (!from.endsWith(pathUtils.sep))
        from = (from + pathUtils.sep);
    if (to.startsWith(from)) {
        return to.slice(from.length);
    }
    else {
        return null;
    }
};
exports.npath.fromPortablePath = fromPortablePath;
exports.npath.toPortablePath = toPortablePath;
exports.npath.contains = (from, to) => contains(exports.npath, from, to);
exports.ppath.contains = (from, to) => contains(exports.ppath, from, to);
const WINDOWS_PATH_REGEXP = /^([a-zA-Z]:.*)$/;
const UNC_WINDOWS_PATH_REGEXP = /^\/\/(\.\/)?(.*)$/;
const PORTABLE_PATH_REGEXP = /^\/([a-zA-Z]:.*)$/;
const UNC_PORTABLE_PATH_REGEXP = /^\/unc\/(\.dot\/)?(.*)$/;
// Path should look like "/N:/berry/scripts/plugin-pack.js"
// And transform to "N:\berry\scripts\plugin-pack.js"
function fromPortablePath(p) {
    if (process.platform !== `win32`)
        return p;
    let portablePathMatch, uncPortablePathMatch;
    if ((portablePathMatch = p.match(PORTABLE_PATH_REGEXP)))
        p = portablePathMatch[1];
    else if ((uncPortablePathMatch = p.match(UNC_PORTABLE_PATH_REGEXP)))
        p = `\\\\${uncPortablePathMatch[1] ? `.\\` : ``}${uncPortablePathMatch[2]}`;
    else
        return p;
    return p.replace(/\//g, `\\`);
}
// Path should look like "N:/berry/scripts/plugin-pack.js"
// And transform to "/N:/berry/scripts/plugin-pack.js"
function toPortablePath(p) {
    if (process.platform !== `win32`)
        return p;
    p = p.replace(/\\/g, `/`);
    let windowsPathMatch, uncWindowsPathMatch;
    if ((windowsPathMatch = p.match(WINDOWS_PATH_REGEXP)))
        p = `/${windowsPathMatch[1]}`;
    else if ((uncWindowsPathMatch = p.match(UNC_WINDOWS_PATH_REGEXP)))
        p = `/unc/${uncWindowsPathMatch[1] ? `.dot/` : ``}${uncWindowsPathMatch[2]}`;
    return p;
}
function convertPath(targetPathUtils, sourcePath) {
    return (targetPathUtils === exports.npath ? fromPortablePath(sourcePath) : toPortablePath(sourcePath));
}
exports.convertPath = convertPath;
function toFilename(filename) {
    if (exports.npath.parse(filename).dir !== `` || exports.ppath.parse(filename).dir !== ``)
        throw new Error(`Invalid filename: "${filename}"`);
    return filename;
}
exports.toFilename = toFilename;
