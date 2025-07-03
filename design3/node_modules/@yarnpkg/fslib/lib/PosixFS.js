"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.PosixFS = void 0;
const ProxiedFS_1 = require("./ProxiedFS");
const path_1 = require("./path");
class PosixFS extends ProxiedFS_1.ProxiedFS {
    constructor(baseFs) {
        super(path_1.npath);
        this.baseFs = baseFs;
    }
    mapFromBase(path) {
        return path_1.npath.fromPortablePath(path);
    }
    mapToBase(path) {
        return path_1.npath.toPortablePath(path);
    }
}
exports.PosixFS = PosixFS;
