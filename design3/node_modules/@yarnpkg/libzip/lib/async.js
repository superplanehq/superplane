"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getLibzipPromise = exports.getLibzipSync = void 0;
const tslib_1 = require("tslib");
const libzipAsync_1 = tslib_1.__importDefault(require("./libzipAsync"));
const makeInterface_1 = require("./makeInterface");
let promise = null;
function getLibzipSync() {
    throw new Error(`Cannot use getLibzipSync when using the async version of the libzip`);
}
exports.getLibzipSync = getLibzipSync;
async function getLibzipPromise() {
    if (promise === null)
        promise = (0, libzipAsync_1.default)().then(libzip => (0, makeInterface_1.makeInterface)(libzip));
    return promise;
}
exports.getLibzipPromise = getLibzipPromise;
