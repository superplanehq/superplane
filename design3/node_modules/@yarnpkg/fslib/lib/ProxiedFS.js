"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProxiedFS = void 0;
const FakeFS_1 = require("./FakeFS");
class ProxiedFS extends FakeFS_1.FakeFS {
    getExtractHint(hints) {
        return this.baseFs.getExtractHint(hints);
    }
    resolve(path) {
        return this.mapFromBase(this.baseFs.resolve(this.mapToBase(path)));
    }
    getRealPath() {
        return this.mapFromBase(this.baseFs.getRealPath());
    }
    async openPromise(p, flags, mode) {
        return this.baseFs.openPromise(this.mapToBase(p), flags, mode);
    }
    openSync(p, flags, mode) {
        return this.baseFs.openSync(this.mapToBase(p), flags, mode);
    }
    async opendirPromise(p, opts) {
        return Object.assign(await this.baseFs.opendirPromise(this.mapToBase(p), opts), { path: p });
    }
    opendirSync(p, opts) {
        return Object.assign(this.baseFs.opendirSync(this.mapToBase(p), opts), { path: p });
    }
    async readPromise(fd, buffer, offset, length, position) {
        return await this.baseFs.readPromise(fd, buffer, offset, length, position);
    }
    readSync(fd, buffer, offset, length, position) {
        return this.baseFs.readSync(fd, buffer, offset, length, position);
    }
    async writePromise(fd, buffer, offset, length, position) {
        if (typeof buffer === `string`) {
            return await this.baseFs.writePromise(fd, buffer, offset);
        }
        else {
            return await this.baseFs.writePromise(fd, buffer, offset, length, position);
        }
    }
    writeSync(fd, buffer, offset, length, position) {
        if (typeof buffer === `string`) {
            return this.baseFs.writeSync(fd, buffer, offset);
        }
        else {
            return this.baseFs.writeSync(fd, buffer, offset, length, position);
        }
    }
    async closePromise(fd) {
        return this.baseFs.closePromise(fd);
    }
    closeSync(fd) {
        this.baseFs.closeSync(fd);
    }
    createReadStream(p, opts) {
        return this.baseFs.createReadStream(p !== null ? this.mapToBase(p) : p, opts);
    }
    createWriteStream(p, opts) {
        return this.baseFs.createWriteStream(p !== null ? this.mapToBase(p) : p, opts);
    }
    async realpathPromise(p) {
        return this.mapFromBase(await this.baseFs.realpathPromise(this.mapToBase(p)));
    }
    realpathSync(p) {
        return this.mapFromBase(this.baseFs.realpathSync(this.mapToBase(p)));
    }
    async existsPromise(p) {
        return this.baseFs.existsPromise(this.mapToBase(p));
    }
    existsSync(p) {
        return this.baseFs.existsSync(this.mapToBase(p));
    }
    accessSync(p, mode) {
        return this.baseFs.accessSync(this.mapToBase(p), mode);
    }
    async accessPromise(p, mode) {
        return this.baseFs.accessPromise(this.mapToBase(p), mode);
    }
    async statPromise(p, opts) {
        return this.baseFs.statPromise(this.mapToBase(p), opts);
    }
    statSync(p, opts) {
        return this.baseFs.statSync(this.mapToBase(p), opts);
    }
    async fstatPromise(fd, opts) {
        return this.baseFs.fstatPromise(fd, opts);
    }
    fstatSync(fd, opts) {
        return this.baseFs.fstatSync(fd, opts);
    }
    lstatPromise(p, opts) {
        return this.baseFs.lstatPromise(this.mapToBase(p), opts);
    }
    lstatSync(p, opts) {
        return this.baseFs.lstatSync(this.mapToBase(p), opts);
    }
    async fchmodPromise(fd, mask) {
        return this.baseFs.fchmodPromise(fd, mask);
    }
    fchmodSync(fd, mask) {
        return this.baseFs.fchmodSync(fd, mask);
    }
    async chmodPromise(p, mask) {
        return this.baseFs.chmodPromise(this.mapToBase(p), mask);
    }
    chmodSync(p, mask) {
        return this.baseFs.chmodSync(this.mapToBase(p), mask);
    }
    async fchownPromise(fd, uid, gid) {
        return this.baseFs.fchownPromise(fd, uid, gid);
    }
    fchownSync(fd, uid, gid) {
        return this.baseFs.fchownSync(fd, uid, gid);
    }
    async chownPromise(p, uid, gid) {
        return this.baseFs.chownPromise(this.mapToBase(p), uid, gid);
    }
    chownSync(p, uid, gid) {
        return this.baseFs.chownSync(this.mapToBase(p), uid, gid);
    }
    async renamePromise(oldP, newP) {
        return this.baseFs.renamePromise(this.mapToBase(oldP), this.mapToBase(newP));
    }
    renameSync(oldP, newP) {
        return this.baseFs.renameSync(this.mapToBase(oldP), this.mapToBase(newP));
    }
    async copyFilePromise(sourceP, destP, flags = 0) {
        return this.baseFs.copyFilePromise(this.mapToBase(sourceP), this.mapToBase(destP), flags);
    }
    copyFileSync(sourceP, destP, flags = 0) {
        return this.baseFs.copyFileSync(this.mapToBase(sourceP), this.mapToBase(destP), flags);
    }
    async appendFilePromise(p, content, opts) {
        return this.baseFs.appendFilePromise(this.fsMapToBase(p), content, opts);
    }
    appendFileSync(p, content, opts) {
        return this.baseFs.appendFileSync(this.fsMapToBase(p), content, opts);
    }
    async writeFilePromise(p, content, opts) {
        return this.baseFs.writeFilePromise(this.fsMapToBase(p), content, opts);
    }
    writeFileSync(p, content, opts) {
        return this.baseFs.writeFileSync(this.fsMapToBase(p), content, opts);
    }
    async unlinkPromise(p) {
        return this.baseFs.unlinkPromise(this.mapToBase(p));
    }
    unlinkSync(p) {
        return this.baseFs.unlinkSync(this.mapToBase(p));
    }
    async utimesPromise(p, atime, mtime) {
        return this.baseFs.utimesPromise(this.mapToBase(p), atime, mtime);
    }
    utimesSync(p, atime, mtime) {
        return this.baseFs.utimesSync(this.mapToBase(p), atime, mtime);
    }
    async mkdirPromise(p, opts) {
        return this.baseFs.mkdirPromise(this.mapToBase(p), opts);
    }
    mkdirSync(p, opts) {
        return this.baseFs.mkdirSync(this.mapToBase(p), opts);
    }
    async rmdirPromise(p, opts) {
        return this.baseFs.rmdirPromise(this.mapToBase(p), opts);
    }
    rmdirSync(p, opts) {
        return this.baseFs.rmdirSync(this.mapToBase(p), opts);
    }
    async linkPromise(existingP, newP) {
        return this.baseFs.linkPromise(this.mapToBase(existingP), this.mapToBase(newP));
    }
    linkSync(existingP, newP) {
        return this.baseFs.linkSync(this.mapToBase(existingP), this.mapToBase(newP));
    }
    async symlinkPromise(target, p, type) {
        const mappedP = this.mapToBase(p);
        if (this.pathUtils.isAbsolute(target))
            return this.baseFs.symlinkPromise(this.mapToBase(target), mappedP, type);
        const mappedAbsoluteTarget = this.mapToBase(this.pathUtils.join(this.pathUtils.dirname(p), target));
        const mappedTarget = this.baseFs.pathUtils.relative(this.baseFs.pathUtils.dirname(mappedP), mappedAbsoluteTarget);
        return this.baseFs.symlinkPromise(mappedTarget, mappedP, type);
    }
    symlinkSync(target, p, type) {
        const mappedP = this.mapToBase(p);
        if (this.pathUtils.isAbsolute(target))
            return this.baseFs.symlinkSync(this.mapToBase(target), mappedP, type);
        const mappedAbsoluteTarget = this.mapToBase(this.pathUtils.join(this.pathUtils.dirname(p), target));
        const mappedTarget = this.baseFs.pathUtils.relative(this.baseFs.pathUtils.dirname(mappedP), mappedAbsoluteTarget);
        return this.baseFs.symlinkSync(mappedTarget, mappedP, type);
    }
    async readFilePromise(p, encoding) {
        // This weird condition is required to tell TypeScript that the signatures are proper (otherwise it thinks that only the generic one is covered)
        if (encoding === `utf8`) {
            return this.baseFs.readFilePromise(this.fsMapToBase(p), encoding);
        }
        else {
            return this.baseFs.readFilePromise(this.fsMapToBase(p), encoding);
        }
    }
    readFileSync(p, encoding) {
        // This weird condition is required to tell TypeScript that the signatures are proper (otherwise it thinks that only the generic one is covered)
        if (encoding === `utf8`) {
            return this.baseFs.readFileSync(this.fsMapToBase(p), encoding);
        }
        else {
            return this.baseFs.readFileSync(this.fsMapToBase(p), encoding);
        }
    }
    async readdirPromise(p, opts) {
        return this.baseFs.readdirPromise(this.mapToBase(p), opts);
    }
    readdirSync(p, opts) {
        return this.baseFs.readdirSync(this.mapToBase(p), opts);
    }
    async readlinkPromise(p) {
        return this.mapFromBase(await this.baseFs.readlinkPromise(this.mapToBase(p)));
    }
    readlinkSync(p) {
        return this.mapFromBase(this.baseFs.readlinkSync(this.mapToBase(p)));
    }
    async truncatePromise(p, len) {
        return this.baseFs.truncatePromise(this.mapToBase(p), len);
    }
    truncateSync(p, len) {
        return this.baseFs.truncateSync(this.mapToBase(p), len);
    }
    async ftruncatePromise(fd, len) {
        return this.baseFs.ftruncatePromise(fd, len);
    }
    ftruncateSync(fd, len) {
        return this.baseFs.ftruncateSync(fd, len);
    }
    watch(p, a, b) {
        return this.baseFs.watch(this.mapToBase(p), 
        // @ts-expect-error
        a, b);
    }
    watchFile(p, a, b) {
        return this.baseFs.watchFile(this.mapToBase(p), 
        // @ts-expect-error
        a, b);
    }
    unwatchFile(p, cb) {
        return this.baseFs.unwatchFile(this.mapToBase(p), cb);
    }
    fsMapToBase(p) {
        if (typeof p === `number`) {
            return p;
        }
        else {
            return this.mapToBase(p);
        }
    }
}
exports.ProxiedFS = ProxiedFS;
