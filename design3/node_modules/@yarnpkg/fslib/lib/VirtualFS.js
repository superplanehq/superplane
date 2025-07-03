"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.VirtualFS = void 0;
const NodeFS_1 = require("./NodeFS");
const ProxiedFS_1 = require("./ProxiedFS");
const path_1 = require("./path");
const NUMBER_REGEXP = /^[0-9]+$/;
// $0: full path
// $1: virtual folder
// $2: virtual segment
// $3: hash
// $4: depth
// $5: subpath
const VIRTUAL_REGEXP = /^(\/(?:[^/]+\/)*?(?:\$\$virtual|__virtual__))((?:\/((?:[^/]+-)?[a-f0-9]+)(?:\/([^/]+))?)?((?:\/.*)?))$/;
const VALID_COMPONENT = /^([^/]+-)?[a-f0-9]+$/;
class VirtualFS extends ProxiedFS_1.ProxiedFS {
    static makeVirtualPath(base, component, to) {
        if (path_1.ppath.basename(base) !== `__virtual__`)
            throw new Error(`Assertion failed: Virtual folders must be named "__virtual__"`);
        if (!path_1.ppath.basename(component).match(VALID_COMPONENT))
            throw new Error(`Assertion failed: Virtual components must be ended by an hexadecimal hash`);
        // Obtains the relative distance between the virtual path and its actual target
        const target = path_1.ppath.relative(path_1.ppath.dirname(base), to);
        const segments = target.split(`/`);
        // Counts how many levels we need to go back to start applying the rest of the path
        let depth = 0;
        while (depth < segments.length && segments[depth] === `..`)
            depth += 1;
        const finalSegments = segments.slice(depth);
        const fullVirtualPath = path_1.ppath.join(base, component, String(depth), ...finalSegments);
        return fullVirtualPath;
    }
    static resolveVirtual(p) {
        const match = p.match(VIRTUAL_REGEXP);
        if (!match || (!match[3] && match[5]))
            return p;
        const target = path_1.ppath.dirname(match[1]);
        if (!match[3] || !match[4])
            return target;
        const isnum = NUMBER_REGEXP.test(match[4]);
        if (!isnum)
            return p;
        const depth = Number(match[4]);
        const backstep = `../`.repeat(depth);
        const subpath = (match[5] || `.`);
        return VirtualFS.resolveVirtual(path_1.ppath.join(target, backstep, subpath));
    }
    constructor({ baseFs = new NodeFS_1.NodeFS() } = {}) {
        super(path_1.ppath);
        this.baseFs = baseFs;
    }
    getExtractHint(hints) {
        return this.baseFs.getExtractHint(hints);
    }
    getRealPath() {
        return this.baseFs.getRealPath();
    }
    realpathSync(p) {
        const match = p.match(VIRTUAL_REGEXP);
        if (!match)
            return this.baseFs.realpathSync(p);
        if (!match[5])
            return p;
        const realpath = this.baseFs.realpathSync(this.mapToBase(p));
        return VirtualFS.makeVirtualPath(match[1], match[3], realpath);
    }
    async realpathPromise(p) {
        const match = p.match(VIRTUAL_REGEXP);
        if (!match)
            return await this.baseFs.realpathPromise(p);
        if (!match[5])
            return p;
        const realpath = await this.baseFs.realpathPromise(this.mapToBase(p));
        return VirtualFS.makeVirtualPath(match[1], match[3], realpath);
    }
    mapToBase(p) {
        if (p === ``)
            return p;
        if (this.pathUtils.isAbsolute(p))
            return VirtualFS.resolveVirtual(p);
        const resolvedRoot = VirtualFS.resolveVirtual(this.baseFs.resolve(path_1.PortablePath.dot));
        const resolvedP = VirtualFS.resolveVirtual(this.baseFs.resolve(p));
        return path_1.ppath.relative(resolvedRoot, resolvedP) || path_1.PortablePath.dot;
    }
    mapFromBase(p) {
        return p;
    }
}
exports.VirtualFS = VirtualFS;
