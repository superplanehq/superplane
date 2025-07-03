"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.CwdFS = void 0;
const NodeFS_1 = require("./NodeFS");
const ProxiedFS_1 = require("./ProxiedFS");
const path_1 = require("./path");
class CwdFS extends ProxiedFS_1.ProxiedFS {
    constructor(target, { baseFs = new NodeFS_1.NodeFS() } = {}) {
        super(path_1.ppath);
        this.target = this.pathUtils.normalize(target);
        this.baseFs = baseFs;
    }
    getRealPath() {
        return this.pathUtils.resolve(this.baseFs.getRealPath(), this.target);
    }
    resolve(p) {
        if (this.pathUtils.isAbsolute(p)) {
            return path_1.ppath.normalize(p);
        }
        else {
            return this.baseFs.resolve(path_1.ppath.join(this.target, p));
        }
    }
    mapFromBase(path) {
        return path;
    }
    mapToBase(path) {
        if (this.pathUtils.isAbsolute(path)) {
            return path;
        }
        else {
            return this.pathUtils.join(this.target, path);
        }
    }
}
exports.CwdFS = CwdFS;
