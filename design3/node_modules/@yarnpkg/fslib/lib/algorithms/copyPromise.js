"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.copyPromise = exports.LinkStrategy = void 0;
const tslib_1 = require("tslib");
const fs_1 = tslib_1.__importDefault(require("fs"));
const constants = tslib_1.__importStar(require("../constants"));
const path_1 = require("../path");
const defaultTime = new Date(constants.SAFE_TIME * 1000);
var LinkStrategy;
(function (LinkStrategy) {
    LinkStrategy["Allow"] = "allow";
    LinkStrategy["ReadOnly"] = "readOnly";
})(LinkStrategy = exports.LinkStrategy || (exports.LinkStrategy = {}));
async function copyPromise(destinationFs, destination, sourceFs, source, opts) {
    const normalizedDestination = destinationFs.pathUtils.normalize(destination);
    const normalizedSource = sourceFs.pathUtils.normalize(source);
    const prelayout = [];
    const postlayout = [];
    const { atime, mtime } = opts.stableTime
        ? { atime: defaultTime, mtime: defaultTime }
        : await sourceFs.lstatPromise(normalizedSource);
    await destinationFs.mkdirpPromise(destinationFs.pathUtils.dirname(destination), { utimes: [atime, mtime] });
    const updateTime = typeof destinationFs.lutimesPromise === `function`
        ? destinationFs.lutimesPromise.bind(destinationFs)
        : destinationFs.utimesPromise.bind(destinationFs);
    await copyImpl(prelayout, postlayout, updateTime, destinationFs, normalizedDestination, sourceFs, normalizedSource, { ...opts, didParentExist: true });
    for (const operation of prelayout)
        await operation();
    await Promise.all(postlayout.map(operation => {
        return operation();
    }));
}
exports.copyPromise = copyPromise;
async function copyImpl(prelayout, postlayout, updateTime, destinationFs, destination, sourceFs, source, opts) {
    var _a, _b;
    const destinationStat = opts.didParentExist ? await maybeLStat(destinationFs, destination) : null;
    const sourceStat = await sourceFs.lstatPromise(source);
    const { atime, mtime } = opts.stableTime
        ? { atime: defaultTime, mtime: defaultTime }
        : sourceStat;
    let updated;
    switch (true) {
        case sourceStat.isDirectory():
            {
                updated = await copyFolder(prelayout, postlayout, updateTime, destinationFs, destination, destinationStat, sourceFs, source, sourceStat, opts);
            }
            break;
        case sourceStat.isFile():
            {
                updated = await copyFile(prelayout, postlayout, updateTime, destinationFs, destination, destinationStat, sourceFs, source, sourceStat, opts);
            }
            break;
        case sourceStat.isSymbolicLink():
            {
                updated = await copySymlink(prelayout, postlayout, updateTime, destinationFs, destination, destinationStat, sourceFs, source, sourceStat, opts);
            }
            break;
        default:
            {
                throw new Error(`Unsupported file type (${sourceStat.mode})`);
            }
            break;
    }
    if (updated || ((_a = destinationStat === null || destinationStat === void 0 ? void 0 : destinationStat.mtime) === null || _a === void 0 ? void 0 : _a.getTime()) !== mtime.getTime() || ((_b = destinationStat === null || destinationStat === void 0 ? void 0 : destinationStat.atime) === null || _b === void 0 ? void 0 : _b.getTime()) !== atime.getTime()) {
        postlayout.push(() => updateTime(destination, atime, mtime));
        updated = true;
    }
    if (destinationStat === null || (destinationStat.mode & 0o777) !== (sourceStat.mode & 0o777)) {
        postlayout.push(() => destinationFs.chmodPromise(destination, sourceStat.mode & 0o777));
        updated = true;
    }
    return updated;
}
async function maybeLStat(baseFs, p) {
    try {
        return await baseFs.lstatPromise(p);
    }
    catch (e) {
        return null;
    }
}
async function copyFolder(prelayout, postlayout, updateTime, destinationFs, destination, destinationStat, sourceFs, source, sourceStat, opts) {
    if (destinationStat !== null && !destinationStat.isDirectory()) {
        if (opts.overwrite) {
            prelayout.push(async () => destinationFs.removePromise(destination));
            destinationStat = null;
        }
        else {
            return false;
        }
    }
    let updated = false;
    if (destinationStat === null) {
        prelayout.push(async () => {
            try {
                await destinationFs.mkdirPromise(destination, { mode: sourceStat.mode });
            }
            catch (err) {
                if (err.code !== `EEXIST`) {
                    throw err;
                }
            }
        });
        updated = true;
    }
    const entries = await sourceFs.readdirPromise(source);
    const nextOpts = opts.didParentExist && !destinationStat ? { ...opts, didParentExist: false } : opts;
    if (opts.stableSort) {
        for (const entry of entries.sort()) {
            if (await copyImpl(prelayout, postlayout, updateTime, destinationFs, destinationFs.pathUtils.join(destination, entry), sourceFs, sourceFs.pathUtils.join(source, entry), nextOpts)) {
                updated = true;
            }
        }
    }
    else {
        const entriesUpdateStatus = await Promise.all(entries.map(async (entry) => {
            await copyImpl(prelayout, postlayout, updateTime, destinationFs, destinationFs.pathUtils.join(destination, entry), sourceFs, sourceFs.pathUtils.join(source, entry), nextOpts);
        }));
        if (entriesUpdateStatus.some(status => status)) {
            updated = true;
        }
    }
    return updated;
}
const isCloneSupportedCache = new WeakMap();
function makeLinkOperation(opFs, destination, source, sourceStat, linkStrategy) {
    return async () => {
        await opFs.linkPromise(source, destination);
        if (linkStrategy === LinkStrategy.ReadOnly) {
            // We mutate the stat, otherwise it'll be reset by copyImpl
            sourceStat.mode &= ~0o222;
            await opFs.chmodPromise(destination, sourceStat.mode);
        }
    };
}
function makeCloneLinkOperation(opFs, destination, source, sourceStat, linkStrategy) {
    const isCloneSupported = isCloneSupportedCache.get(opFs);
    if (typeof isCloneSupported === `undefined`) {
        return async () => {
            try {
                await opFs.copyFilePromise(source, destination, fs_1.default.constants.COPYFILE_FICLONE_FORCE);
                isCloneSupportedCache.set(opFs, true);
            }
            catch (err) {
                if (err.code === `ENOSYS` || err.code === `ENOTSUP`) {
                    isCloneSupportedCache.set(opFs, false);
                    await makeLinkOperation(opFs, destination, source, sourceStat, linkStrategy)();
                }
                else {
                    throw err;
                }
            }
        };
    }
    else {
        if (isCloneSupported) {
            return async () => opFs.copyFilePromise(source, destination, fs_1.default.constants.COPYFILE_FICLONE_FORCE);
        }
        else {
            return makeLinkOperation(opFs, destination, source, sourceStat, linkStrategy);
        }
    }
}
async function copyFile(prelayout, postlayout, updateTime, destinationFs, destination, destinationStat, sourceFs, source, sourceStat, opts) {
    var _a;
    if (destinationStat !== null) {
        if (opts.overwrite) {
            prelayout.push(async () => destinationFs.removePromise(destination));
            destinationStat = null;
        }
        else {
            return false;
        }
    }
    const linkStrategy = (_a = opts.linkStrategy) !== null && _a !== void 0 ? _a : null;
    const op = destinationFs === sourceFs
        ? linkStrategy !== null
            ? makeCloneLinkOperation(destinationFs, destination, source, sourceStat, linkStrategy)
            : async () => destinationFs.copyFilePromise(source, destination, fs_1.default.constants.COPYFILE_FICLONE)
        : linkStrategy !== null
            ? makeLinkOperation(destinationFs, destination, source, sourceStat, linkStrategy)
            : async () => destinationFs.writeFilePromise(destination, await sourceFs.readFilePromise(source));
    prelayout.push(async () => op());
    return true;
}
async function copySymlink(prelayout, postlayout, updateTime, destinationFs, destination, destinationStat, sourceFs, source, sourceStat, opts) {
    if (destinationStat !== null) {
        if (opts.overwrite) {
            prelayout.push(async () => destinationFs.removePromise(destination));
            destinationStat = null;
        }
        else {
            return false;
        }
    }
    prelayout.push(async () => {
        await destinationFs.symlinkPromise((0, path_1.convertPath)(destinationFs.pathUtils, await sourceFs.readlinkPromise(source)), destination);
    });
    return true;
}
