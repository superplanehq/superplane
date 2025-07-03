"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.opendir = exports.CustomDir = void 0;
const tslib_1 = require("tslib");
const errors = tslib_1.__importStar(require("../errors"));
class CustomDir {
    constructor(path, nextDirent, opts = {}) {
        this.path = path;
        this.nextDirent = nextDirent;
        this.opts = opts;
        this.closed = false;
    }
    throwIfClosed() {
        if (this.closed) {
            throw errors.ERR_DIR_CLOSED();
        }
    }
    async *[Symbol.asyncIterator]() {
        try {
            let dirent;
            // eslint-disable-next-line no-cond-assign
            while ((dirent = await this.read()) !== null) {
                yield dirent;
            }
        }
        finally {
            await this.close();
        }
    }
    read(cb) {
        const dirent = this.readSync();
        if (typeof cb !== `undefined`)
            return cb(null, dirent);
        return Promise.resolve(dirent);
    }
    readSync() {
        this.throwIfClosed();
        return this.nextDirent();
    }
    close(cb) {
        this.closeSync();
        if (typeof cb !== `undefined`)
            return cb(null);
        return Promise.resolve();
    }
    closeSync() {
        var _a, _b;
        this.throwIfClosed();
        (_b = (_a = this.opts).onClose) === null || _b === void 0 ? void 0 : _b.call(_a);
        this.closed = true;
    }
}
exports.CustomDir = CustomDir;
function opendir(fakeFs, path, entries, opts) {
    const nextDirent = () => {
        const filename = entries.shift();
        if (typeof filename === `undefined`)
            return null;
        return Object.assign(fakeFs.statSync(fakeFs.pathUtils.join(path, filename)), {
            name: filename,
        });
    };
    return new CustomDir(path, nextDirent, opts);
}
exports.opendir = opendir;
