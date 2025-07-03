"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ZipOpenFS = exports.getArchivePart = void 0;
const tslib_1 = require("tslib");
const fs_1 = require("fs");
const FakeFS_1 = require("./FakeFS");
const NodeFS_1 = require("./NodeFS");
const ZipFS_1 = require("./ZipFS");
const watchFile_1 = require("./algorithms/watchFile");
const errors = tslib_1.__importStar(require("./errors"));
const path_1 = require("./path");
// Only file descriptors prefixed by those values will be forwarded to the ZipFS
// instances. Note that the highest ZIP_MAGIC bit MUST NOT be set, otherwise the
// resulting fd becomes a negative integer, which isn't supposed to happen per
// the unix rules (caused problems w/ Go).
//
// Those values must be synced with packages/yarnpkg-pnp/sources/esm-loader/fspatch.ts
//
const ZIP_MASK = 0xff000000;
const ZIP_MAGIC = 0x2a000000;
/**
 * Extracts the archive part (ending in the first instance of `extension`) from a path.
 *
 * The indexOf-based implementation is ~3.7x faster than a RegExp-based implementation.
 */
const getArchivePart = (path, extension) => {
    let idx = path.indexOf(extension);
    if (idx <= 0)
        return null;
    let nextCharIdx = idx;
    while (idx >= 0) {
        nextCharIdx = idx + extension.length;
        if (path[nextCharIdx] === path_1.ppath.sep)
            break;
        // Disallow files named ".zip"
        if (path[idx - 1] === path_1.ppath.sep)
            return null;
        idx = path.indexOf(extension, nextCharIdx);
    }
    // The path either has to end in ".zip" or contain an archive subpath (".zip/...")
    if (path.length > nextCharIdx && path[nextCharIdx] !== path_1.ppath.sep)
        return null;
    return path.slice(0, nextCharIdx);
};
exports.getArchivePart = getArchivePart;
class ZipOpenFS extends FakeFS_1.BasePortableFakeFS {
    static async openPromise(fn, opts) {
        const zipOpenFs = new ZipOpenFS(opts);
        try {
            return await fn(zipOpenFs);
        }
        finally {
            zipOpenFs.saveAndClose();
        }
    }
    get libzip() {
        if (typeof this.libzipInstance === `undefined`)
            this.libzipInstance = this.libzipFactory();
        return this.libzipInstance;
    }
    constructor({ libzip, baseFs = new NodeFS_1.NodeFS(), filter = null, maxOpenFiles = Infinity, readOnlyArchives = false, useCache = true, maxAge = 5000, fileExtensions = null }) {
        super();
        this.fdMap = new Map();
        this.nextFd = 3;
        this.isZip = new Set();
        this.notZip = new Set();
        this.realPaths = new Map();
        this.limitOpenFilesTimeout = null;
        this.libzipFactory = typeof libzip !== `function`
            ? () => libzip
            : libzip;
        this.baseFs = baseFs;
        this.zipInstances = useCache ? new Map() : null;
        this.filter = filter;
        this.maxOpenFiles = maxOpenFiles;
        this.readOnlyArchives = readOnlyArchives;
        this.maxAge = maxAge;
        this.fileExtensions = fileExtensions;
    }
    getExtractHint(hints) {
        return this.baseFs.getExtractHint(hints);
    }
    getRealPath() {
        return this.baseFs.getRealPath();
    }
    saveAndClose() {
        (0, watchFile_1.unwatchAllFiles)(this);
        if (this.zipInstances) {
            for (const [path, { zipFs }] of this.zipInstances.entries()) {
                zipFs.saveAndClose();
                this.zipInstances.delete(path);
            }
        }
    }
    discardAndClose() {
        (0, watchFile_1.unwatchAllFiles)(this);
        if (this.zipInstances) {
            for (const [path, { zipFs }] of this.zipInstances.entries()) {
                zipFs.discardAndClose();
                this.zipInstances.delete(path);
            }
        }
    }
    resolve(p) {
        return this.baseFs.resolve(p);
    }
    remapFd(zipFs, fd) {
        const remappedFd = this.nextFd++ | ZIP_MAGIC;
        this.fdMap.set(remappedFd, [zipFs, fd]);
        return remappedFd;
    }
    async openPromise(p, flags, mode) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.openPromise(p, flags, mode);
        }, async (zipFs, { subPath }) => {
            return this.remapFd(zipFs, await zipFs.openPromise(subPath, flags, mode));
        });
    }
    openSync(p, flags, mode) {
        return this.makeCallSync(p, () => {
            return this.baseFs.openSync(p, flags, mode);
        }, (zipFs, { subPath }) => {
            return this.remapFd(zipFs, zipFs.openSync(subPath, flags, mode));
        });
    }
    async opendirPromise(p, opts) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.opendirPromise(p, opts);
        }, async (zipFs, { subPath }) => {
            return await zipFs.opendirPromise(subPath, opts);
        }, {
            requireSubpath: false,
        });
    }
    opendirSync(p, opts) {
        return this.makeCallSync(p, () => {
            return this.baseFs.opendirSync(p, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.opendirSync(subPath, opts);
        }, {
            requireSubpath: false,
        });
    }
    async readPromise(fd, buffer, offset, length, position) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return await this.baseFs.readPromise(fd, buffer, offset, length, position);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`read`);
        const [zipFs, realFd] = entry;
        return await zipFs.readPromise(realFd, buffer, offset, length, position);
    }
    readSync(fd, buffer, offset, length, position) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.readSync(fd, buffer, offset, length, position);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`readSync`);
        const [zipFs, realFd] = entry;
        return zipFs.readSync(realFd, buffer, offset, length, position);
    }
    async writePromise(fd, buffer, offset, length, position) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC) {
            if (typeof buffer === `string`) {
                return await this.baseFs.writePromise(fd, buffer, offset);
            }
            else {
                return await this.baseFs.writePromise(fd, buffer, offset, length, position);
            }
        }
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`write`);
        const [zipFs, realFd] = entry;
        if (typeof buffer === `string`) {
            return await zipFs.writePromise(realFd, buffer, offset);
        }
        else {
            return await zipFs.writePromise(realFd, buffer, offset, length, position);
        }
    }
    writeSync(fd, buffer, offset, length, position) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC) {
            if (typeof buffer === `string`) {
                return this.baseFs.writeSync(fd, buffer, offset);
            }
            else {
                return this.baseFs.writeSync(fd, buffer, offset, length, position);
            }
        }
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`writeSync`);
        const [zipFs, realFd] = entry;
        if (typeof buffer === `string`) {
            return zipFs.writeSync(realFd, buffer, offset);
        }
        else {
            return zipFs.writeSync(realFd, buffer, offset, length, position);
        }
    }
    async closePromise(fd) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return await this.baseFs.closePromise(fd);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`close`);
        this.fdMap.delete(fd);
        const [zipFs, realFd] = entry;
        return await zipFs.closePromise(realFd);
    }
    closeSync(fd) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.closeSync(fd);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`closeSync`);
        this.fdMap.delete(fd);
        const [zipFs, realFd] = entry;
        return zipFs.closeSync(realFd);
    }
    createReadStream(p, opts) {
        if (p === null)
            return this.baseFs.createReadStream(p, opts);
        return this.makeCallSync(p, () => {
            return this.baseFs.createReadStream(p, opts);
        }, (zipFs, { archivePath, subPath }) => {
            const stream = zipFs.createReadStream(subPath, opts);
            // This is a very hacky workaround. `ZipOpenFS` shouldn't have to work with `NativePath`s.
            // Ref: https://github.com/yarnpkg/berry/pull/3774
            // TODO: think of a better solution
            stream.path = path_1.npath.fromPortablePath(this.pathUtils.join(archivePath, subPath));
            return stream;
        });
    }
    createWriteStream(p, opts) {
        if (p === null)
            return this.baseFs.createWriteStream(p, opts);
        return this.makeCallSync(p, () => {
            return this.baseFs.createWriteStream(p, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.createWriteStream(subPath, opts);
        });
    }
    async realpathPromise(p) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.realpathPromise(p);
        }, async (zipFs, { archivePath, subPath }) => {
            let realArchivePath = this.realPaths.get(archivePath);
            if (typeof realArchivePath === `undefined`) {
                realArchivePath = await this.baseFs.realpathPromise(archivePath);
                this.realPaths.set(archivePath, realArchivePath);
            }
            return this.pathUtils.join(realArchivePath, this.pathUtils.relative(path_1.PortablePath.root, await zipFs.realpathPromise(subPath)));
        });
    }
    realpathSync(p) {
        return this.makeCallSync(p, () => {
            return this.baseFs.realpathSync(p);
        }, (zipFs, { archivePath, subPath }) => {
            let realArchivePath = this.realPaths.get(archivePath);
            if (typeof realArchivePath === `undefined`) {
                realArchivePath = this.baseFs.realpathSync(archivePath);
                this.realPaths.set(archivePath, realArchivePath);
            }
            return this.pathUtils.join(realArchivePath, this.pathUtils.relative(path_1.PortablePath.root, zipFs.realpathSync(subPath)));
        });
    }
    async existsPromise(p) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.existsPromise(p);
        }, async (zipFs, { subPath }) => {
            return await zipFs.existsPromise(subPath);
        });
    }
    existsSync(p) {
        return this.makeCallSync(p, () => {
            return this.baseFs.existsSync(p);
        }, (zipFs, { subPath }) => {
            return zipFs.existsSync(subPath);
        });
    }
    async accessPromise(p, mode) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.accessPromise(p, mode);
        }, async (zipFs, { subPath }) => {
            return await zipFs.accessPromise(subPath, mode);
        });
    }
    accessSync(p, mode) {
        return this.makeCallSync(p, () => {
            return this.baseFs.accessSync(p, mode);
        }, (zipFs, { subPath }) => {
            return zipFs.accessSync(subPath, mode);
        });
    }
    async statPromise(p, opts) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.statPromise(p, opts);
        }, async (zipFs, { subPath }) => {
            return await zipFs.statPromise(subPath, opts);
        });
    }
    statSync(p, opts) {
        return this.makeCallSync(p, () => {
            return this.baseFs.statSync(p, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.statSync(subPath, opts);
        });
    }
    async fstatPromise(fd, opts) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.fstatPromise(fd, opts);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`fstat`);
        const [zipFs, realFd] = entry;
        return zipFs.fstatPromise(realFd, opts);
    }
    fstatSync(fd, opts) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.fstatSync(fd, opts);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`fstatSync`);
        const [zipFs, realFd] = entry;
        return zipFs.fstatSync(realFd, opts);
    }
    async lstatPromise(p, opts) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.lstatPromise(p, opts);
        }, async (zipFs, { subPath }) => {
            return await zipFs.lstatPromise(subPath, opts);
        });
    }
    lstatSync(p, opts) {
        return this.makeCallSync(p, () => {
            return this.baseFs.lstatSync(p, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.lstatSync(subPath, opts);
        });
    }
    async fchmodPromise(fd, mask) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.fchmodPromise(fd, mask);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`fchmod`);
        const [zipFs, realFd] = entry;
        return zipFs.fchmodPromise(realFd, mask);
    }
    fchmodSync(fd, mask) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.fchmodSync(fd, mask);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`fchmodSync`);
        const [zipFs, realFd] = entry;
        return zipFs.fchmodSync(realFd, mask);
    }
    async chmodPromise(p, mask) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.chmodPromise(p, mask);
        }, async (zipFs, { subPath }) => {
            return await zipFs.chmodPromise(subPath, mask);
        });
    }
    chmodSync(p, mask) {
        return this.makeCallSync(p, () => {
            return this.baseFs.chmodSync(p, mask);
        }, (zipFs, { subPath }) => {
            return zipFs.chmodSync(subPath, mask);
        });
    }
    async fchownPromise(fd, uid, gid) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.fchownPromise(fd, uid, gid);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`fchown`);
        const [zipFs, realFd] = entry;
        return zipFs.fchownPromise(realFd, uid, gid);
    }
    fchownSync(fd, uid, gid) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.fchownSync(fd, uid, gid);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`fchownSync`);
        const [zipFs, realFd] = entry;
        return zipFs.fchownSync(realFd, uid, gid);
    }
    async chownPromise(p, uid, gid) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.chownPromise(p, uid, gid);
        }, async (zipFs, { subPath }) => {
            return await zipFs.chownPromise(subPath, uid, gid);
        });
    }
    chownSync(p, uid, gid) {
        return this.makeCallSync(p, () => {
            return this.baseFs.chownSync(p, uid, gid);
        }, (zipFs, { subPath }) => {
            return zipFs.chownSync(subPath, uid, gid);
        });
    }
    async renamePromise(oldP, newP) {
        return await this.makeCallPromise(oldP, async () => {
            return await this.makeCallPromise(newP, async () => {
                return await this.baseFs.renamePromise(oldP, newP);
            }, async () => {
                throw Object.assign(new Error(`EEXDEV: cross-device link not permitted`), { code: `EEXDEV` });
            });
        }, async (zipFsO, { subPath: subPathO }) => {
            return await this.makeCallPromise(newP, async () => {
                throw Object.assign(new Error(`EEXDEV: cross-device link not permitted`), { code: `EEXDEV` });
            }, async (zipFsN, { subPath: subPathN }) => {
                if (zipFsO !== zipFsN) {
                    throw Object.assign(new Error(`EEXDEV: cross-device link not permitted`), { code: `EEXDEV` });
                }
                else {
                    return await zipFsO.renamePromise(subPathO, subPathN);
                }
            });
        });
    }
    renameSync(oldP, newP) {
        return this.makeCallSync(oldP, () => {
            return this.makeCallSync(newP, () => {
                return this.baseFs.renameSync(oldP, newP);
            }, () => {
                throw Object.assign(new Error(`EEXDEV: cross-device link not permitted`), { code: `EEXDEV` });
            });
        }, (zipFsO, { subPath: subPathO }) => {
            return this.makeCallSync(newP, () => {
                throw Object.assign(new Error(`EEXDEV: cross-device link not permitted`), { code: `EEXDEV` });
            }, (zipFsN, { subPath: subPathN }) => {
                if (zipFsO !== zipFsN) {
                    throw Object.assign(new Error(`EEXDEV: cross-device link not permitted`), { code: `EEXDEV` });
                }
                else {
                    return zipFsO.renameSync(subPathO, subPathN);
                }
            });
        });
    }
    async copyFilePromise(sourceP, destP, flags = 0) {
        const fallback = async (sourceFs, sourceP, destFs, destP) => {
            if ((flags & fs_1.constants.COPYFILE_FICLONE_FORCE) !== 0)
                throw Object.assign(new Error(`EXDEV: cross-device clone not permitted, copyfile '${sourceP}' -> ${destP}'`), { code: `EXDEV` });
            if ((flags & fs_1.constants.COPYFILE_EXCL) && await this.existsPromise(sourceP))
                throw Object.assign(new Error(`EEXIST: file already exists, copyfile '${sourceP}' -> '${destP}'`), { code: `EEXIST` });
            let content;
            try {
                content = await sourceFs.readFilePromise(sourceP);
            }
            catch (error) {
                throw Object.assign(new Error(`EINVAL: invalid argument, copyfile '${sourceP}' -> '${destP}'`), { code: `EINVAL` });
            }
            await destFs.writeFilePromise(destP, content);
        };
        return await this.makeCallPromise(sourceP, async () => {
            return await this.makeCallPromise(destP, async () => {
                return await this.baseFs.copyFilePromise(sourceP, destP, flags);
            }, async (zipFsD, { subPath: subPathD }) => {
                return await fallback(this.baseFs, sourceP, zipFsD, subPathD);
            });
        }, async (zipFsS, { subPath: subPathS }) => {
            return await this.makeCallPromise(destP, async () => {
                return await fallback(zipFsS, subPathS, this.baseFs, destP);
            }, async (zipFsD, { subPath: subPathD }) => {
                if (zipFsS !== zipFsD) {
                    return await fallback(zipFsS, subPathS, zipFsD, subPathD);
                }
                else {
                    return await zipFsS.copyFilePromise(subPathS, subPathD, flags);
                }
            });
        });
    }
    copyFileSync(sourceP, destP, flags = 0) {
        const fallback = (sourceFs, sourceP, destFs, destP) => {
            if ((flags & fs_1.constants.COPYFILE_FICLONE_FORCE) !== 0)
                throw Object.assign(new Error(`EXDEV: cross-device clone not permitted, copyfile '${sourceP}' -> ${destP}'`), { code: `EXDEV` });
            if ((flags & fs_1.constants.COPYFILE_EXCL) && this.existsSync(sourceP))
                throw Object.assign(new Error(`EEXIST: file already exists, copyfile '${sourceP}' -> '${destP}'`), { code: `EEXIST` });
            let content;
            try {
                content = sourceFs.readFileSync(sourceP);
            }
            catch (error) {
                throw Object.assign(new Error(`EINVAL: invalid argument, copyfile '${sourceP}' -> '${destP}'`), { code: `EINVAL` });
            }
            destFs.writeFileSync(destP, content);
        };
        return this.makeCallSync(sourceP, () => {
            return this.makeCallSync(destP, () => {
                return this.baseFs.copyFileSync(sourceP, destP, flags);
            }, (zipFsD, { subPath: subPathD }) => {
                return fallback(this.baseFs, sourceP, zipFsD, subPathD);
            });
        }, (zipFsS, { subPath: subPathS }) => {
            return this.makeCallSync(destP, () => {
                return fallback(zipFsS, subPathS, this.baseFs, destP);
            }, (zipFsD, { subPath: subPathD }) => {
                if (zipFsS !== zipFsD) {
                    return fallback(zipFsS, subPathS, zipFsD, subPathD);
                }
                else {
                    return zipFsS.copyFileSync(subPathS, subPathD, flags);
                }
            });
        });
    }
    async appendFilePromise(p, content, opts) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.appendFilePromise(p, content, opts);
        }, async (zipFs, { subPath }) => {
            return await zipFs.appendFilePromise(subPath, content, opts);
        });
    }
    appendFileSync(p, content, opts) {
        return this.makeCallSync(p, () => {
            return this.baseFs.appendFileSync(p, content, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.appendFileSync(subPath, content, opts);
        });
    }
    async writeFilePromise(p, content, opts) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.writeFilePromise(p, content, opts);
        }, async (zipFs, { subPath }) => {
            return await zipFs.writeFilePromise(subPath, content, opts);
        });
    }
    writeFileSync(p, content, opts) {
        return this.makeCallSync(p, () => {
            return this.baseFs.writeFileSync(p, content, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.writeFileSync(subPath, content, opts);
        });
    }
    async unlinkPromise(p) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.unlinkPromise(p);
        }, async (zipFs, { subPath }) => {
            return await zipFs.unlinkPromise(subPath);
        });
    }
    unlinkSync(p) {
        return this.makeCallSync(p, () => {
            return this.baseFs.unlinkSync(p);
        }, (zipFs, { subPath }) => {
            return zipFs.unlinkSync(subPath);
        });
    }
    async utimesPromise(p, atime, mtime) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.utimesPromise(p, atime, mtime);
        }, async (zipFs, { subPath }) => {
            return await zipFs.utimesPromise(subPath, atime, mtime);
        });
    }
    utimesSync(p, atime, mtime) {
        return this.makeCallSync(p, () => {
            return this.baseFs.utimesSync(p, atime, mtime);
        }, (zipFs, { subPath }) => {
            return zipFs.utimesSync(subPath, atime, mtime);
        });
    }
    async mkdirPromise(p, opts) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.mkdirPromise(p, opts);
        }, async (zipFs, { subPath }) => {
            return await zipFs.mkdirPromise(subPath, opts);
        });
    }
    mkdirSync(p, opts) {
        return this.makeCallSync(p, () => {
            return this.baseFs.mkdirSync(p, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.mkdirSync(subPath, opts);
        });
    }
    async rmdirPromise(p, opts) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.rmdirPromise(p, opts);
        }, async (zipFs, { subPath }) => {
            return await zipFs.rmdirPromise(subPath, opts);
        });
    }
    rmdirSync(p, opts) {
        return this.makeCallSync(p, () => {
            return this.baseFs.rmdirSync(p, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.rmdirSync(subPath, opts);
        });
    }
    async linkPromise(existingP, newP) {
        return await this.makeCallPromise(newP, async () => {
            return await this.baseFs.linkPromise(existingP, newP);
        }, async (zipFs, { subPath }) => {
            return await zipFs.linkPromise(existingP, subPath);
        });
    }
    linkSync(existingP, newP) {
        return this.makeCallSync(newP, () => {
            return this.baseFs.linkSync(existingP, newP);
        }, (zipFs, { subPath }) => {
            return zipFs.linkSync(existingP, subPath);
        });
    }
    async symlinkPromise(target, p, type) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.symlinkPromise(target, p, type);
        }, async (zipFs, { subPath }) => {
            return await zipFs.symlinkPromise(target, subPath);
        });
    }
    symlinkSync(target, p, type) {
        return this.makeCallSync(p, () => {
            return this.baseFs.symlinkSync(target, p, type);
        }, (zipFs, { subPath }) => {
            return zipFs.symlinkSync(target, subPath);
        });
    }
    async readFilePromise(p, encoding) {
        return this.makeCallPromise(p, async () => {
            // This weird switch is required to tell TypeScript that the signatures are proper (otherwise it thinks that only the generic one is covered)
            switch (encoding) {
                case `utf8`:
                    return await this.baseFs.readFilePromise(p, encoding);
                default:
                    return await this.baseFs.readFilePromise(p, encoding);
            }
        }, async (zipFs, { subPath }) => {
            return await zipFs.readFilePromise(subPath, encoding);
        });
    }
    readFileSync(p, encoding) {
        return this.makeCallSync(p, () => {
            // This weird switch is required to tell TypeScript that the signatures are proper (otherwise it thinks that only the generic one is covered)
            switch (encoding) {
                case `utf8`:
                    return this.baseFs.readFileSync(p, encoding);
                default:
                    return this.baseFs.readFileSync(p, encoding);
            }
        }, (zipFs, { subPath }) => {
            return zipFs.readFileSync(subPath, encoding);
        });
    }
    async readdirPromise(p, opts) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.readdirPromise(p, opts);
        }, async (zipFs, { subPath }) => {
            return await zipFs.readdirPromise(subPath, opts);
        }, {
            requireSubpath: false,
        });
    }
    readdirSync(p, opts) {
        return this.makeCallSync(p, () => {
            return this.baseFs.readdirSync(p, opts);
        }, (zipFs, { subPath }) => {
            return zipFs.readdirSync(subPath, opts);
        }, {
            requireSubpath: false,
        });
    }
    async readlinkPromise(p) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.readlinkPromise(p);
        }, async (zipFs, { subPath }) => {
            return await zipFs.readlinkPromise(subPath);
        });
    }
    readlinkSync(p) {
        return this.makeCallSync(p, () => {
            return this.baseFs.readlinkSync(p);
        }, (zipFs, { subPath }) => {
            return zipFs.readlinkSync(subPath);
        });
    }
    async truncatePromise(p, len) {
        return await this.makeCallPromise(p, async () => {
            return await this.baseFs.truncatePromise(p, len);
        }, async (zipFs, { subPath }) => {
            return await zipFs.truncatePromise(subPath, len);
        });
    }
    truncateSync(p, len) {
        return this.makeCallSync(p, () => {
            return this.baseFs.truncateSync(p, len);
        }, (zipFs, { subPath }) => {
            return zipFs.truncateSync(subPath, len);
        });
    }
    async ftruncatePromise(fd, len) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.ftruncatePromise(fd, len);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`ftruncate`);
        const [zipFs, realFd] = entry;
        return zipFs.ftruncatePromise(realFd, len);
    }
    ftruncateSync(fd, len) {
        if ((fd & ZIP_MASK) !== ZIP_MAGIC)
            return this.baseFs.ftruncateSync(fd, len);
        const entry = this.fdMap.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`ftruncateSync`);
        const [zipFs, realFd] = entry;
        return zipFs.ftruncateSync(realFd, len);
    }
    watch(p, a, b) {
        return this.makeCallSync(p, () => {
            return this.baseFs.watch(p, 
            // @ts-expect-error
            a, b);
        }, (zipFs, { subPath }) => {
            return zipFs.watch(subPath, 
            // @ts-expect-error
            a, b);
        });
    }
    watchFile(p, a, b) {
        return this.makeCallSync(p, () => {
            return this.baseFs.watchFile(p, 
            // @ts-expect-error
            a, b);
        }, () => {
            return (0, watchFile_1.watchFile)(this, p, a, b);
        });
    }
    unwatchFile(p, cb) {
        return this.makeCallSync(p, () => {
            return this.baseFs.unwatchFile(p, cb);
        }, () => {
            return (0, watchFile_1.unwatchFile)(this, p, cb);
        });
    }
    async makeCallPromise(p, discard, accept, { requireSubpath = true } = {}) {
        if (typeof p !== `string`)
            return await discard();
        const normalizedP = this.resolve(p);
        const zipInfo = this.findZip(normalizedP);
        if (!zipInfo)
            return await discard();
        if (requireSubpath && zipInfo.subPath === `/`)
            return await discard();
        return await this.getZipPromise(zipInfo.archivePath, async (zipFs) => await accept(zipFs, zipInfo));
    }
    makeCallSync(p, discard, accept, { requireSubpath = true } = {}) {
        if (typeof p !== `string`)
            return discard();
        const normalizedP = this.resolve(p);
        const zipInfo = this.findZip(normalizedP);
        if (!zipInfo)
            return discard();
        if (requireSubpath && zipInfo.subPath === `/`)
            return discard();
        return this.getZipSync(zipInfo.archivePath, zipFs => accept(zipFs, zipInfo));
    }
    findZip(p) {
        if (this.filter && !this.filter.test(p))
            return null;
        let filePath = ``;
        while (true) {
            const pathPartWithArchive = p.substring(filePath.length);
            let archivePart;
            if (!this.fileExtensions) {
                archivePart = (0, exports.getArchivePart)(pathPartWithArchive, `.zip`);
            }
            else {
                for (const ext of this.fileExtensions) {
                    archivePart = (0, exports.getArchivePart)(pathPartWithArchive, ext);
                    if (archivePart) {
                        break;
                    }
                }
            }
            if (!archivePart)
                return null;
            filePath = this.pathUtils.join(filePath, archivePart);
            if (this.isZip.has(filePath) === false) {
                if (this.notZip.has(filePath))
                    continue;
                try {
                    if (!this.baseFs.lstatSync(filePath).isFile()) {
                        this.notZip.add(filePath);
                        continue;
                    }
                }
                catch {
                    return null;
                }
                this.isZip.add(filePath);
            }
            return {
                archivePath: filePath,
                subPath: this.pathUtils.join(path_1.PortablePath.root, p.substring(filePath.length)),
            };
        }
    }
    limitOpenFiles(max) {
        if (this.zipInstances === null)
            return;
        const now = Date.now();
        let nextExpiresAt = now + this.maxAge;
        let closeCount = max === null ? 0 : this.zipInstances.size - max;
        for (const [path, { zipFs, expiresAt, refCount }] of this.zipInstances.entries()) {
            if (refCount !== 0 || zipFs.hasOpenFileHandles()) {
                continue;
            }
            else if (now >= expiresAt) {
                zipFs.saveAndClose();
                this.zipInstances.delete(path);
                closeCount -= 1;
                continue;
            }
            else if (max === null || closeCount <= 0) {
                nextExpiresAt = expiresAt;
                break;
            }
            zipFs.saveAndClose();
            this.zipInstances.delete(path);
            closeCount -= 1;
        }
        if (this.limitOpenFilesTimeout === null && ((max === null && this.zipInstances.size > 0) || max !== null)) {
            this.limitOpenFilesTimeout = setTimeout(() => {
                this.limitOpenFilesTimeout = null;
                this.limitOpenFiles(null);
            }, nextExpiresAt - now).unref();
        }
    }
    async getZipPromise(p, accept) {
        const getZipOptions = async () => ({
            baseFs: this.baseFs,
            libzip: this.libzip,
            readOnly: this.readOnlyArchives,
            stats: await this.baseFs.statPromise(p),
        });
        if (this.zipInstances) {
            let cachedZipFs = this.zipInstances.get(p);
            if (!cachedZipFs) {
                const zipOptions = await getZipOptions();
                // We need to recheck because concurrent getZipPromise calls may
                // have instantiated the zip archive while we were waiting
                cachedZipFs = this.zipInstances.get(p);
                if (!cachedZipFs) {
                    cachedZipFs = {
                        zipFs: new ZipFS_1.ZipFS(p, zipOptions),
                        expiresAt: 0,
                        refCount: 0,
                    };
                }
            }
            // Removing then re-adding the field allows us to easily implement
            // a basic LRU garbage collection strategy
            this.zipInstances.delete(p);
            this.limitOpenFiles(this.maxOpenFiles - 1);
            this.zipInstances.set(p, cachedZipFs);
            cachedZipFs.expiresAt = Date.now() + this.maxAge;
            cachedZipFs.refCount += 1;
            try {
                return await accept(cachedZipFs.zipFs);
            }
            finally {
                cachedZipFs.refCount -= 1;
            }
        }
        else {
            const zipFs = new ZipFS_1.ZipFS(p, await getZipOptions());
            try {
                return await accept(zipFs);
            }
            finally {
                zipFs.saveAndClose();
            }
        }
    }
    getZipSync(p, accept) {
        const getZipOptions = () => ({
            baseFs: this.baseFs,
            libzip: this.libzip,
            readOnly: this.readOnlyArchives,
            stats: this.baseFs.statSync(p),
        });
        if (this.zipInstances) {
            let cachedZipFs = this.zipInstances.get(p);
            if (!cachedZipFs) {
                cachedZipFs = {
                    zipFs: new ZipFS_1.ZipFS(p, getZipOptions()),
                    expiresAt: 0,
                    refCount: 0,
                };
            }
            // Removing then re-adding the field allows us to easily implement
            // a basic LRU garbage collection strategy
            this.zipInstances.delete(p);
            this.limitOpenFiles(this.maxOpenFiles - 1);
            this.zipInstances.set(p, cachedZipFs);
            cachedZipFs.expiresAt = Date.now() + this.maxAge;
            return accept(cachedZipFs.zipFs);
        }
        else {
            const zipFs = new ZipFS_1.ZipFS(p, getZipOptions());
            try {
                return accept(zipFs);
            }
            finally {
                zipFs.saveAndClose();
            }
        }
    }
}
exports.ZipOpenFS = ZipOpenFS;
