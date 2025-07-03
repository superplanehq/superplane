"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.JailFS = void 0;
const NodeFS_1 = require("./NodeFS");
const ProxiedFS_1 = require("./ProxiedFS");
const path_1 = require("./path");
const JAIL_ROOT = path_1.PortablePath.root;
class JailFS extends ProxiedFS_1.ProxiedFS {
    constructor(target, { baseFs = new NodeFS_1.NodeFS() } = {}) {
        super(path_1.ppath);
        this.target = this.pathUtils.resolve(path_1.PortablePath.root, target);
        this.baseFs = baseFs;
    }
    getRealPath() {
        return this.pathUtils.resolve(this.baseFs.getRealPath(), this.pathUtils.relative(path_1.PortablePath.root, this.target));
    }
    getTarget() {
        return this.target;
    }
    getBaseFs() {
        return this.baseFs;
    }
    mapToBase(p) {
        const normalized = this.pathUtils.normalize(p);
        if (this.pathUtils.isAbsolute(p))
            return this.pathUtils.resolve(this.target, this.pathUtils.relative(JAIL_ROOT, p));
        if (normalized.match(/^\.\.\/?/))
            throw new Error(`Resolving this path (${p}) would escape the jail`);
        return this.pathUtils.resolve(this.target, p);
    }
    mapFromBase(p) {
        return this.pathUtils.resolve(JAIL_ROOT, this.pathUtils.relative(this.target, p));
    }
}
exports.JailFS = JailFS;
