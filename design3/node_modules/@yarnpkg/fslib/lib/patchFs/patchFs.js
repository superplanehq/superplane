"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.extendFs = exports.patchFs = void 0;
const util_1 = require("util");
const NodePathFS_1 = require("../NodePathFS");
const FileHandle_1 = require("./FileHandle");
const SYNC_IMPLEMENTATIONS = new Set([
    `accessSync`,
    `appendFileSync`,
    `createReadStream`,
    `createWriteStream`,
    `chmodSync`,
    `fchmodSync`,
    `chownSync`,
    `fchownSync`,
    `closeSync`,
    `copyFileSync`,
    `linkSync`,
    `lstatSync`,
    `fstatSync`,
    `lutimesSync`,
    `mkdirSync`,
    `openSync`,
    `opendirSync`,
    `readlinkSync`,
    `readFileSync`,
    `readdirSync`,
    `readlinkSync`,
    `realpathSync`,
    `renameSync`,
    `rmdirSync`,
    `statSync`,
    `symlinkSync`,
    `truncateSync`,
    `ftruncateSync`,
    `unlinkSync`,
    `unwatchFile`,
    `utimesSync`,
    `watch`,
    `watchFile`,
    `writeFileSync`,
    `writeSync`,
]);
const ASYNC_IMPLEMENTATIONS = new Set([
    `accessPromise`,
    `appendFilePromise`,
    `fchmodPromise`,
    `chmodPromise`,
    `fchownPromise`,
    `chownPromise`,
    `closePromise`,
    `copyFilePromise`,
    `linkPromise`,
    `fstatPromise`,
    `lstatPromise`,
    `lutimesPromise`,
    `mkdirPromise`,
    `openPromise`,
    `opendirPromise`,
    `readdirPromise`,
    `realpathPromise`,
    `readFilePromise`,
    `readdirPromise`,
    `readlinkPromise`,
    `renamePromise`,
    `rmdirPromise`,
    `statPromise`,
    `symlinkPromise`,
    `truncatePromise`,
    `ftruncatePromise`,
    `unlinkPromise`,
    `utimesPromise`,
    `writeFilePromise`,
    `writeSync`,
]);
//#endregion
function patchFs(patchedFs, fakeFs) {
    // We wrap the `fakeFs` with a `NodePathFS` to add support for all path types supported by Node
    fakeFs = new NodePathFS_1.NodePathFS(fakeFs);
    const setupFn = (target, name, replacement) => {
        const orig = target[name];
        target[name] = replacement;
        // Preserve any util.promisify implementations
        if (typeof (orig === null || orig === void 0 ? void 0 : orig[util_1.promisify.custom]) !== `undefined`) {
            replacement[util_1.promisify.custom] = orig[util_1.promisify.custom];
        }
    };
    /** Callback implementations */
    {
        setupFn(patchedFs, `exists`, (p, ...args) => {
            const hasCallback = typeof args[args.length - 1] === `function`;
            const callback = hasCallback ? args.pop() : () => { };
            process.nextTick(() => {
                fakeFs.existsPromise(p).then(exists => {
                    callback(exists);
                }, () => {
                    callback(false);
                });
            });
        });
        // Adapted from https://github.com/nodejs/node/blob/e5c1fd7a2a1801fd75bdde23b260488e85453eb2/lib/fs.js#L603-L667
        setupFn(patchedFs, `read`, (...args) => {
            let [fd, buffer, offset, length, position, callback] = args;
            if (args.length <= 3) {
                // Assume fs.read(fd, options, callback)
                let options = {};
                if (args.length < 3) {
                    // This is fs.read(fd, callback)
                    callback = args[1];
                }
                else {
                    // This is fs.read(fd, {}, callback)
                    options = args[1];
                    callback = args[2];
                }
                ({
                    buffer = Buffer.alloc(16384),
                    offset = 0,
                    length = buffer.byteLength,
                    position,
                } = options);
            }
            if (offset == null)
                offset = 0;
            length |= 0;
            if (length === 0) {
                process.nextTick(() => {
                    callback(null, 0, buffer);
                });
                return;
            }
            if (position == null)
                position = -1;
            process.nextTick(() => {
                fakeFs.readPromise(fd, buffer, offset, length, position).then(bytesRead => {
                    callback(null, bytesRead, buffer);
                }, error => {
                    // https://github.com/nodejs/node/blob/1317252dfe8824fd9cfee125d2aaa94004db2f3b/lib/fs.js#L655-L658
                    // Known issue: bytesRead could theoretically be > than 0, but we currently always return 0
                    callback(error, 0, buffer);
                });
            });
        });
        for (const fnName of ASYNC_IMPLEMENTATIONS) {
            const origName = fnName.replace(/Promise$/, ``);
            if (typeof patchedFs[origName] === `undefined`)
                continue;
            const fakeImpl = fakeFs[fnName];
            if (typeof fakeImpl === `undefined`)
                continue;
            const wrapper = (...args) => {
                const hasCallback = typeof args[args.length - 1] === `function`;
                const callback = hasCallback ? args.pop() : () => { };
                process.nextTick(() => {
                    fakeImpl.apply(fakeFs, args).then((result) => {
                        callback(null, result);
                    }, (error) => {
                        callback(error);
                    });
                });
            };
            setupFn(patchedFs, origName, wrapper);
        }
        patchedFs.realpath.native = patchedFs.realpath;
    }
    /** Sync implementations */
    {
        setupFn(patchedFs, `existsSync`, (p) => {
            try {
                return fakeFs.existsSync(p);
            }
            catch (error) {
                return false;
            }
        });
        // Adapted from https://github.com/nodejs/node/blob/e5c1fd7a2a1801fd75bdde23b260488e85453eb2/lib/fs.js#L684-L725
        setupFn(patchedFs, `readSync`, (...args) => {
            let [fd, buffer, offset, length, position] = args;
            if (args.length <= 3) {
                // Assume fs.read(fd, buffer, options)
                const options = args[2] || {};
                ({ offset = 0, length = buffer.byteLength, position } = options);
            }
            if (offset == null)
                offset = 0;
            length |= 0;
            if (length === 0)
                return 0;
            if (position == null)
                position = -1;
            return fakeFs.readSync(fd, buffer, offset, length, position);
        });
        for (const fnName of SYNC_IMPLEMENTATIONS) {
            const origName = fnName;
            if (typeof patchedFs[origName] === `undefined`)
                continue;
            const fakeImpl = fakeFs[fnName];
            if (typeof fakeImpl === `undefined`)
                continue;
            setupFn(patchedFs, origName, fakeImpl.bind(fakeFs));
        }
        patchedFs.realpathSync.native = patchedFs.realpathSync;
    }
    /** Promise implementations */
    {
        // `fs.promises` is a getter that returns a reference to require(`fs/promises`),
        // so we can just patch `fs.promises` and both will be updated
        const origEmitWarning = process.emitWarning;
        process.emitWarning = () => { };
        let patchedFsPromises;
        try {
            patchedFsPromises = patchedFs.promises;
        }
        finally {
            process.emitWarning = origEmitWarning;
        }
        if (typeof patchedFsPromises !== `undefined`) {
            // `fs.promises.exists` doesn't exist
            for (const fnName of ASYNC_IMPLEMENTATIONS) {
                const origName = fnName.replace(/Promise$/, ``);
                if (typeof patchedFsPromises[origName] === `undefined`)
                    continue;
                const fakeImpl = fakeFs[fnName];
                if (typeof fakeImpl === `undefined`)
                    continue;
                // Open is a bit particular with fs.promises: it returns a file handle
                // instance instead of the traditional file descriptor number
                if (fnName === `open`)
                    continue;
                setupFn(patchedFsPromises, origName, (pathLike, ...args) => {
                    if (pathLike instanceof FileHandle_1.FileHandle) {
                        return pathLike[origName].apply(pathLike, args);
                    }
                    else {
                        return fakeImpl.call(fakeFs, pathLike, ...args);
                    }
                });
            }
            setupFn(patchedFsPromises, `open`, async (...args) => {
                // @ts-expect-error
                const fd = await fakeFs.openPromise(...args);
                return new FileHandle_1.FileHandle(fd, fakeFs);
            });
            // `fs.promises.realpath` doesn't have a `native` property
        }
    }
    /** util.promisify implementations */
    {
        // TODO add promisified `fs.readv` and `fs.writev`, once they are implemented
        // Override the promisified versions of `fs.read` and `fs.write` to return an object as per
        // https://github.com/nodejs/node/blob/dc79f3f37caf6f25b8efee4623bec31e2c20f595/lib/fs.js#L559-L560
        // and
        // https://github.com/nodejs/node/blob/dc79f3f37caf6f25b8efee4623bec31e2c20f595/lib/fs.js#L690-L691
        // and
        // https://github.com/nodejs/node/blob/ba684805b6c0eded76e5cd89ee00328ac7a59365/lib/internal/util.js#L293
        // @ts-expect-error
        patchedFs.read[util_1.promisify.custom] = async (fd, buffer, ...args) => {
            const res = fakeFs.readPromise(fd, buffer, ...args);
            return { bytesRead: await res, buffer };
        };
        // @ts-expect-error
        patchedFs.write[util_1.promisify.custom] = async (fd, buffer, ...args) => {
            const res = fakeFs.writePromise(fd, buffer, ...args);
            return { bytesWritten: await res, buffer };
        };
    }
}
exports.patchFs = patchFs;
function extendFs(realFs, fakeFs) {
    const patchedFs = Object.create(realFs);
    patchFs(patchedFs, fakeFs);
    return patchedFs;
}
exports.extendFs = extendFs;
