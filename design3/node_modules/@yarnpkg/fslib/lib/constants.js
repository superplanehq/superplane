"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.SAFE_TIME = exports.S_IFLNK = exports.S_IFREG = exports.S_IFDIR = exports.S_IFMT = void 0;
exports.S_IFMT = 0o170000;
exports.S_IFDIR = 0o040000;
exports.S_IFREG = 0o100000;
exports.S_IFLNK = 0o120000;
/**
 * Unix timestamp for `1984-06-22T21:50:00.000Z`
 *
 * It needs to be after 1980-01-01 because that's what Zip supports, and it
 * needs to have a slight offset to account for different timezones (because
 * zip assumes that all times are local to whoever writes the file, which is
 * really silly).
 */
exports.SAFE_TIME = 456789000;
