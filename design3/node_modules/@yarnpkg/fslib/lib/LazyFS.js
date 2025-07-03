"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.LazyFS = void 0;
const ProxiedFS_1 = require("./ProxiedFS");
class LazyFS extends ProxiedFS_1.ProxiedFS {
    constructor(factory, pathUtils) {
        super(pathUtils);
        this.instance = null;
        this.factory = factory;
    }
    get baseFs() {
        if (!this.instance)
            this.instance = this.factory();
        return this.instance;
    }
    set baseFs(value) {
        this.instance = value;
    }
    mapFromBase(p) {
        return p;
    }
    mapToBase(p) {
        return p;
    }
}
exports.LazyFS = LazyFS;
