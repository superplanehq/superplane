"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.NodeFS = void 0;
const tslib_1 = require("tslib");
const fs_1 = tslib_1.__importDefault(require("fs"));
const FakeFS_1 = require("./FakeFS");
const errors_1 = require("./errors");
const path_1 = require("./path");
class NodeFS extends FakeFS_1.BasePortableFakeFS {
    constructor(realFs = fs_1.default) {
        super();
        this.realFs = realFs;
        // @ts-expect-error
        if (typeof this.realFs.lutimes !== `undefined`) {
            this.lutimesPromise = this.lutimesPromiseImpl;
            this.lutimesSync = this.lutimesSyncImpl;
        }
    }
    getExtractHint() {
        return false;
    }
    getRealPath() {
        return path_1.PortablePath.root;
    }
    resolve(p) {
        return path_1.ppath.resolve(p);
    }
    async openPromise(p, flags, mode) {
        return await new Promise((resolve, reject) => {
            this.realFs.open(path_1.npath.fromPortablePath(p), flags, mode, this.makeCallback(resolve, reject));
        });
    }
    openSync(p, flags, mode) {
        return this.realFs.openSync(path_1.npath.fromPortablePath(p), flags, mode);
    }
    async opendirPromise(p, opts) {
        return await new Promise((resolve, reject) => {
            if (typeof opts !== `undefined`) {
                this.realFs.opendir(path_1.npath.fromPortablePath(p), opts, this.makeCallback(resolve, reject));
            }
            else {
                this.realFs.opendir(path_1.npath.fromPortablePath(p), this.makeCallback(resolve, reject));
            }
        }).then(dir => {
            return Object.defineProperty(dir, `path`, { value: p, configurable: true, writable: true });
        });
    }
    opendirSync(p, opts) {
        const dir = typeof opts !== `undefined`
            ? this.realFs.opendirSync(path_1.npath.fromPortablePath(p), opts)
            : this.realFs.opendirSync(path_1.npath.fromPortablePath(p));
        return Object.defineProperty(dir, `path`, { value: p, configurable: true, writable: true });
    }
    async readPromise(fd, buffer, offset = 0, length = 0, position = -1) {
        return await new Promise((resolve, reject) => {
            this.realFs.read(fd, buffer, offset, length, position, (error, bytesRead) => {
                if (error) {
                    reject(error);
                }
                else {
                    resolve(bytesRead);
                }
            });
        });
    }
    readSync(fd, buffer, offset, length, position) {
        return this.realFs.readSync(fd, buffer, offset, length, position);
    }
    async writePromise(fd, buffer, offset, length, position) {
        return await new Promise((resolve, reject) => {
            if (typeof buffer === `string`) {
                return this.realFs.write(fd, buffer, offset, this.makeCallback(resolve, reject));
            }
            else {
                return this.realFs.write(fd, buffer, offset, length, position, this.makeCallback(resolve, reject));
            }
        });
    }
    writeSync(fd, buffer, offset, length, position) {
        if (typeof buffer === `string`) {
            return this.realFs.writeSync(fd, buffer, offset);
        }
        else {
            return this.realFs.writeSync(fd, buffer, offset, length, position);
        }
    }
    async closePromise(fd) {
        await new Promise((resolve, reject) => {
            this.realFs.close(fd, this.makeCallback(resolve, reject));
        });
    }
    closeSync(fd) {
        this.realFs.closeSync(fd);
    }
    createReadStream(p, opts) {
        const realPath = (p !== null ? path_1.npath.fromPortablePath(p) : p);
        return this.realFs.createReadStream(realPath, opts);
    }
    createWriteStream(p, opts) {
        const realPath = (p !== null ? path_1.npath.fromPortablePath(p) : p);
        return this.realFs.createWriteStream(realPath, opts);
    }
    async realpathPromise(p) {
        return await new Promise((resolve, reject) => {
            this.realFs.realpath(path_1.npath.fromPortablePath(p), {}, this.makeCallback(resolve, reject));
        }).then(path => {
            return path_1.npath.toPortablePath(path);
        });
    }
    realpathSync(p) {
        return path_1.npath.toPortablePath(this.realFs.realpathSync(path_1.npath.fromPortablePath(p), {}));
    }
    async existsPromise(p) {
        return await new Promise(resolve => {
            this.realFs.exists(path_1.npath.fromPortablePath(p), resolve);
        });
    }
    accessSync(p, mode) {
        return this.realFs.accessSync(path_1.npath.fromPortablePath(p), mode);
    }
    async accessPromise(p, mode) {
        return await new Promise((resolve, reject) => {
            this.realFs.access(path_1.npath.fromPortablePath(p), mode, this.makeCallback(resolve, reject));
        });
    }
    existsSync(p) {
        return this.realFs.existsSync(path_1.npath.fromPortablePath(p));
    }
    async statPromise(p, opts) {
        return await new Promise((resolve, reject) => {
            if (opts) {
                // @ts-expect-error The node types are out of date
                this.realFs.stat(path_1.npath.fromPortablePath(p), opts, this.makeCallback(resolve, reject));
            }
            else {
                this.realFs.stat(path_1.npath.fromPortablePath(p), this.makeCallback(resolve, reject));
            }
        });
    }
    statSync(p, opts) {
        if (opts) {
            // @ts-expect-error The node types are out of date
            return this.realFs.statSync(path_1.npath.fromPortablePath(p), opts);
        }
        else {
            return this.realFs.statSync(path_1.npath.fromPortablePath(p));
        }
    }
    async fstatPromise(fd, opts) {
        return await new Promise((resolve, reject) => {
            if (opts) {
                // @ts-expect-error - The node typings doesn't know about the options
                this.realFs.fstat(fd, opts, this.makeCallback(resolve, reject));
            }
            else {
                this.realFs.fstat(fd, this.makeCallback(resolve, reject));
            }
        });
    }
    fstatSync(fd, opts) {
        if (opts) {
            // @ts-expect-error - The node typings doesn't know about the options
            return this.realFs.fstatSync(fd, opts);
        }
        else {
            return this.realFs.fstatSync(fd);
        }
    }
    async lstatPromise(p, opts) {
        return await new Promise((resolve, reject) => {
            if (opts) {
                // @ts-expect-error - TS does not know this takes options
                this.realFs.lstat(path_1.npath.fromPortablePath(p), opts, this.makeCallback(resolve, reject));
            }
            else {
                this.realFs.lstat(path_1.npath.fromPortablePath(p), this.makeCallback(resolve, reject));
            }
        });
    }
    lstatSync(p, opts) {
        if (opts) {
            // @ts-expect-error - TS does not know this takes options
            return this.realFs.lstatSync(path_1.npath.fromPortablePath(p), opts);
        }
        else {
            return this.realFs.lstatSync(path_1.npath.fromPortablePath(p));
        }
    }
    async fchmodPromise(fd, mask) {
        return await new Promise((resolve, reject) => {
            this.realFs.fchmod(fd, mask, this.makeCallback(resolve, reject));
        });
    }
    fchmodSync(fd, mask) {
        return this.realFs.fchmodSync(fd, mask);
    }
    async chmodPromise(p, mask) {
        return await new Promise((resolve, reject) => {
            this.realFs.chmod(path_1.npath.fromPortablePath(p), mask, this.makeCallback(resolve, reject));
        });
    }
    chmodSync(p, mask) {
        return this.realFs.chmodSync(path_1.npath.fromPortablePath(p), mask);
    }
    async fchownPromise(fd, uid, gid) {
        return await new Promise((resolve, reject) => {
            this.realFs.fchown(fd, uid, gid, this.makeCallback(resolve, reject));
        });
    }
    fchownSync(fd, uid, gid) {
        return this.realFs.fchownSync(fd, uid, gid);
    }
    async chownPromise(p, uid, gid) {
        return await new Promise((resolve, reject) => {
            this.realFs.chown(path_1.npath.fromPortablePath(p), uid, gid, this.makeCallback(resolve, reject));
        });
    }
    chownSync(p, uid, gid) {
        return this.realFs.chownSync(path_1.npath.fromPortablePath(p), uid, gid);
    }
    async renamePromise(oldP, newP) {
        return await new Promise((resolve, reject) => {
            this.realFs.rename(path_1.npath.fromPortablePath(oldP), path_1.npath.fromPortablePath(newP), this.makeCallback(resolve, reject));
        });
    }
    renameSync(oldP, newP) {
        return this.realFs.renameSync(path_1.npath.fromPortablePath(oldP), path_1.npath.fromPortablePath(newP));
    }
    async copyFilePromise(sourceP, destP, flags = 0) {
        return await new Promise((resolve, reject) => {
            this.realFs.copyFile(path_1.npath.fromPortablePath(sourceP), path_1.npath.fromPortablePath(destP), flags, this.makeCallback(resolve, reject));
        });
    }
    copyFileSync(sourceP, destP, flags = 0) {
        return this.realFs.copyFileSync(path_1.npath.fromPortablePath(sourceP), path_1.npath.fromPortablePath(destP), flags);
    }
    async appendFilePromise(p, content, opts) {
        return await new Promise((resolve, reject) => {
            const fsNativePath = typeof p === `string` ? path_1.npath.fromPortablePath(p) : p;
            if (opts) {
                this.realFs.appendFile(fsNativePath, content, opts, this.makeCallback(resolve, reject));
            }
            else {
                this.realFs.appendFile(fsNativePath, content, this.makeCallback(resolve, reject));
            }
        });
    }
    appendFileSync(p, content, opts) {
        const fsNativePath = typeof p === `string` ? path_1.npath.fromPortablePath(p) : p;
        if (opts) {
            this.realFs.appendFileSync(fsNativePath, content, opts);
        }
        else {
            this.realFs.appendFileSync(fsNativePath, content);
        }
    }
    async writeFilePromise(p, content, opts) {
        return await new Promise((resolve, reject) => {
            const fsNativePath = typeof p === `string` ? path_1.npath.fromPortablePath(p) : p;
            if (opts) {
                this.realFs.writeFile(fsNativePath, content, opts, this.makeCallback(resolve, reject));
            }
            else {
                this.realFs.writeFile(fsNativePath, content, this.makeCallback(resolve, reject));
            }
        });
    }
    writeFileSync(p, content, opts) {
        const fsNativePath = typeof p === `string` ? path_1.npath.fromPortablePath(p) : p;
        if (opts) {
            this.realFs.writeFileSync(fsNativePath, content, opts);
        }
        else {
            this.realFs.writeFileSync(fsNativePath, content);
        }
    }
    async unlinkPromise(p) {
        return await new Promise((resolve, reject) => {
            this.realFs.unlink(path_1.npath.fromPortablePath(p), this.makeCallback(resolve, reject));
        });
    }
    unlinkSync(p) {
        return this.realFs.unlinkSync(path_1.npath.fromPortablePath(p));
    }
    async utimesPromise(p, atime, mtime) {
        return await new Promise((resolve, reject) => {
            this.realFs.utimes(path_1.npath.fromPortablePath(p), atime, mtime, this.makeCallback(resolve, reject));
        });
    }
    utimesSync(p, atime, mtime) {
        this.realFs.utimesSync(path_1.npath.fromPortablePath(p), atime, mtime);
    }
    async lutimesPromiseImpl(p, atime, mtime) {
        // @ts-expect-error: Not yet in DefinitelyTyped
        const lutimes = this.realFs.lutimes;
        if (typeof lutimes === `undefined`)
            throw (0, errors_1.ENOSYS)(`unavailable Node binding`, `lutimes '${p}'`);
        return await new Promise((resolve, reject) => {
            lutimes.call(this.realFs, path_1.npath.fromPortablePath(p), atime, mtime, this.makeCallback(resolve, reject));
        });
    }
    lutimesSyncImpl(p, atime, mtime) {
        // @ts-expect-error: Not yet in DefinitelyTyped
        const lutimesSync = this.realFs.lutimesSync;
        if (typeof lutimesSync === `undefined`)
            throw (0, errors_1.ENOSYS)(`unavailable Node binding`, `lutimes '${p}'`);
        lutimesSync.call(this.realFs, path_1.npath.fromPortablePath(p), atime, mtime);
    }
    async mkdirPromise(p, opts) {
        return await new Promise((resolve, reject) => {
            // @ts-expect-error - Types are outdated, the second argument in the callback is either a string or undefined
            this.realFs.mkdir(path_1.npath.fromPortablePath(p), opts, this.makeCallback(resolve, reject));
        });
    }
    mkdirSync(p, opts) {
        // @ts-expect-error - Types are outdated, returns either a string or undefined
        return this.realFs.mkdirSync(path_1.npath.fromPortablePath(p), opts);
    }
    async rmdirPromise(p, opts) {
        return await new Promise((resolve, reject) => {
            // TODO: always pass opts when min node version is 12.10+
            if (opts) {
                this.realFs.rmdir(path_1.npath.fromPortablePath(p), opts, this.makeCallback(resolve, reject));
            }
            else {
                this.realFs.rmdir(path_1.npath.fromPortablePath(p), this.makeCallback(resolve, reject));
            }
        });
    }
    rmdirSync(p, opts) {
        return this.realFs.rmdirSync(path_1.npath.fromPortablePath(p), opts);
    }
    async linkPromise(existingP, newP) {
        return await new Promise((resolve, reject) => {
            this.realFs.link(path_1.npath.fromPortablePath(existingP), path_1.npath.fromPortablePath(newP), this.makeCallback(resolve, reject));
        });
    }
    linkSync(existingP, newP) {
        return this.realFs.linkSync(path_1.npath.fromPortablePath(existingP), path_1.npath.fromPortablePath(newP));
    }
    async symlinkPromise(target, p, type) {
        return await new Promise((resolve, reject) => {
            this.realFs.symlink(path_1.npath.fromPortablePath(target.replace(/\/+$/, ``)), path_1.npath.fromPortablePath(p), type, this.makeCallback(resolve, reject));
        });
    }
    symlinkSync(target, p, type) {
        return this.realFs.symlinkSync(path_1.npath.fromPortablePath(target.replace(/\/+$/, ``)), path_1.npath.fromPortablePath(p), type);
    }
    async readFilePromise(p, encoding) {
        return await new Promise((resolve, reject) => {
            const fsNativePath = typeof p === `string` ? path_1.npath.fromPortablePath(p) : p;
            this.realFs.readFile(fsNativePath, encoding, this.makeCallback(resolve, reject));
        });
    }
    readFileSync(p, encoding) {
        const fsNativePath = typeof p === `string` ? path_1.npath.fromPortablePath(p) : p;
        return this.realFs.readFileSync(fsNativePath, encoding);
    }
    async readdirPromise(p, opts) {
        return await new Promise((resolve, reject) => {
            if (opts === null || opts === void 0 ? void 0 : opts.withFileTypes) {
                this.realFs.readdir(path_1.npath.fromPortablePath(p), { withFileTypes: true }, this.makeCallback(resolve, reject));
            }
            else {
                this.realFs.readdir(path_1.npath.fromPortablePath(p), this.makeCallback(value => resolve(value), reject));
            }
        });
    }
    readdirSync(p, opts) {
        if (opts === null || opts === void 0 ? void 0 : opts.withFileTypes) {
            return this.realFs.readdirSync(path_1.npath.fromPortablePath(p), { withFileTypes: true });
        }
        else {
            return this.realFs.readdirSync(path_1.npath.fromPortablePath(p));
        }
    }
    async readlinkPromise(p) {
        return await new Promise((resolve, reject) => {
            this.realFs.readlink(path_1.npath.fromPortablePath(p), this.makeCallback(resolve, reject));
        }).then(path => {
            return path_1.npath.toPortablePath(path);
        });
    }
    readlinkSync(p) {
        return path_1.npath.toPortablePath(this.realFs.readlinkSync(path_1.npath.fromPortablePath(p)));
    }
    async truncatePromise(p, len) {
        return await new Promise((resolve, reject) => {
            this.realFs.truncate(path_1.npath.fromPortablePath(p), len, this.makeCallback(resolve, reject));
        });
    }
    truncateSync(p, len) {
        return this.realFs.truncateSync(path_1.npath.fromPortablePath(p), len);
    }
    async ftruncatePromise(fd, len) {
        return await new Promise((resolve, reject) => {
            this.realFs.ftruncate(fd, len, this.makeCallback(resolve, reject));
        });
    }
    ftruncateSync(fd, len) {
        return this.realFs.ftruncateSync(fd, len);
    }
    watch(p, a, b) {
        return this.realFs.watch(path_1.npath.fromPortablePath(p), 
        // @ts-expect-error
        a, b);
    }
    watchFile(p, a, b) {
        return this.realFs.watchFile(path_1.npath.fromPortablePath(p), 
        // @ts-expect-error
        a, b);
    }
    unwatchFile(p, cb) {
        return this.realFs.unwatchFile(path_1.npath.fromPortablePath(p), cb);
    }
    makeCallback(resolve, reject) {
        return (err, result) => {
            if (err) {
                reject(err);
            }
            else {
                resolve(result);
            }
        };
    }
}
exports.NodeFS = NodeFS;
