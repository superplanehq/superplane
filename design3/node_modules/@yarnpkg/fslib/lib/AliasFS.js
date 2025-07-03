"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AliasFS = void 0;
const ProxiedFS_1 = require("./ProxiedFS");
class AliasFS extends ProxiedFS_1.ProxiedFS {
    constructor(target, { baseFs, pathUtils }) {
        super(pathUtils);
        this.target = target;
        this.baseFs = baseFs;
    }
    getRealPath() {
        return this.target;
    }
    getBaseFs() {
        return this.baseFs;
    }
    mapFromBase(p) {
        return p;
    }
    mapToBase(p) {
        return p;
    }
}
exports.AliasFS = AliasFS;
