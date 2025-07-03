"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.unwatchAllFiles = exports.unwatchFile = exports.watchFile = void 0;
const CustomStatWatcher_1 = require("./watchFile/CustomStatWatcher");
const statWatchersByFakeFS = new WeakMap();
function watchFile(fakeFs, path, a, b) {
    let bigint;
    let persistent;
    let interval;
    let listener;
    switch (typeof a) {
        case `function`:
            {
                bigint = false;
                persistent = true;
                interval = 5007;
                listener = a;
            }
            break;
        default:
            {
                ({
                    bigint = false,
                    persistent = true,
                    interval = 5007,
                } = a);
                listener = b;
            }
            break;
    }
    let statWatchers = statWatchersByFakeFS.get(fakeFs);
    if (typeof statWatchers === `undefined`)
        statWatchersByFakeFS.set(fakeFs, statWatchers = new Map());
    let statWatcher = statWatchers.get(path);
    if (typeof statWatcher === `undefined`) {
        statWatcher = CustomStatWatcher_1.CustomStatWatcher.create(fakeFs, path, { bigint });
        statWatchers.set(path, statWatcher);
    }
    statWatcher.registerChangeListener(listener, { persistent, interval });
    return statWatcher;
}
exports.watchFile = watchFile;
function unwatchFile(fakeFs, path, cb) {
    const statWatchers = statWatchersByFakeFS.get(fakeFs);
    if (typeof statWatchers === `undefined`)
        return;
    const statWatcher = statWatchers.get(path);
    if (typeof statWatcher === `undefined`)
        return;
    if (typeof cb === `undefined`)
        statWatcher.unregisterAllChangeListeners();
    else
        statWatcher.unregisterChangeListener(cb);
    if (!statWatcher.hasChangeListeners()) {
        statWatcher.stop();
        statWatchers.delete(path);
    }
}
exports.unwatchFile = unwatchFile;
function unwatchAllFiles(fakeFs) {
    const statWatchers = statWatchersByFakeFS.get(fakeFs);
    if (typeof statWatchers === `undefined`)
        return;
    for (const path of statWatchers.keys()) {
        unwatchFile(fakeFs, path);
    }
}
exports.unwatchAllFiles = unwatchAllFiles;
