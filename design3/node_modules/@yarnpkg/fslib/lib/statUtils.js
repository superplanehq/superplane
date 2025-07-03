"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.areStatsEqual = exports.convertToBigIntStats = exports.clearStats = exports.makeEmptyStats = exports.makeDefaultStats = exports.BigIntStatsEntry = exports.StatEntry = exports.DirEntry = exports.DEFAULT_MODE = void 0;
const tslib_1 = require("tslib");
const nodeUtils = tslib_1.__importStar(require("util"));
const constants_1 = require("./constants");
exports.DEFAULT_MODE = constants_1.S_IFREG | 0o644;
class DirEntry {
    constructor() {
        this.name = ``;
        this.mode = 0;
    }
    isBlockDevice() {
        return false;
    }
    isCharacterDevice() {
        return false;
    }
    isDirectory() {
        return (this.mode & constants_1.S_IFMT) === constants_1.S_IFDIR;
    }
    isFIFO() {
        return false;
    }
    isFile() {
        return (this.mode & constants_1.S_IFMT) === constants_1.S_IFREG;
    }
    isSocket() {
        return false;
    }
    isSymbolicLink() {
        return (this.mode & constants_1.S_IFMT) === constants_1.S_IFLNK;
    }
}
exports.DirEntry = DirEntry;
class StatEntry {
    constructor() {
        this.uid = 0;
        this.gid = 0;
        this.size = 0;
        this.blksize = 0;
        this.atimeMs = 0;
        this.mtimeMs = 0;
        this.ctimeMs = 0;
        this.birthtimeMs = 0;
        this.atime = new Date(0);
        this.mtime = new Date(0);
        this.ctime = new Date(0);
        this.birthtime = new Date(0);
        this.dev = 0;
        this.ino = 0;
        this.mode = exports.DEFAULT_MODE;
        this.nlink = 1;
        this.rdev = 0;
        this.blocks = 1;
    }
    isBlockDevice() {
        return false;
    }
    isCharacterDevice() {
        return false;
    }
    isDirectory() {
        return (this.mode & constants_1.S_IFMT) === constants_1.S_IFDIR;
    }
    isFIFO() {
        return false;
    }
    isFile() {
        return (this.mode & constants_1.S_IFMT) === constants_1.S_IFREG;
    }
    isSocket() {
        return false;
    }
    isSymbolicLink() {
        return (this.mode & constants_1.S_IFMT) === constants_1.S_IFLNK;
    }
}
exports.StatEntry = StatEntry;
class BigIntStatsEntry {
    constructor() {
        this.uid = BigInt(0);
        this.gid = BigInt(0);
        this.size = BigInt(0);
        this.blksize = BigInt(0);
        this.atimeMs = BigInt(0);
        this.mtimeMs = BigInt(0);
        this.ctimeMs = BigInt(0);
        this.birthtimeMs = BigInt(0);
        this.atimeNs = BigInt(0);
        this.mtimeNs = BigInt(0);
        this.ctimeNs = BigInt(0);
        this.birthtimeNs = BigInt(0);
        this.atime = new Date(0);
        this.mtime = new Date(0);
        this.ctime = new Date(0);
        this.birthtime = new Date(0);
        this.dev = BigInt(0);
        this.ino = BigInt(0);
        this.mode = BigInt(exports.DEFAULT_MODE);
        this.nlink = BigInt(1);
        this.rdev = BigInt(0);
        this.blocks = BigInt(1);
    }
    isBlockDevice() {
        return false;
    }
    isCharacterDevice() {
        return false;
    }
    isDirectory() {
        return (this.mode & BigInt(constants_1.S_IFMT)) === BigInt(constants_1.S_IFDIR);
    }
    isFIFO() {
        return false;
    }
    isFile() {
        return (this.mode & BigInt(constants_1.S_IFMT)) === BigInt(constants_1.S_IFREG);
    }
    isSocket() {
        return false;
    }
    isSymbolicLink() {
        return (this.mode & BigInt(constants_1.S_IFMT)) === BigInt(constants_1.S_IFLNK);
    }
}
exports.BigIntStatsEntry = BigIntStatsEntry;
function makeDefaultStats() {
    return new StatEntry();
}
exports.makeDefaultStats = makeDefaultStats;
function makeEmptyStats() {
    return clearStats(makeDefaultStats());
}
exports.makeEmptyStats = makeEmptyStats;
/**
 * Mutates the provided stats object to zero it out then returns it for convenience
 */
function clearStats(stats) {
    for (const key in stats) {
        if (Object.prototype.hasOwnProperty.call(stats, key)) {
            const element = stats[key];
            if (typeof element === `number`) {
                // @ts-expect-error Typescript can't tell that stats[key] is a number
                stats[key] = 0;
            }
            else if (typeof element === `bigint`) {
                // @ts-expect-error Typescript can't tell that stats[key] is a bigint
                stats[key] = BigInt(0);
            }
            else if (nodeUtils.types.isDate(element)) {
                // @ts-expect-error Typescript can't tell that stats[key] is a bigint
                stats[key] = new Date(0);
            }
        }
    }
    return stats;
}
exports.clearStats = clearStats;
function convertToBigIntStats(stats) {
    const bigintStats = new BigIntStatsEntry();
    for (const key in stats) {
        if (Object.prototype.hasOwnProperty.call(stats, key)) {
            const element = stats[key];
            if (typeof element === `number`) {
                // @ts-expect-error Typescript isn't able to tell this is valid
                bigintStats[key] = BigInt(element);
            }
            else if (nodeUtils.types.isDate(element)) {
                // @ts-expect-error Typescript isn't able to tell this is valid
                bigintStats[key] = new Date(element);
            }
        }
    }
    bigintStats.atimeNs = bigintStats.atimeMs * BigInt(1e6);
    bigintStats.mtimeNs = bigintStats.mtimeMs * BigInt(1e6);
    bigintStats.ctimeNs = bigintStats.ctimeMs * BigInt(1e6);
    bigintStats.birthtimeNs = bigintStats.birthtimeMs * BigInt(1e6);
    return bigintStats;
}
exports.convertToBigIntStats = convertToBigIntStats;
function areStatsEqual(a, b) {
    if (a.atimeMs !== b.atimeMs)
        return false;
    if (a.birthtimeMs !== b.birthtimeMs)
        return false;
    if (a.blksize !== b.blksize)
        return false;
    if (a.blocks !== b.blocks)
        return false;
    if (a.ctimeMs !== b.ctimeMs)
        return false;
    if (a.dev !== b.dev)
        return false;
    if (a.gid !== b.gid)
        return false;
    if (a.ino !== b.ino)
        return false;
    if (a.isBlockDevice() !== b.isBlockDevice())
        return false;
    if (a.isCharacterDevice() !== b.isCharacterDevice())
        return false;
    if (a.isDirectory() !== b.isDirectory())
        return false;
    if (a.isFIFO() !== b.isFIFO())
        return false;
    if (a.isFile() !== b.isFile())
        return false;
    if (a.isSocket() !== b.isSocket())
        return false;
    if (a.isSymbolicLink() !== b.isSymbolicLink())
        return false;
    if (a.mode !== b.mode)
        return false;
    if (a.mtimeMs !== b.mtimeMs)
        return false;
    if (a.nlink !== b.nlink)
        return false;
    if (a.rdev !== b.rdev)
        return false;
    if (a.size !== b.size)
        return false;
    if (a.uid !== b.uid)
        return false;
    const aN = a;
    const bN = b;
    if (aN.atimeNs !== bN.atimeNs)
        return false;
    if (aN.mtimeNs !== bN.mtimeNs)
        return false;
    if (aN.ctimeNs !== bN.ctimeNs)
        return false;
    if (aN.birthtimeNs !== bN.birthtimeNs)
        return false;
    return true;
}
exports.areStatsEqual = areStatsEqual;
