"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.LibzipError = exports.ERR_DIR_CLOSED = exports.EOPNOTSUPP = exports.ENOTEMPTY = exports.EROFS = exports.EEXIST = exports.EISDIR = exports.ENOTDIR = exports.ENOENT = exports.EBADF = exports.EINVAL = exports.ENOSYS = exports.EBUSY = void 0;
function makeError(code, message) {
    return Object.assign(new Error(`${code}: ${message}`), { code });
}
function EBUSY(message) {
    return makeError(`EBUSY`, message);
}
exports.EBUSY = EBUSY;
function ENOSYS(message, reason) {
    return makeError(`ENOSYS`, `${message}, ${reason}`);
}
exports.ENOSYS = ENOSYS;
function EINVAL(reason) {
    return makeError(`EINVAL`, `invalid argument, ${reason}`);
}
exports.EINVAL = EINVAL;
function EBADF(reason) {
    return makeError(`EBADF`, `bad file descriptor, ${reason}`);
}
exports.EBADF = EBADF;
function ENOENT(reason) {
    return makeError(`ENOENT`, `no such file or directory, ${reason}`);
}
exports.ENOENT = ENOENT;
function ENOTDIR(reason) {
    return makeError(`ENOTDIR`, `not a directory, ${reason}`);
}
exports.ENOTDIR = ENOTDIR;
function EISDIR(reason) {
    return makeError(`EISDIR`, `illegal operation on a directory, ${reason}`);
}
exports.EISDIR = EISDIR;
function EEXIST(reason) {
    return makeError(`EEXIST`, `file already exists, ${reason}`);
}
exports.EEXIST = EEXIST;
function EROFS(reason) {
    return makeError(`EROFS`, `read-only filesystem, ${reason}`);
}
exports.EROFS = EROFS;
function ENOTEMPTY(reason) {
    return makeError(`ENOTEMPTY`, `directory not empty, ${reason}`);
}
exports.ENOTEMPTY = ENOTEMPTY;
function EOPNOTSUPP(reason) {
    return makeError(`EOPNOTSUPP`, `operation not supported, ${reason}`);
}
exports.EOPNOTSUPP = EOPNOTSUPP;
// ------------------------------------------------------------------------
function ERR_DIR_CLOSED() {
    return makeError(`ERR_DIR_CLOSED`, `Directory handle was closed`);
}
exports.ERR_DIR_CLOSED = ERR_DIR_CLOSED;
// ------------------------------------------------------------------------
class LibzipError extends Error {
    constructor(message, code) {
        super(message);
        this.name = `Libzip Error`;
        this.code = code;
    }
}
exports.LibzipError = LibzipError;
