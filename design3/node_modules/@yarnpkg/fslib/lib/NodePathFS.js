"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.NodePathFS = void 0;
const url_1 = require("url");
const util_1 = require("util");
const ProxiedFS_1 = require("./ProxiedFS");
const path_1 = require("./path");
/**
 * Adds support for file URLs and Buffers to the wrapped `baseFs`, but *not* inside the typings.
 *
 * Only exists for compatibility with Node's behavior.
 *
 * Automatically wraps all FS instances passed to `patchFs` & `extendFs`.
 *
 * Don't use it!
 */
class NodePathFS extends ProxiedFS_1.ProxiedFS {
    constructor(baseFs) {
        super(path_1.npath);
        this.baseFs = baseFs;
    }
    mapFromBase(path) {
        return path;
    }
    mapToBase(path) {
        if (typeof path === `string`)
            return path;
        if (path instanceof url_1.URL)
            return (0, url_1.fileURLToPath)(path);
        if (Buffer.isBuffer(path)) {
            const str = path.toString();
            if (Buffer.byteLength(str) !== path.byteLength)
                throw new Error(`Non-utf8 buffers are not supported at the moment. Please upvote the following issue if you encounter this error: https://github.com/yarnpkg/berry/issues/4942`);
            return str;
        }
        throw new Error(`Unsupported path type: ${(0, util_1.inspect)(path)}`);
    }
}
exports.NodePathFS = NodePathFS;
