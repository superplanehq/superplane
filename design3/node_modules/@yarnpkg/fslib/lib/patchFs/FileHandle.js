"use strict";
var _a, _b, _c, _d;
Object.defineProperty(exports, "__esModule", { value: true });
exports.FileHandle = void 0;
const readline_1 = require("readline");
const kBaseFs = Symbol(`kBaseFs`);
const kFd = Symbol(`kFd`);
const kClosePromise = Symbol(`kClosePromise`);
const kCloseResolve = Symbol(`kCloseResolve`);
const kCloseReject = Symbol(`kCloseReject`);
const kRefs = Symbol(`kRefs`);
const kRef = Symbol(`kRef`);
const kUnref = Symbol(`kUnref`);
class FileHandle {
    constructor(fd, baseFs) {
        this[_a] = 1;
        this[_b] = undefined;
        this[_c] = undefined;
        this[_d] = undefined;
        this[kBaseFs] = baseFs;
        this[kFd] = fd;
    }
    get fd() {
        return this[kFd];
    }
    async appendFile(data, options) {
        var _e;
        try {
            this[kRef](this.appendFile);
            const encoding = (_e = (typeof options === `string` ? options : options === null || options === void 0 ? void 0 : options.encoding)) !== null && _e !== void 0 ? _e : undefined;
            return await this[kBaseFs].appendFilePromise(this.fd, data, encoding ? { encoding } : undefined);
        }
        finally {
            this[kUnref]();
        }
    }
    async chown(uid, gid) {
        try {
            this[kRef](this.chown);
            return await this[kBaseFs].fchownPromise(this.fd, uid, gid);
        }
        finally {
            this[kUnref]();
        }
    }
    async chmod(mode) {
        try {
            this[kRef](this.chmod);
            return await this[kBaseFs].fchmodPromise(this.fd, mode);
        }
        finally {
            this[kUnref]();
        }
    }
    createReadStream(options) {
        return this[kBaseFs].createReadStream(null, { ...options, fd: this.fd });
    }
    createWriteStream(options) {
        return this[kBaseFs].createWriteStream(null, { ...options, fd: this.fd });
    }
    // FIXME: Missing FakeFS version
    datasync() {
        throw new Error(`Method not implemented.`);
    }
    // FIXME: Missing FakeFS version
    sync() {
        throw new Error(`Method not implemented.`);
    }
    async read(bufferOrOptions, offset, length, position) {
        var _e, _f, _g;
        try {
            this[kRef](this.read);
            let buffer;
            if (!Buffer.isBuffer(bufferOrOptions)) {
                bufferOrOptions !== null && bufferOrOptions !== void 0 ? bufferOrOptions : (bufferOrOptions = {});
                buffer = (_e = bufferOrOptions.buffer) !== null && _e !== void 0 ? _e : Buffer.alloc(16384);
                offset = bufferOrOptions.offset || 0;
                length = (_f = bufferOrOptions.length) !== null && _f !== void 0 ? _f : buffer.byteLength;
                position = (_g = bufferOrOptions.position) !== null && _g !== void 0 ? _g : null;
            }
            else {
                buffer = bufferOrOptions;
            }
            offset !== null && offset !== void 0 ? offset : (offset = 0);
            length !== null && length !== void 0 ? length : (length = 0);
            if (length === 0) {
                return {
                    bytesRead: length,
                    buffer,
                };
            }
            const bytesRead = await this[kBaseFs].readPromise(this.fd, buffer, offset, length, position);
            return {
                bytesRead,
                buffer,
            };
        }
        finally {
            this[kUnref]();
        }
    }
    async readFile(options) {
        var _e;
        try {
            this[kRef](this.readFile);
            const encoding = (_e = (typeof options === `string` ? options : options === null || options === void 0 ? void 0 : options.encoding)) !== null && _e !== void 0 ? _e : undefined;
            return await this[kBaseFs].readFilePromise(this.fd, encoding);
        }
        finally {
            this[kUnref]();
        }
    }
    readLines(options) {
        return (0, readline_1.createInterface)({
            input: this.createReadStream(options),
            crlfDelay: Infinity,
        });
    }
    async stat(opts) {
        try {
            this[kRef](this.stat);
            return await this[kBaseFs].fstatPromise(this.fd, opts);
        }
        finally {
            this[kUnref]();
        }
    }
    async truncate(len) {
        try {
            this[kRef](this.truncate);
            return await this[kBaseFs].ftruncatePromise(this.fd, len);
        }
        finally {
            this[kUnref]();
        }
    }
    // FIXME: Missing FakeFS version
    utimes(atime, mtime) {
        throw new Error(`Method not implemented.`);
    }
    async writeFile(data, options) {
        var _e;
        try {
            this[kRef](this.writeFile);
            const encoding = (_e = (typeof options === `string` ? options : options === null || options === void 0 ? void 0 : options.encoding)) !== null && _e !== void 0 ? _e : undefined;
            await this[kBaseFs].writeFilePromise(this.fd, data, encoding);
        }
        finally {
            this[kUnref]();
        }
    }
    async write(...args) {
        try {
            this[kRef](this.write);
            if (ArrayBuffer.isView(args[0])) {
                const [buffer, offset, length, position] = args;
                const bytesWritten = await this[kBaseFs].writePromise(this.fd, buffer, offset !== null && offset !== void 0 ? offset : undefined, length !== null && length !== void 0 ? length : undefined, position !== null && position !== void 0 ? position : undefined);
                return { bytesWritten, buffer };
            }
            else {
                const [data, position, encoding] = args;
                // @ts-expect-error - FIXME: Types/implementation need to be updated in FakeFS
                const bytesWritten = await this[kBaseFs].writePromise(this.fd, data, position, encoding);
                return { bytesWritten, buffer: data };
            }
        }
        finally {
            this[kUnref]();
        }
    }
    // TODO: Use writev from FakeFS when that is implemented
    async writev(buffers, position) {
        try {
            this[kRef](this.writev);
            let bytesWritten = 0;
            if (typeof position !== `undefined`) {
                for (const buffer of buffers) {
                    const writeResult = await this.write(buffer, undefined, undefined, position);
                    bytesWritten += writeResult.bytesWritten;
                    position += writeResult.bytesWritten;
                }
            }
            else {
                for (const buffer of buffers) {
                    const writeResult = await this.write(buffer);
                    bytesWritten += writeResult.bytesWritten;
                }
            }
            return {
                buffers,
                bytesWritten,
            };
        }
        finally {
            this[kUnref]();
        }
    }
    // FIXME: Missing FakeFS version
    readv(buffers, position) {
        throw new Error(`Method not implemented.`);
    }
    close() {
        if (this[kFd] === -1)
            return Promise.resolve();
        if (this[kClosePromise])
            return this[kClosePromise];
        this[kRefs]--;
        if (this[kRefs] === 0) {
            const fd = this[kFd];
            this[kFd] = -1;
            this[kClosePromise] = this[kBaseFs].closePromise(fd).finally(() => {
                this[kClosePromise] = undefined;
            });
        }
        else {
            this[kClosePromise] =
                new Promise((resolve, reject) => {
                    this[kCloseResolve] = resolve;
                    this[kCloseReject] = reject;
                }).finally(() => {
                    this[kClosePromise] = undefined;
                    this[kCloseReject] = undefined;
                    this[kCloseResolve] = undefined;
                });
        }
        return this[kClosePromise];
    }
    [(_a = kRefs, _b = kClosePromise, _c = kCloseResolve, _d = kCloseReject, kRef)](caller) {
        if (this[kFd] === -1) {
            const err = new Error(`file closed`);
            err.code = `EBADF`;
            err.syscall = caller.name;
            throw err;
        }
        this[kRefs]++;
    }
    [kUnref]() {
        this[kRefs]--;
        if (this[kRefs] === 0) {
            const fd = this[kFd];
            this[kFd] = -1;
            this[kBaseFs].closePromise(fd).then(this[kCloseResolve], this[kCloseReject]);
        }
    }
}
exports.FileHandle = FileHandle;
