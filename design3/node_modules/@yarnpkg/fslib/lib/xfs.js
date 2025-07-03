"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.xfs = void 0;
const tslib_1 = require("tslib");
const os_1 = tslib_1.__importDefault(require("os"));
const NodeFS_1 = require("./NodeFS");
const path_1 = require("./path");
function getTempName(prefix) {
    const hash = Math.ceil(Math.random() * 0x100000000).toString(16).padStart(8, `0`);
    return `${prefix}${hash}`;
}
const tmpdirs = new Set();
let tmpEnv = null;
function initTmpEnv() {
    if (tmpEnv)
        return tmpEnv;
    const tmpdir = path_1.npath.toPortablePath(os_1.default.tmpdir());
    const realTmpdir = exports.xfs.realpathSync(tmpdir);
    process.once(`exit`, () => {
        exports.xfs.rmtempSync();
    });
    return tmpEnv = {
        tmpdir,
        realTmpdir,
    };
}
exports.xfs = Object.assign(new NodeFS_1.NodeFS(), {
    detachTemp(p) {
        tmpdirs.delete(p);
    },
    mktempSync(cb) {
        const { tmpdir, realTmpdir } = initTmpEnv();
        while (true) {
            const name = getTempName(`xfs-`);
            try {
                this.mkdirSync(path_1.ppath.join(tmpdir, name));
            }
            catch (error) {
                if (error.code === `EEXIST`) {
                    continue;
                }
                else {
                    throw error;
                }
            }
            const realP = path_1.ppath.join(realTmpdir, name);
            tmpdirs.add(realP);
            if (typeof cb === `undefined`)
                return realP;
            try {
                return cb(realP);
            }
            finally {
                if (tmpdirs.has(realP)) {
                    tmpdirs.delete(realP);
                    try {
                        this.removeSync(realP);
                    }
                    catch {
                        // Too bad if there's an error
                    }
                }
            }
        }
    },
    async mktempPromise(cb) {
        const { tmpdir, realTmpdir } = initTmpEnv();
        while (true) {
            const name = getTempName(`xfs-`);
            try {
                await this.mkdirPromise(path_1.ppath.join(tmpdir, name));
            }
            catch (error) {
                if (error.code === `EEXIST`) {
                    continue;
                }
                else {
                    throw error;
                }
            }
            const realP = path_1.ppath.join(realTmpdir, name);
            tmpdirs.add(realP);
            if (typeof cb === `undefined`)
                return realP;
            try {
                return await cb(realP);
            }
            finally {
                if (tmpdirs.has(realP)) {
                    tmpdirs.delete(realP);
                    try {
                        await this.removePromise(realP);
                    }
                    catch {
                        // Too bad if there's an error
                    }
                }
            }
        }
    },
    async rmtempPromise() {
        await Promise.all(Array.from(tmpdirs.values()).map(async (p) => {
            try {
                await exports.xfs.removePromise(p, { maxRetries: 0 });
                tmpdirs.delete(p);
            }
            catch {
                // Too bad if there's an error
            }
        }));
    },
    rmtempSync() {
        for (const p of tmpdirs) {
            try {
                exports.xfs.removeSync(p);
                tmpdirs.delete(p);
            }
            catch {
                // Too bad if there's an error
            }
        }
    },
});
