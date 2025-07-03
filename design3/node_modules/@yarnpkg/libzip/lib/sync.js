"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getLibzipPromise = exports.getLibzipSync = void 0;
const tslib_1 = require("tslib");
const libzipSync_1 = tslib_1.__importDefault(require("./libzipSync"));
const makeInterface_1 = require("./makeInterface");
let mod = null;
function getLibzipSync() {
    if (mod === null)
        mod = (0, makeInterface_1.makeInterface)((0, libzipSync_1.default)());
    return mod;
}
exports.getLibzipSync = getLibzipSync;
async function getLibzipPromise() {
    return getLibzipSync();
}
exports.getLibzipPromise = getLibzipPromise;
