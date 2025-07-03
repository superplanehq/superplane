"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeLineEndings = exports.BasePortableFakeFS = exports.FakeFS = void 0;
const os_1 = require("os");
const copyPromise_1 = require("./algorithms/copyPromise");
const path_1 = require("./path");
class FakeFS {
    constructor(pathUtils) {
        this.pathUtils = pathUtils;
    }
    async *genTraversePromise(init, { stableSort = false } = {}) {
        const stack = [init];
        while (stack.length > 0) {
            const p = stack.shift();
            const entry = await this.lstatPromise(p);
            if (entry.isDirectory()) {
                const entries = await this.readdirPromise(p);
                if (stableSort) {
                    for (const entry of entries.sort()) {
                        stack.push(this.pathUtils.join(p, entry));
                    }
                }
                else {
                    throw new Error(`Not supported`);
                }
            }
            else {
                yield p;
            }
        }
    }
    async removePromise(p, { recursive = true, maxRetries = 5 } = {}) {
        let stat;
        try {
            stat = await this.lstatPromise(p);
        }
        catch (error) {
            if (error.code === `ENOENT`) {
                return;
            }
            else {
                throw error;
            }
        }
        if (stat.isDirectory()) {
            if (recursive) {
                const entries = await this.readdirPromise(p);
                await Promise.all(entries.map(entry => {
                    return this.removePromise(this.pathUtils.resolve(p, entry));
                }));
            }
            // 5 gives 1s worth of retries at worst
            for (let t = 0; t <= maxRetries; t++) {
                try {
                    await this.rmdirPromise(p);
                    break;
                }
                catch (error) {
                    if (error.code !== `EBUSY` && error.code !== `ENOTEMPTY`) {
                        throw error;
                    }
                    else if (t < maxRetries) {
                        await new Promise(resolve => setTimeout(resolve, t * 100));
                    }
                }
            }
        }
        else {
            await this.unlinkPromise(p);
        }
    }
    removeSync(p, { recursive = true } = {}) {
        let stat;
        try {
            stat = this.lstatSync(p);
        }
        catch (error) {
            if (error.code === `ENOENT`) {
                return;
            }
            else {
                throw error;
            }
        }
        if (stat.isDirectory()) {
            if (recursive)
                for (const entry of this.readdirSync(p))
                    this.removeSync(this.pathUtils.resolve(p, entry));
            this.rmdirSync(p);
        }
        else {
            this.unlinkSync(p);
        }
    }
    async mkdirpPromise(p, { chmod, utimes } = {}) {
        p = this.resolve(p);
        if (p === this.pathUtils.dirname(p))
            return undefined;
        const parts = p.split(this.pathUtils.sep);
        let createdDirectory;
        for (let u = 2; u <= parts.length; ++u) {
            const subPath = parts.slice(0, u).join(this.pathUtils.sep);
            if (!this.existsSync(subPath)) {
                try {
                    await this.mkdirPromise(subPath);
                }
                catch (error) {
                    if (error.code === `EEXIST`) {
                        continue;
                    }
                    else {
                        throw error;
                    }
                }
                createdDirectory !== null && createdDirectory !== void 0 ? createdDirectory : (createdDirectory = subPath);
                if (chmod != null)
                    await this.chmodPromise(subPath, chmod);
                if (utimes != null) {
                    await this.utimesPromise(subPath, utimes[0], utimes[1]);
                }
                else {
                    const parentStat = await this.statPromise(this.pathUtils.dirname(subPath));
                    await this.utimesPromise(subPath, parentStat.atime, parentStat.mtime);
                }
            }
        }
        return createdDirectory;
    }
    mkdirpSync(p, { chmod, utimes } = {}) {
        p = this.resolve(p);
        if (p === this.pathUtils.dirname(p))
            return undefined;
        const parts = p.split(this.pathUtils.sep);
        let createdDirectory;
        for (let u = 2; u <= parts.length; ++u) {
            const subPath = parts.slice(0, u).join(this.pathUtils.sep);
            if (!this.existsSync(subPath)) {
                try {
                    this.mkdirSync(subPath);
                }
                catch (error) {
                    if (error.code === `EEXIST`) {
                        continue;
                    }
                    else {
                        throw error;
                    }
                }
                createdDirectory !== null && createdDirectory !== void 0 ? createdDirectory : (createdDirectory = subPath);
                if (chmod != null)
                    this.chmodSync(subPath, chmod);
                if (utimes != null) {
                    this.utimesSync(subPath, utimes[0], utimes[1]);
                }
                else {
                    const parentStat = this.statSync(this.pathUtils.dirname(subPath));
                    this.utimesSync(subPath, parentStat.atime, parentStat.mtime);
                }
            }
        }
        return createdDirectory;
    }
    async copyPromise(destination, source, { baseFs = this, overwrite = true, stableSort = false, stableTime = false, linkStrategy = null } = {}) {
        return await (0, copyPromise_1.copyPromise)(this, destination, baseFs, source, { overwrite, stableSort, stableTime, linkStrategy });
    }
    copySync(destination, source, { baseFs = this, overwrite = true } = {}) {
        const stat = baseFs.lstatSync(source);
        const exists = this.existsSync(destination);
        if (stat.isDirectory()) {
            this.mkdirpSync(destination);
            const directoryListing = baseFs.readdirSync(source);
            for (const entry of directoryListing) {
                this.copySync(this.pathUtils.join(destination, entry), baseFs.pathUtils.join(source, entry), { baseFs, overwrite });
            }
        }
        else if (stat.isFile()) {
            if (!exists || overwrite) {
                if (exists)
                    this.removeSync(destination);
                const content = baseFs.readFileSync(source);
                this.writeFileSync(destination, content);
            }
        }
        else if (stat.isSymbolicLink()) {
            if (!exists || overwrite) {
                if (exists)
                    this.removeSync(destination);
                const target = baseFs.readlinkSync(source);
                this.symlinkSync((0, path_1.convertPath)(this.pathUtils, target), destination);
            }
        }
        else {
            throw new Error(`Unsupported file type (file: ${source}, mode: 0o${stat.mode.toString(8).padStart(6, `0`)})`);
        }
        const mode = stat.mode & 0o777;
        this.chmodSync(destination, mode);
    }
    async changeFilePromise(p, content, opts = {}) {
        if (Buffer.isBuffer(content)) {
            return this.changeFileBufferPromise(p, content, opts);
        }
        else {
            return this.changeFileTextPromise(p, content, opts);
        }
    }
    async changeFileBufferPromise(p, content, { mode } = {}) {
        let current = Buffer.alloc(0);
        try {
            current = await this.readFilePromise(p);
        }
        catch (error) {
            // ignore errors, no big deal
        }
        if (Buffer.compare(current, content) === 0)
            return;
        await this.writeFilePromise(p, content, { mode });
    }
    async changeFileTextPromise(p, content, { automaticNewlines, mode } = {}) {
        let current = ``;
        try {
            current = await this.readFilePromise(p, `utf8`);
        }
        catch (error) {
            // ignore errors, no big deal
        }
        const normalizedContent = automaticNewlines
            ? normalizeLineEndings(current, content)
            : content;
        if (current === normalizedContent)
            return;
        await this.writeFilePromise(p, normalizedContent, { mode });
    }
    changeFileSync(p, content, opts = {}) {
        if (Buffer.isBuffer(content)) {
            return this.changeFileBufferSync(p, content, opts);
        }
        else {
            return this.changeFileTextSync(p, content, opts);
        }
    }
    changeFileBufferSync(p, content, { mode } = {}) {
        let current = Buffer.alloc(0);
        try {
            current = this.readFileSync(p);
        }
        catch (error) {
            // ignore errors, no big deal
        }
        if (Buffer.compare(current, content) === 0)
            return;
        this.writeFileSync(p, content, { mode });
    }
    changeFileTextSync(p, content, { automaticNewlines = false, mode } = {}) {
        let current = ``;
        try {
            current = this.readFileSync(p, `utf8`);
        }
        catch (error) {
            // ignore errors, no big deal
        }
        const normalizedContent = automaticNewlines
            ? normalizeLineEndings(current, content)
            : content;
        if (current === normalizedContent)
            return;
        this.writeFileSync(p, normalizedContent, { mode });
    }
    async movePromise(fromP, toP) {
        try {
            await this.renamePromise(fromP, toP);
        }
        catch (error) {
            if (error.code === `EXDEV`) {
                await this.copyPromise(toP, fromP);
                await this.removePromise(fromP);
            }
            else {
                throw error;
            }
        }
    }
    moveSync(fromP, toP) {
        try {
            this.renameSync(fromP, toP);
        }
        catch (error) {
            if (error.code === `EXDEV`) {
                this.copySync(toP, fromP);
                this.removeSync(fromP);
            }
            else {
                throw error;
            }
        }
    }
    async lockPromise(affectedPath, callback) {
        const lockPath = `${affectedPath}.flock`;
        const interval = 1000 / 60;
        const startTime = Date.now();
        let fd = null;
        // Even when we detect that a lock file exists, we still look inside to see
        // whether the pid that created it is still alive. It's not foolproof
        // (there are false positive), but there are no false negative and that's
        // all that matters in 99% of the cases.
        const isAlive = async () => {
            let pid;
            try {
                ([pid] = await this.readJsonPromise(lockPath));
            }
            catch (error) {
                // If we can't read the file repeatedly, we assume the process was
                // aborted before even writing finishing writing the payload.
                return Date.now() - startTime < 500;
            }
            try {
                // "As a special case, a signal of 0 can be used to test for the
                // existence of a process" - so we check whether it's alive.
                process.kill(pid, 0);
                return true;
            }
            catch (error) {
                return false;
            }
        };
        while (fd === null) {
            try {
                fd = await this.openPromise(lockPath, `wx`);
            }
            catch (error) {
                if (error.code === `EEXIST`) {
                    if (!await isAlive()) {
                        try {
                            await this.unlinkPromise(lockPath);
                            continue;
                        }
                        catch (error) {
                            // No big deal if we can't remove it. Just fallback to wait for
                            // it to be eventually released by its owner.
                        }
                    }
                    if (Date.now() - startTime < 60 * 1000) {
                        await new Promise(resolve => setTimeout(resolve, interval));
                    }
                    else {
                        throw new Error(`Couldn't acquire a lock in a reasonable time (via ${lockPath})`);
                    }
                }
                else {
                    throw error;
                }
            }
        }
        await this.writePromise(fd, JSON.stringify([process.pid]));
        try {
            return await callback();
        }
        finally {
            try {
                // closePromise needs to come before unlinkPromise otherwise another process can attempt
                // to get the file handle after the unlink but before close resuling in
                // EPERM: operation not permitted, open
                await this.closePromise(fd);
                await this.unlinkPromise(lockPath);
            }
            catch (error) {
                // noop
            }
        }
    }
    async readJsonPromise(p) {
        const content = await this.readFilePromise(p, `utf8`);
        try {
            return JSON.parse(content);
        }
        catch (error) {
            error.message += ` (in ${p})`;
            throw error;
        }
    }
    readJsonSync(p) {
        const content = this.readFileSync(p, `utf8`);
        try {
            return JSON.parse(content);
        }
        catch (error) {
            error.message += ` (in ${p})`;
            throw error;
        }
    }
    async writeJsonPromise(p, data) {
        return await this.writeFilePromise(p, `${JSON.stringify(data, null, 2)}\n`);
    }
    writeJsonSync(p, data) {
        return this.writeFileSync(p, `${JSON.stringify(data, null, 2)}\n`);
    }
    async preserveTimePromise(p, cb) {
        const stat = await this.lstatPromise(p);
        const result = await cb();
        if (typeof result !== `undefined`)
            p = result;
        if (this.lutimesPromise) {
            await this.lutimesPromise(p, stat.atime, stat.mtime);
        }
        else if (!stat.isSymbolicLink()) {
            await this.utimesPromise(p, stat.atime, stat.mtime);
        }
    }
    async preserveTimeSync(p, cb) {
        const stat = this.lstatSync(p);
        const result = cb();
        if (typeof result !== `undefined`)
            p = result;
        if (this.lutimesSync) {
            this.lutimesSync(p, stat.atime, stat.mtime);
        }
        else if (!stat.isSymbolicLink()) {
            this.utimesSync(p, stat.atime, stat.mtime);
        }
    }
}
exports.FakeFS = FakeFS;
class BasePortableFakeFS extends FakeFS {
    constructor() {
        super(path_1.ppath);
    }
}
exports.BasePortableFakeFS = BasePortableFakeFS;
function getEndOfLine(content) {
    const matches = content.match(/\r?\n/g);
    if (matches === null)
        return os_1.EOL;
    const crlf = matches.filter(nl => nl === `\r\n`).length;
    const lf = matches.length - crlf;
    return crlf > lf ? `\r\n` : `\n`;
}
function normalizeLineEndings(originalContent, newContent) {
    return newContent.replace(/\r?\n/g, getEndOfLine(originalContent));
}
exports.normalizeLineEndings = normalizeLineEndings;
