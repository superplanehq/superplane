"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ZipFS = exports.makeEmptyArchive = exports.DEFAULT_COMPRESSION_LEVEL = void 0;
const tslib_1 = require("tslib");
const fs_1 = require("fs");
const stream_1 = require("stream");
const util_1 = require("util");
const zlib_1 = tslib_1.__importDefault(require("zlib"));
const FakeFS_1 = require("./FakeFS");
const NodeFS_1 = require("./NodeFS");
const opendir_1 = require("./algorithms/opendir");
const watchFile_1 = require("./algorithms/watchFile");
const constants_1 = require("./constants");
const errors = tslib_1.__importStar(require("./errors"));
const path_1 = require("./path");
const statUtils = tslib_1.__importStar(require("./statUtils"));
exports.DEFAULT_COMPRESSION_LEVEL = `mixed`;
function toUnixTimestamp(time) {
    if (typeof time === `string` && String(+time) === time)
        return +time;
    if (typeof time === `number` && Number.isFinite(time)) {
        if (time < 0) {
            return Date.now() / 1000;
        }
        else {
            return time;
        }
    }
    // convert to 123.456 UNIX timestamp
    if (util_1.types.isDate(time))
        return time.getTime() / 1000;
    throw new Error(`Invalid time`);
}
function makeEmptyArchive() {
    return Buffer.from([
        0x50, 0x4B, 0x05, 0x06,
        0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00,
        0x00, 0x00,
    ]);
}
exports.makeEmptyArchive = makeEmptyArchive;
class ZipFS extends FakeFS_1.BasePortableFakeFS {
    constructor(source, opts) {
        super();
        this.lzSource = null;
        this.listings = new Map();
        this.entries = new Map();
        /**
         * A cache of indices mapped to file sources.
         * Populated by `setFileSource` calls.
         * Required for supporting read after write.
         */
        this.fileSources = new Map();
        this.fds = new Map();
        this.nextFd = 0;
        this.ready = false;
        this.readOnly = false;
        this.libzip = opts.libzip;
        const pathOptions = opts;
        this.level = typeof pathOptions.level !== `undefined`
            ? pathOptions.level
            : exports.DEFAULT_COMPRESSION_LEVEL;
        source !== null && source !== void 0 ? source : (source = makeEmptyArchive());
        if (typeof source === `string`) {
            const { baseFs = new NodeFS_1.NodeFS() } = pathOptions;
            this.baseFs = baseFs;
            this.path = source;
        }
        else {
            this.path = null;
            this.baseFs = null;
        }
        if (opts.stats) {
            this.stats = opts.stats;
        }
        else {
            if (typeof source === `string`) {
                try {
                    this.stats = this.baseFs.statSync(source);
                }
                catch (error) {
                    if (error.code === `ENOENT` && pathOptions.create) {
                        this.stats = statUtils.makeDefaultStats();
                    }
                    else {
                        throw error;
                    }
                }
            }
            else {
                this.stats = statUtils.makeDefaultStats();
            }
        }
        const errPtr = this.libzip.malloc(4);
        try {
            let flags = 0;
            if (typeof source === `string` && pathOptions.create)
                flags |= this.libzip.ZIP_CREATE | this.libzip.ZIP_TRUNCATE;
            if (opts.readOnly) {
                flags |= this.libzip.ZIP_RDONLY;
                this.readOnly = true;
            }
            if (typeof source === `string`) {
                this.zip = this.libzip.open(path_1.npath.fromPortablePath(source), flags, errPtr);
            }
            else {
                const lzSource = this.allocateUnattachedSource(source);
                try {
                    this.zip = this.libzip.openFromSource(lzSource, flags, errPtr);
                    this.lzSource = lzSource;
                }
                catch (error) {
                    this.libzip.source.free(lzSource);
                    throw error;
                }
            }
            if (this.zip === 0) {
                const error = this.libzip.struct.errorS();
                this.libzip.error.initWithCode(error, this.libzip.getValue(errPtr, `i32`));
                throw this.makeLibzipError(error);
            }
        }
        finally {
            this.libzip.free(errPtr);
        }
        this.listings.set(path_1.PortablePath.root, new Set());
        const entryCount = this.libzip.getNumEntries(this.zip, 0);
        for (let t = 0; t < entryCount; ++t) {
            const raw = this.libzip.getName(this.zip, t, 0);
            if (path_1.ppath.isAbsolute(raw))
                continue;
            const p = path_1.ppath.resolve(path_1.PortablePath.root, raw);
            this.registerEntry(p, t);
            // If the raw path is a directory, register it
            // to prevent empty folder being skipped
            if (raw.endsWith(`/`)) {
                this.registerListing(p);
            }
        }
        this.symlinkCount = this.libzip.ext.countSymlinks(this.zip);
        if (this.symlinkCount === -1)
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        this.ready = true;
    }
    makeLibzipError(error) {
        const errorCode = this.libzip.struct.errorCodeZip(error);
        const strerror = this.libzip.error.strerror(error);
        const libzipError = new errors.LibzipError(strerror, this.libzip.errors[errorCode]);
        // This error should never come up because of the file source cache
        if (errorCode === this.libzip.errors.ZIP_ER_CHANGED)
            throw new Error(`Assertion failed: Unexpected libzip error: ${libzipError.message}`);
        return libzipError;
    }
    getExtractHint(hints) {
        for (const fileName of this.entries.keys()) {
            const ext = this.pathUtils.extname(fileName);
            if (hints.relevantExtensions.has(ext)) {
                return true;
            }
        }
        return false;
    }
    getAllFiles() {
        return Array.from(this.entries.keys());
    }
    getRealPath() {
        if (!this.path)
            throw new Error(`ZipFS don't have real paths when loaded from a buffer`);
        return this.path;
    }
    getBufferAndClose() {
        this.prepareClose();
        if (!this.lzSource)
            throw new Error(`ZipFS was not created from a Buffer`);
        // zip_source_open on an unlink-after-write empty archive fails with "Entry has been deleted"
        if (this.entries.size === 0) {
            this.discardAndClose();
            return makeEmptyArchive();
        }
        try {
            // Prevent close from cleaning up the source
            this.libzip.source.keep(this.lzSource);
            // Close the zip archive
            if (this.libzip.close(this.zip) === -1)
                throw this.makeLibzipError(this.libzip.getError(this.zip));
            // Open the source for reading
            if (this.libzip.source.open(this.lzSource) === -1)
                throw this.makeLibzipError(this.libzip.source.error(this.lzSource));
            // Move to the end of source
            if (this.libzip.source.seek(this.lzSource, 0, 0, this.libzip.SEEK_END) === -1)
                throw this.makeLibzipError(this.libzip.source.error(this.lzSource));
            // Get the size of source
            const size = this.libzip.source.tell(this.lzSource);
            if (size === -1)
                throw this.makeLibzipError(this.libzip.source.error(this.lzSource));
            // Move to the start of source
            if (this.libzip.source.seek(this.lzSource, 0, 0, this.libzip.SEEK_SET) === -1)
                throw this.makeLibzipError(this.libzip.source.error(this.lzSource));
            const buffer = this.libzip.malloc(size);
            if (!buffer)
                throw new Error(`Couldn't allocate enough memory`);
            try {
                const rc = this.libzip.source.read(this.lzSource, buffer, size);
                if (rc === -1)
                    throw this.makeLibzipError(this.libzip.source.error(this.lzSource));
                else if (rc < size)
                    throw new Error(`Incomplete read`);
                else if (rc > size)
                    throw new Error(`Overread`);
                const memory = this.libzip.HEAPU8.subarray(buffer, buffer + size);
                return Buffer.from(memory);
            }
            finally {
                this.libzip.free(buffer);
            }
        }
        finally {
            this.libzip.source.close(this.lzSource);
            this.libzip.source.free(this.lzSource);
            this.ready = false;
        }
    }
    prepareClose() {
        if (!this.ready)
            throw errors.EBUSY(`archive closed, close`);
        (0, watchFile_1.unwatchAllFiles)(this);
    }
    saveAndClose() {
        if (!this.path || !this.baseFs)
            throw new Error(`ZipFS cannot be saved and must be discarded when loaded from a buffer`);
        this.prepareClose();
        if (this.readOnly) {
            this.discardAndClose();
            return;
        }
        const newMode = this.baseFs.existsSync(this.path) || this.stats.mode === statUtils.DEFAULT_MODE
            ? undefined
            : this.stats.mode;
        // zip_close doesn't persist empty archives
        if (this.entries.size === 0) {
            this.discardAndClose();
            this.baseFs.writeFileSync(this.path, makeEmptyArchive(), { mode: newMode });
        }
        else {
            const rc = this.libzip.close(this.zip);
            if (rc === -1)
                throw this.makeLibzipError(this.libzip.getError(this.zip));
            if (typeof newMode !== `undefined`) {
                this.baseFs.chmodSync(this.path, newMode);
            }
        }
        this.ready = false;
    }
    discardAndClose() {
        this.prepareClose();
        this.libzip.discard(this.zip);
        this.ready = false;
    }
    resolve(p) {
        return path_1.ppath.resolve(path_1.PortablePath.root, p);
    }
    async openPromise(p, flags, mode) {
        return this.openSync(p, flags, mode);
    }
    openSync(p, flags, mode) {
        const fd = this.nextFd++;
        this.fds.set(fd, { cursor: 0, p });
        return fd;
    }
    hasOpenFileHandles() {
        return !!this.fds.size;
    }
    async opendirPromise(p, opts) {
        return this.opendirSync(p, opts);
    }
    opendirSync(p, opts = {}) {
        const resolvedP = this.resolveFilename(`opendir '${p}'`, p);
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP))
            throw errors.ENOENT(`opendir '${p}'`);
        const directoryListing = this.listings.get(resolvedP);
        if (!directoryListing)
            throw errors.ENOTDIR(`opendir '${p}'`);
        const entries = [...directoryListing];
        const fd = this.openSync(resolvedP, `r`);
        const onClose = () => {
            this.closeSync(fd);
        };
        return (0, opendir_1.opendir)(this, resolvedP, entries, { onClose });
    }
    async readPromise(fd, buffer, offset, length, position) {
        return this.readSync(fd, buffer, offset, length, position);
    }
    readSync(fd, buffer, offset = 0, length = buffer.byteLength, position = -1) {
        const entry = this.fds.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`read`);
        const realPosition = position === -1 || position === null
            ? entry.cursor
            : position;
        const source = this.readFileSync(entry.p);
        source.copy(buffer, offset, realPosition, realPosition + length);
        const bytesRead = Math.max(0, Math.min(source.length - realPosition, length));
        if (position === -1 || position === null)
            entry.cursor += bytesRead;
        return bytesRead;
    }
    async writePromise(fd, buffer, offset, length, position) {
        if (typeof buffer === `string`) {
            return this.writeSync(fd, buffer, position);
        }
        else {
            return this.writeSync(fd, buffer, offset, length, position);
        }
    }
    writeSync(fd, buffer, offset, length, position) {
        const entry = this.fds.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`read`);
        throw new Error(`Unimplemented`);
    }
    async closePromise(fd) {
        return this.closeSync(fd);
    }
    closeSync(fd) {
        const entry = this.fds.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`read`);
        this.fds.delete(fd);
    }
    createReadStream(p, { encoding } = {}) {
        if (p === null)
            throw new Error(`Unimplemented`);
        const fd = this.openSync(p, `r`);
        const stream = Object.assign(new stream_1.PassThrough({
            emitClose: true,
            autoDestroy: true,
            destroy: (error, callback) => {
                clearImmediate(immediate);
                this.closeSync(fd);
                callback(error);
            },
        }), {
            close() {
                stream.destroy();
            },
            bytesRead: 0,
            path: p,
        });
        const immediate = setImmediate(async () => {
            try {
                const data = await this.readFilePromise(p, encoding);
                stream.bytesRead = data.length;
                stream.end(data);
            }
            catch (error) {
                stream.destroy(error);
            }
        });
        return stream;
    }
    createWriteStream(p, { encoding } = {}) {
        if (this.readOnly)
            throw errors.EROFS(`open '${p}'`);
        if (p === null)
            throw new Error(`Unimplemented`);
        const chunks = [];
        const fd = this.openSync(p, `w`);
        const stream = Object.assign(new stream_1.PassThrough({
            autoDestroy: true,
            emitClose: true,
            destroy: (error, callback) => {
                try {
                    if (error) {
                        callback(error);
                    }
                    else {
                        this.writeFileSync(p, Buffer.concat(chunks), encoding);
                        callback(null);
                    }
                }
                catch (err) {
                    callback(err);
                }
                finally {
                    this.closeSync(fd);
                }
            },
        }), {
            bytesWritten: 0,
            path: p,
            close() {
                stream.destroy();
            },
        });
        stream.on(`data`, chunk => {
            const chunkBuffer = Buffer.from(chunk);
            stream.bytesWritten += chunkBuffer.length;
            chunks.push(chunkBuffer);
        });
        return stream;
    }
    async realpathPromise(p) {
        return this.realpathSync(p);
    }
    realpathSync(p) {
        const resolvedP = this.resolveFilename(`lstat '${p}'`, p);
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP))
            throw errors.ENOENT(`lstat '${p}'`);
        return resolvedP;
    }
    async existsPromise(p) {
        return this.existsSync(p);
    }
    existsSync(p) {
        if (!this.ready)
            throw errors.EBUSY(`archive closed, existsSync '${p}'`);
        if (this.symlinkCount === 0) {
            const resolvedP = path_1.ppath.resolve(path_1.PortablePath.root, p);
            return this.entries.has(resolvedP) || this.listings.has(resolvedP);
        }
        let resolvedP;
        try {
            resolvedP = this.resolveFilename(`stat '${p}'`, p, undefined, false);
        }
        catch (error) {
            return false;
        }
        if (resolvedP === undefined)
            return false;
        return this.entries.has(resolvedP) || this.listings.has(resolvedP);
    }
    async accessPromise(p, mode) {
        return this.accessSync(p, mode);
    }
    accessSync(p, mode = fs_1.constants.F_OK) {
        const resolvedP = this.resolveFilename(`access '${p}'`, p);
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP))
            throw errors.ENOENT(`access '${p}'`);
        if (this.readOnly && (mode & fs_1.constants.W_OK)) {
            throw errors.EROFS(`access '${p}'`);
        }
    }
    async statPromise(p, opts = { bigint: false }) {
        if (opts.bigint)
            return this.statSync(p, { bigint: true });
        return this.statSync(p);
    }
    statSync(p, opts = { bigint: false, throwIfNoEntry: true }) {
        const resolvedP = this.resolveFilename(`stat '${p}'`, p, undefined, opts.throwIfNoEntry);
        if (resolvedP === undefined)
            return undefined;
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP)) {
            if (opts.throwIfNoEntry === false)
                return undefined;
            throw errors.ENOENT(`stat '${p}'`);
        }
        if (p[p.length - 1] === `/` && !this.listings.has(resolvedP))
            throw errors.ENOTDIR(`stat '${p}'`);
        return this.statImpl(`stat '${p}'`, resolvedP, opts);
    }
    async fstatPromise(fd, opts) {
        return this.fstatSync(fd, opts);
    }
    fstatSync(fd, opts) {
        const entry = this.fds.get(fd);
        if (typeof entry === `undefined`)
            throw errors.EBADF(`fstatSync`);
        const { p } = entry;
        const resolvedP = this.resolveFilename(`stat '${p}'`, p);
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP))
            throw errors.ENOENT(`stat '${p}'`);
        if (p[p.length - 1] === `/` && !this.listings.has(resolvedP))
            throw errors.ENOTDIR(`stat '${p}'`);
        return this.statImpl(`fstat '${p}'`, resolvedP, opts);
    }
    async lstatPromise(p, opts = { bigint: false }) {
        if (opts.bigint)
            return this.lstatSync(p, { bigint: true });
        return this.lstatSync(p);
    }
    lstatSync(p, opts = { bigint: false, throwIfNoEntry: true }) {
        const resolvedP = this.resolveFilename(`lstat '${p}'`, p, false, opts.throwIfNoEntry);
        if (resolvedP === undefined)
            return undefined;
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP)) {
            if (opts.throwIfNoEntry === false)
                return undefined;
            throw errors.ENOENT(`lstat '${p}'`);
        }
        if (p[p.length - 1] === `/` && !this.listings.has(resolvedP))
            throw errors.ENOTDIR(`lstat '${p}'`);
        return this.statImpl(`lstat '${p}'`, resolvedP, opts);
    }
    statImpl(reason, p, opts = {}) {
        const entry = this.entries.get(p);
        // File, or explicit directory
        if (typeof entry !== `undefined`) {
            const stat = this.libzip.struct.statS();
            const rc = this.libzip.statIndex(this.zip, entry, 0, 0, stat);
            if (rc === -1)
                throw this.makeLibzipError(this.libzip.getError(this.zip));
            const uid = this.stats.uid;
            const gid = this.stats.gid;
            const size = (this.libzip.struct.statSize(stat) >>> 0);
            const blksize = 512;
            const blocks = Math.ceil(size / blksize);
            const mtimeMs = (this.libzip.struct.statMtime(stat) >>> 0) * 1000;
            const atimeMs = mtimeMs;
            const birthtimeMs = mtimeMs;
            const ctimeMs = mtimeMs;
            const atime = new Date(atimeMs);
            const birthtime = new Date(birthtimeMs);
            const ctime = new Date(ctimeMs);
            const mtime = new Date(mtimeMs);
            const type = this.listings.has(p)
                ? constants_1.S_IFDIR
                : this.isSymbolicLink(entry)
                    ? constants_1.S_IFLNK
                    : constants_1.S_IFREG;
            const defaultMode = type === constants_1.S_IFDIR
                ? 0o755
                : 0o644;
            const mode = type | (this.getUnixMode(entry, defaultMode) & 0o777);
            const crc = this.libzip.struct.statCrc(stat);
            const statInstance = Object.assign(new statUtils.StatEntry(), { uid, gid, size, blksize, blocks, atime, birthtime, ctime, mtime, atimeMs, birthtimeMs, ctimeMs, mtimeMs, mode, crc });
            return opts.bigint === true ? statUtils.convertToBigIntStats(statInstance) : statInstance;
        }
        // Implicit directory
        if (this.listings.has(p)) {
            const uid = this.stats.uid;
            const gid = this.stats.gid;
            const size = 0;
            const blksize = 512;
            const blocks = 0;
            const atimeMs = this.stats.mtimeMs;
            const birthtimeMs = this.stats.mtimeMs;
            const ctimeMs = this.stats.mtimeMs;
            const mtimeMs = this.stats.mtimeMs;
            const atime = new Date(atimeMs);
            const birthtime = new Date(birthtimeMs);
            const ctime = new Date(ctimeMs);
            const mtime = new Date(mtimeMs);
            const mode = constants_1.S_IFDIR | 0o755;
            const crc = 0;
            const statInstance = Object.assign(new statUtils.StatEntry(), { uid, gid, size, blksize, blocks, atime, birthtime, ctime, mtime, atimeMs, birthtimeMs, ctimeMs, mtimeMs, mode, crc });
            return opts.bigint === true ? statUtils.convertToBigIntStats(statInstance) : statInstance;
        }
        throw new Error(`Unreachable`);
    }
    getUnixMode(index, defaultMode) {
        const rc = this.libzip.file.getExternalAttributes(this.zip, index, 0, 0, this.libzip.uint08S, this.libzip.uint32S);
        if (rc === -1)
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        const opsys = this.libzip.getValue(this.libzip.uint08S, `i8`) >>> 0;
        if (opsys !== this.libzip.ZIP_OPSYS_UNIX)
            return defaultMode;
        return this.libzip.getValue(this.libzip.uint32S, `i32`) >>> 16;
    }
    registerListing(p) {
        const existingListing = this.listings.get(p);
        if (existingListing)
            return existingListing;
        const parentListing = this.registerListing(path_1.ppath.dirname(p));
        parentListing.add(path_1.ppath.basename(p));
        const newListing = new Set();
        this.listings.set(p, newListing);
        return newListing;
    }
    registerEntry(p, index) {
        const parentListing = this.registerListing(path_1.ppath.dirname(p));
        parentListing.add(path_1.ppath.basename(p));
        this.entries.set(p, index);
    }
    unregisterListing(p) {
        this.listings.delete(p);
        const parentListing = this.listings.get(path_1.ppath.dirname(p));
        parentListing === null || parentListing === void 0 ? void 0 : parentListing.delete(path_1.ppath.basename(p));
    }
    unregisterEntry(p) {
        this.unregisterListing(p);
        const entry = this.entries.get(p);
        this.entries.delete(p);
        if (typeof entry === `undefined`)
            return;
        this.fileSources.delete(entry);
        if (this.isSymbolicLink(entry)) {
            this.symlinkCount--;
        }
    }
    deleteEntry(p, index) {
        this.unregisterEntry(p);
        const rc = this.libzip.delete(this.zip, index);
        if (rc === -1) {
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        }
    }
    resolveFilename(reason, p, resolveLastComponent = true, throwIfNoEntry = true) {
        if (!this.ready)
            throw errors.EBUSY(`archive closed, ${reason}`);
        let resolvedP = path_1.ppath.resolve(path_1.PortablePath.root, p);
        if (resolvedP === `/`)
            return path_1.PortablePath.root;
        const fileIndex = this.entries.get(resolvedP);
        if (resolveLastComponent && fileIndex !== undefined) {
            if (this.symlinkCount !== 0 && this.isSymbolicLink(fileIndex)) {
                const target = this.getFileSource(fileIndex).toString();
                return this.resolveFilename(reason, path_1.ppath.resolve(path_1.ppath.dirname(resolvedP), target), true, throwIfNoEntry);
            }
            else {
                return resolvedP;
            }
        }
        while (true) {
            const parentP = this.resolveFilename(reason, path_1.ppath.dirname(resolvedP), true, throwIfNoEntry);
            if (parentP === undefined)
                return parentP;
            const isDir = this.listings.has(parentP);
            const doesExist = this.entries.has(parentP);
            if (!isDir && !doesExist) {
                if (throwIfNoEntry === false)
                    return undefined;
                throw errors.ENOENT(reason);
            }
            if (!isDir)
                throw errors.ENOTDIR(reason);
            resolvedP = path_1.ppath.resolve(parentP, path_1.ppath.basename(resolvedP));
            if (!resolveLastComponent || this.symlinkCount === 0)
                break;
            const index = this.libzip.name.locate(this.zip, resolvedP.slice(1), 0);
            if (index === -1)
                break;
            if (this.isSymbolicLink(index)) {
                const target = this.getFileSource(index).toString();
                resolvedP = path_1.ppath.resolve(path_1.ppath.dirname(resolvedP), target);
            }
            else {
                break;
            }
        }
        return resolvedP;
    }
    allocateBuffer(content) {
        if (!Buffer.isBuffer(content))
            content = Buffer.from(content);
        const buffer = this.libzip.malloc(content.byteLength);
        if (!buffer)
            throw new Error(`Couldn't allocate enough memory`);
        // Copy the file into the Emscripten heap
        const heap = new Uint8Array(this.libzip.HEAPU8.buffer, buffer, content.byteLength);
        heap.set(content);
        return { buffer, byteLength: content.byteLength };
    }
    allocateUnattachedSource(content) {
        const error = this.libzip.struct.errorS();
        const { buffer, byteLength } = this.allocateBuffer(content);
        const source = this.libzip.source.fromUnattachedBuffer(buffer, byteLength, 0, 1, error);
        if (source === 0) {
            this.libzip.free(error);
            throw this.makeLibzipError(error);
        }
        return source;
    }
    allocateSource(content) {
        const { buffer, byteLength } = this.allocateBuffer(content);
        const source = this.libzip.source.fromBuffer(this.zip, buffer, byteLength, 0, 1);
        if (source === 0) {
            this.libzip.free(buffer);
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        }
        return source;
    }
    setFileSource(p, content) {
        const buffer = Buffer.isBuffer(content) ? content : Buffer.from(content);
        const target = path_1.ppath.relative(path_1.PortablePath.root, p);
        const lzSource = this.allocateSource(content);
        try {
            const newIndex = this.libzip.file.add(this.zip, target, lzSource, this.libzip.ZIP_FL_OVERWRITE);
            if (newIndex === -1)
                throw this.makeLibzipError(this.libzip.getError(this.zip));
            if (this.level !== `mixed`) {
                // Use store for level 0, and deflate for 1..9
                const method = this.level === 0
                    ? this.libzip.ZIP_CM_STORE
                    : this.libzip.ZIP_CM_DEFLATE;
                const rc = this.libzip.file.setCompression(this.zip, newIndex, 0, method, this.level);
                if (rc === -1) {
                    throw this.makeLibzipError(this.libzip.getError(this.zip));
                }
            }
            this.fileSources.set(newIndex, buffer);
            return newIndex;
        }
        catch (error) {
            this.libzip.source.free(lzSource);
            throw error;
        }
    }
    isSymbolicLink(index) {
        if (this.symlinkCount === 0)
            return false;
        const attrs = this.libzip.file.getExternalAttributes(this.zip, index, 0, 0, this.libzip.uint08S, this.libzip.uint32S);
        if (attrs === -1)
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        const opsys = this.libzip.getValue(this.libzip.uint08S, `i8`) >>> 0;
        if (opsys !== this.libzip.ZIP_OPSYS_UNIX)
            return false;
        const attributes = this.libzip.getValue(this.libzip.uint32S, `i32`) >>> 16;
        return (attributes & constants_1.S_IFMT) === constants_1.S_IFLNK;
    }
    getFileSource(index, opts = { asyncDecompress: false }) {
        const cachedFileSource = this.fileSources.get(index);
        if (typeof cachedFileSource !== `undefined`)
            return cachedFileSource;
        const stat = this.libzip.struct.statS();
        const rc = this.libzip.statIndex(this.zip, index, 0, 0, stat);
        if (rc === -1)
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        const size = this.libzip.struct.statCompSize(stat);
        const compressionMethod = this.libzip.struct.statCompMethod(stat);
        const buffer = this.libzip.malloc(size);
        try {
            const file = this.libzip.fopenIndex(this.zip, index, 0, this.libzip.ZIP_FL_COMPRESSED);
            if (file === 0)
                throw this.makeLibzipError(this.libzip.getError(this.zip));
            try {
                const rc = this.libzip.fread(file, buffer, size, 0);
                if (rc === -1)
                    throw this.makeLibzipError(this.libzip.file.getError(file));
                else if (rc < size)
                    throw new Error(`Incomplete read`);
                else if (rc > size)
                    throw new Error(`Overread`);
                const memory = this.libzip.HEAPU8.subarray(buffer, buffer + size);
                const data = Buffer.from(memory);
                if (compressionMethod === 0) {
                    this.fileSources.set(index, data);
                    return data;
                }
                else if (opts.asyncDecompress) {
                    return new Promise((resolve, reject) => {
                        zlib_1.default.inflateRaw(data, (error, result) => {
                            if (error) {
                                reject(error);
                            }
                            else {
                                this.fileSources.set(index, result);
                                resolve(result);
                            }
                        });
                    });
                }
                else {
                    const decompressedData = zlib_1.default.inflateRawSync(data);
                    this.fileSources.set(index, decompressedData);
                    return decompressedData;
                }
            }
            finally {
                this.libzip.fclose(file);
            }
        }
        finally {
            this.libzip.free(buffer);
        }
    }
    async fchmodPromise(fd, mask) {
        return this.chmodPromise(this.fdToPath(fd, `fchmod`), mask);
    }
    fchmodSync(fd, mask) {
        return this.chmodSync(this.fdToPath(fd, `fchmodSync`), mask);
    }
    async chmodPromise(p, mask) {
        return this.chmodSync(p, mask);
    }
    chmodSync(p, mask) {
        if (this.readOnly)
            throw errors.EROFS(`chmod '${p}'`);
        // We don't allow to make the extracted entries group-writable
        mask &= 0o755;
        const resolvedP = this.resolveFilename(`chmod '${p}'`, p, false);
        const entry = this.entries.get(resolvedP);
        if (typeof entry === `undefined`)
            throw new Error(`Assertion failed: The entry should have been registered (${resolvedP})`);
        const oldMod = this.getUnixMode(entry, constants_1.S_IFREG | 0o000);
        const newMod = oldMod & (~0o777) | mask;
        const rc = this.libzip.file.setExternalAttributes(this.zip, entry, 0, 0, this.libzip.ZIP_OPSYS_UNIX, newMod << 16);
        if (rc === -1) {
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        }
    }
    async fchownPromise(fd, uid, gid) {
        return this.chownPromise(this.fdToPath(fd, `fchown`), uid, gid);
    }
    fchownSync(fd, uid, gid) {
        return this.chownSync(this.fdToPath(fd, `fchownSync`), uid, gid);
    }
    async chownPromise(p, uid, gid) {
        return this.chownSync(p, uid, gid);
    }
    chownSync(p, uid, gid) {
        throw new Error(`Unimplemented`);
    }
    async renamePromise(oldP, newP) {
        return this.renameSync(oldP, newP);
    }
    renameSync(oldP, newP) {
        throw new Error(`Unimplemented`);
    }
    async copyFilePromise(sourceP, destP, flags) {
        const { indexSource, indexDest, resolvedDestP } = this.prepareCopyFile(sourceP, destP, flags);
        const source = await this.getFileSource(indexSource, { asyncDecompress: true });
        const newIndex = this.setFileSource(resolvedDestP, source);
        if (newIndex !== indexDest) {
            this.registerEntry(resolvedDestP, newIndex);
        }
    }
    copyFileSync(sourceP, destP, flags = 0) {
        const { indexSource, indexDest, resolvedDestP } = this.prepareCopyFile(sourceP, destP, flags);
        const source = this.getFileSource(indexSource);
        const newIndex = this.setFileSource(resolvedDestP, source);
        if (newIndex !== indexDest) {
            this.registerEntry(resolvedDestP, newIndex);
        }
    }
    prepareCopyFile(sourceP, destP, flags = 0) {
        if (this.readOnly)
            throw errors.EROFS(`copyfile '${sourceP} -> '${destP}'`);
        if ((flags & fs_1.constants.COPYFILE_FICLONE_FORCE) !== 0)
            throw errors.ENOSYS(`unsupported clone operation`, `copyfile '${sourceP}' -> ${destP}'`);
        const resolvedSourceP = this.resolveFilename(`copyfile '${sourceP} -> ${destP}'`, sourceP);
        const indexSource = this.entries.get(resolvedSourceP);
        if (typeof indexSource === `undefined`)
            throw errors.EINVAL(`copyfile '${sourceP}' -> '${destP}'`);
        const resolvedDestP = this.resolveFilename(`copyfile '${sourceP}' -> ${destP}'`, destP);
        const indexDest = this.entries.get(resolvedDestP);
        if ((flags & (fs_1.constants.COPYFILE_EXCL | fs_1.constants.COPYFILE_FICLONE_FORCE)) !== 0 && typeof indexDest !== `undefined`)
            throw errors.EEXIST(`copyfile '${sourceP}' -> '${destP}'`);
        return {
            indexSource,
            resolvedDestP,
            indexDest,
        };
    }
    async appendFilePromise(p, content, opts) {
        if (this.readOnly)
            throw errors.EROFS(`open '${p}'`);
        if (typeof opts === `undefined`)
            opts = { flag: `a` };
        else if (typeof opts === `string`)
            opts = { flag: `a`, encoding: opts };
        else if (typeof opts.flag === `undefined`)
            opts = { flag: `a`, ...opts };
        return this.writeFilePromise(p, content, opts);
    }
    appendFileSync(p, content, opts = {}) {
        if (this.readOnly)
            throw errors.EROFS(`open '${p}'`);
        if (typeof opts === `undefined`)
            opts = { flag: `a` };
        else if (typeof opts === `string`)
            opts = { flag: `a`, encoding: opts };
        else if (typeof opts.flag === `undefined`)
            opts = { flag: `a`, ...opts };
        return this.writeFileSync(p, content, opts);
    }
    fdToPath(fd, reason) {
        var _a;
        const path = (_a = this.fds.get(fd)) === null || _a === void 0 ? void 0 : _a.p;
        if (typeof path === `undefined`)
            throw errors.EBADF(reason);
        return path;
    }
    async writeFilePromise(p, content, opts) {
        const { encoding, mode, index, resolvedP } = this.prepareWriteFile(p, opts);
        if (index !== undefined && typeof opts === `object` && opts.flag && opts.flag.includes(`a`))
            content = Buffer.concat([await this.getFileSource(index, { asyncDecompress: true }), Buffer.from(content)]);
        if (encoding !== null)
            content = content.toString(encoding);
        const newIndex = this.setFileSource(resolvedP, content);
        if (newIndex !== index)
            this.registerEntry(resolvedP, newIndex);
        if (mode !== null) {
            await this.chmodPromise(resolvedP, mode);
        }
    }
    writeFileSync(p, content, opts) {
        const { encoding, mode, index, resolvedP } = this.prepareWriteFile(p, opts);
        if (index !== undefined && typeof opts === `object` && opts.flag && opts.flag.includes(`a`))
            content = Buffer.concat([this.getFileSource(index), Buffer.from(content)]);
        if (encoding !== null)
            content = content.toString(encoding);
        const newIndex = this.setFileSource(resolvedP, content);
        if (newIndex !== index)
            this.registerEntry(resolvedP, newIndex);
        if (mode !== null) {
            this.chmodSync(resolvedP, mode);
        }
    }
    prepareWriteFile(p, opts) {
        if (typeof p === `number`)
            p = this.fdToPath(p, `read`);
        if (this.readOnly)
            throw errors.EROFS(`open '${p}'`);
        const resolvedP = this.resolveFilename(`open '${p}'`, p);
        if (this.listings.has(resolvedP))
            throw errors.EISDIR(`open '${p}'`);
        let encoding = null, mode = null;
        if (typeof opts === `string`) {
            encoding = opts;
        }
        else if (typeof opts === `object`) {
            ({
                encoding = null,
                mode = null,
            } = opts);
        }
        const index = this.entries.get(resolvedP);
        return {
            encoding,
            mode,
            resolvedP,
            index,
        };
    }
    async unlinkPromise(p) {
        return this.unlinkSync(p);
    }
    unlinkSync(p) {
        if (this.readOnly)
            throw errors.EROFS(`unlink '${p}'`);
        const resolvedP = this.resolveFilename(`unlink '${p}'`, p);
        if (this.listings.has(resolvedP))
            throw errors.EISDIR(`unlink '${p}'`);
        const index = this.entries.get(resolvedP);
        if (typeof index === `undefined`)
            throw errors.EINVAL(`unlink '${p}'`);
        this.deleteEntry(resolvedP, index);
    }
    async utimesPromise(p, atime, mtime) {
        return this.utimesSync(p, atime, mtime);
    }
    utimesSync(p, atime, mtime) {
        if (this.readOnly)
            throw errors.EROFS(`utimes '${p}'`);
        const resolvedP = this.resolveFilename(`utimes '${p}'`, p);
        this.utimesImpl(resolvedP, mtime);
    }
    async lutimesPromise(p, atime, mtime) {
        return this.lutimesSync(p, atime, mtime);
    }
    lutimesSync(p, atime, mtime) {
        if (this.readOnly)
            throw errors.EROFS(`lutimes '${p}'`);
        const resolvedP = this.resolveFilename(`utimes '${p}'`, p, false);
        this.utimesImpl(resolvedP, mtime);
    }
    utimesImpl(resolvedP, mtime) {
        if (this.listings.has(resolvedP))
            if (!this.entries.has(resolvedP))
                this.hydrateDirectory(resolvedP);
        const entry = this.entries.get(resolvedP);
        if (entry === undefined)
            throw new Error(`Unreachable`);
        const rc = this.libzip.file.setMtime(this.zip, entry, 0, toUnixTimestamp(mtime), 0);
        if (rc === -1) {
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        }
    }
    async mkdirPromise(p, opts) {
        return this.mkdirSync(p, opts);
    }
    mkdirSync(p, { mode = 0o755, recursive = false } = {}) {
        if (recursive)
            return this.mkdirpSync(p, { chmod: mode });
        if (this.readOnly)
            throw errors.EROFS(`mkdir '${p}'`);
        const resolvedP = this.resolveFilename(`mkdir '${p}'`, p);
        if (this.entries.has(resolvedP) || this.listings.has(resolvedP))
            throw errors.EEXIST(`mkdir '${p}'`);
        this.hydrateDirectory(resolvedP);
        this.chmodSync(resolvedP, mode);
        return undefined;
    }
    async rmdirPromise(p, opts) {
        return this.rmdirSync(p, opts);
    }
    rmdirSync(p, { recursive = false } = {}) {
        if (this.readOnly)
            throw errors.EROFS(`rmdir '${p}'`);
        if (recursive) {
            this.removeSync(p);
            return;
        }
        const resolvedP = this.resolveFilename(`rmdir '${p}'`, p);
        const directoryListing = this.listings.get(resolvedP);
        if (!directoryListing)
            throw errors.ENOTDIR(`rmdir '${p}'`);
        if (directoryListing.size > 0)
            throw errors.ENOTEMPTY(`rmdir '${p}'`);
        const index = this.entries.get(resolvedP);
        if (typeof index === `undefined`)
            throw errors.EINVAL(`rmdir '${p}'`);
        this.deleteEntry(p, index);
    }
    hydrateDirectory(resolvedP) {
        const index = this.libzip.dir.add(this.zip, path_1.ppath.relative(path_1.PortablePath.root, resolvedP));
        if (index === -1)
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        this.registerListing(resolvedP);
        this.registerEntry(resolvedP, index);
        return index;
    }
    async linkPromise(existingP, newP) {
        return this.linkSync(existingP, newP);
    }
    linkSync(existingP, newP) {
        // Zip archives don't support hard links:
        // https://stackoverflow.com/questions/8859616/are-hard-links-possible-within-a-zip-archive
        throw errors.EOPNOTSUPP(`link '${existingP}' -> '${newP}'`);
    }
    async symlinkPromise(target, p) {
        return this.symlinkSync(target, p);
    }
    symlinkSync(target, p) {
        if (this.readOnly)
            throw errors.EROFS(`symlink '${target}' -> '${p}'`);
        const resolvedP = this.resolveFilename(`symlink '${target}' -> '${p}'`, p);
        if (this.listings.has(resolvedP))
            throw errors.EISDIR(`symlink '${target}' -> '${p}'`);
        if (this.entries.has(resolvedP))
            throw errors.EEXIST(`symlink '${target}' -> '${p}'`);
        const index = this.setFileSource(resolvedP, target);
        this.registerEntry(resolvedP, index);
        const rc = this.libzip.file.setExternalAttributes(this.zip, index, 0, 0, this.libzip.ZIP_OPSYS_UNIX, (constants_1.S_IFLNK | 0o777) << 16);
        if (rc === -1)
            throw this.makeLibzipError(this.libzip.getError(this.zip));
        this.symlinkCount += 1;
    }
    async readFilePromise(p, encoding) {
        // This is messed up regarding the TS signatures
        if (typeof encoding === `object`)
            // @ts-expect-error
            encoding = encoding ? encoding.encoding : undefined;
        const data = await this.readFileBuffer(p, { asyncDecompress: true });
        return encoding ? data.toString(encoding) : data;
    }
    readFileSync(p, encoding) {
        // This is messed up regarding the TS signatures
        if (typeof encoding === `object`)
            // @ts-expect-error
            encoding = encoding ? encoding.encoding : undefined;
        const data = this.readFileBuffer(p);
        return encoding ? data.toString(encoding) : data;
    }
    readFileBuffer(p, opts = { asyncDecompress: false }) {
        if (typeof p === `number`)
            p = this.fdToPath(p, `read`);
        const resolvedP = this.resolveFilename(`open '${p}'`, p);
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP))
            throw errors.ENOENT(`open '${p}'`);
        // Ensures that the last component is a directory, if the user said so (even if it is we'll throw right after with EISDIR anyway)
        if (p[p.length - 1] === `/` && !this.listings.has(resolvedP))
            throw errors.ENOTDIR(`open '${p}'`);
        if (this.listings.has(resolvedP))
            throw errors.EISDIR(`read`);
        const entry = this.entries.get(resolvedP);
        if (entry === undefined)
            throw new Error(`Unreachable`);
        return this.getFileSource(entry, opts);
    }
    async readdirPromise(p, opts) {
        return this.readdirSync(p, opts);
    }
    readdirSync(p, opts) {
        const resolvedP = this.resolveFilename(`scandir '${p}'`, p);
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP))
            throw errors.ENOENT(`scandir '${p}'`);
        const directoryListing = this.listings.get(resolvedP);
        if (!directoryListing)
            throw errors.ENOTDIR(`scandir '${p}'`);
        const entries = [...directoryListing];
        if (!(opts === null || opts === void 0 ? void 0 : opts.withFileTypes))
            return entries;
        return entries.map(name => {
            return Object.assign(this.statImpl(`lstat`, path_1.ppath.join(p, name)), {
                name,
            });
        });
    }
    async readlinkPromise(p) {
        const entry = this.prepareReadlink(p);
        return (await this.getFileSource(entry, { asyncDecompress: true })).toString();
    }
    readlinkSync(p) {
        const entry = this.prepareReadlink(p);
        return this.getFileSource(entry).toString();
    }
    prepareReadlink(p) {
        const resolvedP = this.resolveFilename(`readlink '${p}'`, p, false);
        if (!this.entries.has(resolvedP) && !this.listings.has(resolvedP))
            throw errors.ENOENT(`readlink '${p}'`);
        // Ensure that the last component is a directory (if it is we'll throw right after with EISDIR anyway)
        if (p[p.length - 1] === `/` && !this.listings.has(resolvedP))
            throw errors.ENOTDIR(`open '${p}'`);
        if (this.listings.has(resolvedP))
            throw errors.EINVAL(`readlink '${p}'`);
        const entry = this.entries.get(resolvedP);
        if (entry === undefined)
            throw new Error(`Unreachable`);
        if (!this.isSymbolicLink(entry))
            throw errors.EINVAL(`readlink '${p}'`);
        return entry;
    }
    async truncatePromise(p, len = 0) {
        const resolvedP = this.resolveFilename(`open '${p}'`, p);
        const index = this.entries.get(resolvedP);
        if (typeof index === `undefined`)
            throw errors.EINVAL(`open '${p}'`);
        const source = await this.getFileSource(index, { asyncDecompress: true });
        const truncated = Buffer.alloc(len, 0x00);
        source.copy(truncated);
        return await this.writeFilePromise(p, truncated);
    }
    truncateSync(p, len = 0) {
        const resolvedP = this.resolveFilename(`open '${p}'`, p);
        const index = this.entries.get(resolvedP);
        if (typeof index === `undefined`)
            throw errors.EINVAL(`open '${p}'`);
        const source = this.getFileSource(index);
        const truncated = Buffer.alloc(len, 0x00);
        source.copy(truncated);
        return this.writeFileSync(p, truncated);
    }
    async ftruncatePromise(fd, len) {
        return this.truncatePromise(this.fdToPath(fd, `ftruncate`), len);
    }
    ftruncateSync(fd, len) {
        return this.truncateSync(this.fdToPath(fd, `ftruncateSync`), len);
    }
    watch(p, a, b) {
        let persistent;
        switch (typeof a) {
            case `function`:
            case `string`:
            case `undefined`:
                {
                    persistent = true;
                }
                break;
            default:
                {
                    ({ persistent = true } = a);
                }
                break;
        }
        if (!persistent)
            return { on: () => { }, close: () => { } };
        const interval = setInterval(() => { }, 24 * 60 * 60 * 1000);
        return { on: () => { }, close: () => {
                clearInterval(interval);
            } };
    }
    watchFile(p, a, b) {
        const resolvedP = path_1.ppath.resolve(path_1.PortablePath.root, p);
        return (0, watchFile_1.watchFile)(this, resolvedP, a, b);
    }
    unwatchFile(p, cb) {
        const resolvedP = path_1.ppath.resolve(path_1.PortablePath.root, p);
        return (0, watchFile_1.unwatchFile)(this, resolvedP, cb);
    }
}
exports.ZipFS = ZipFS;
