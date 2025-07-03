"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.NoFS = void 0;
const FakeFS_1 = require("./FakeFS");
const path_1 = require("./path");
const makeError = () => Object.assign(new Error(`ENOSYS: unsupported filesystem access`), { code: `ENOSYS` });
class NoFS extends FakeFS_1.FakeFS {
    constructor() {
        super(path_1.ppath);
    }
    getExtractHint() {
        throw makeError();
    }
    getRealPath() {
        throw makeError();
    }
    resolve() {
        throw makeError();
    }
    async openPromise() {
        throw makeError();
    }
    openSync() {
        throw makeError();
    }
    async opendirPromise() {
        throw makeError();
    }
    opendirSync() {
        throw makeError();
    }
    async readPromise() {
        throw makeError();
    }
    readSync() {
        throw makeError();
    }
    async writePromise() {
        throw makeError();
    }
    writeSync() {
        throw makeError();
    }
    async closePromise() {
        throw makeError();
    }
    closeSync() {
        throw makeError();
    }
    createWriteStream() {
        throw makeError();
    }
    createReadStream() {
        throw makeError();
    }
    async realpathPromise() {
        throw makeError();
    }
    realpathSync() {
        throw makeError();
    }
    async readdirPromise() {
        throw makeError();
    }
    readdirSync() {
        throw makeError();
    }
    async existsPromise(p) {
        throw makeError();
    }
    existsSync(p) {
        throw makeError();
    }
    async accessPromise() {
        throw makeError();
    }
    accessSync() {
        throw makeError();
    }
    async statPromise() {
        throw makeError();
    }
    statSync() {
        throw makeError();
    }
    async fstatPromise(fd) {
        throw makeError();
    }
    fstatSync(fd) {
        throw makeError();
    }
    async lstatPromise(p) {
        throw makeError();
    }
    lstatSync(p) {
        throw makeError();
    }
    async fchmodPromise() {
        throw makeError();
    }
    fchmodSync() {
        throw makeError();
    }
    async chmodPromise() {
        throw makeError();
    }
    chmodSync() {
        throw makeError();
    }
    async fchownPromise() {
        throw makeError();
    }
    fchownSync() {
        throw makeError();
    }
    async chownPromise() {
        throw makeError();
    }
    chownSync() {
        throw makeError();
    }
    async mkdirPromise() {
        throw makeError();
    }
    mkdirSync() {
        throw makeError();
    }
    async rmdirPromise() {
        throw makeError();
    }
    rmdirSync() {
        throw makeError();
    }
    async linkPromise() {
        throw makeError();
    }
    linkSync() {
        throw makeError();
    }
    async symlinkPromise() {
        throw makeError();
    }
    symlinkSync() {
        throw makeError();
    }
    async renamePromise() {
        throw makeError();
    }
    renameSync() {
        throw makeError();
    }
    async copyFilePromise() {
        throw makeError();
    }
    copyFileSync() {
        throw makeError();
    }
    async appendFilePromise() {
        throw makeError();
    }
    appendFileSync() {
        throw makeError();
    }
    async writeFilePromise() {
        throw makeError();
    }
    writeFileSync() {
        throw makeError();
    }
    async unlinkPromise() {
        throw makeError();
    }
    unlinkSync() {
        throw makeError();
    }
    async utimesPromise() {
        throw makeError();
    }
    utimesSync() {
        throw makeError();
    }
    async readFilePromise() {
        throw makeError();
    }
    readFileSync() {
        throw makeError();
    }
    async readlinkPromise() {
        throw makeError();
    }
    readlinkSync() {
        throw makeError();
    }
    async truncatePromise() {
        throw makeError();
    }
    truncateSync() {
        throw makeError();
    }
    async ftruncatePromise(fd, len) {
        throw makeError();
    }
    ftruncateSync(fd, len) {
        throw makeError();
    }
    watch() {
        throw makeError();
    }
    watchFile() {
        throw makeError();
    }
    unwatchFile() {
        throw makeError();
    }
}
NoFS.instance = new NoFS();
exports.NoFS = NoFS;
