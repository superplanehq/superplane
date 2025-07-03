"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.CustomStatWatcher = exports.assertStatus = exports.Status = exports.Event = void 0;
const tslib_1 = require("tslib");
const events_1 = require("events");
const statUtils = tslib_1.__importStar(require("../../statUtils"));
var Event;
(function (Event) {
    Event["Change"] = "change";
    Event["Stop"] = "stop";
})(Event = exports.Event || (exports.Event = {}));
var Status;
(function (Status) {
    Status["Ready"] = "ready";
    Status["Running"] = "running";
    Status["Stopped"] = "stopped";
})(Status = exports.Status || (exports.Status = {}));
function assertStatus(current, expected) {
    if (current !== expected) {
        throw new Error(`Invalid StatWatcher status: expected '${expected}', got '${current}'`);
    }
}
exports.assertStatus = assertStatus;
class CustomStatWatcher extends events_1.EventEmitter {
    static create(fakeFs, path, opts) {
        const statWatcher = new CustomStatWatcher(fakeFs, path, opts);
        statWatcher.start();
        return statWatcher;
    }
    constructor(fakeFs, path, { bigint = false } = {}) {
        super();
        this.status = Status.Ready;
        this.changeListeners = new Map();
        this.startTimeout = null;
        this.fakeFs = fakeFs;
        this.path = path;
        this.bigint = bigint;
        this.lastStats = this.stat();
    }
    start() {
        assertStatus(this.status, Status.Ready);
        this.status = Status.Running;
        // Node allows other listeners to be registered up to 3 milliseconds
        // after the watcher has been started, so that's what we're doing too
        this.startTimeout = setTimeout(() => {
            this.startTimeout = null;
            // Per the Node FS docs:
            // "When an fs.watchFile operation results in an ENOENT error,
            // it will invoke the listener once, with all the fields zeroed
            // (or, for dates, the Unix Epoch)."
            if (!this.fakeFs.existsSync(this.path)) {
                this.emit(Event.Change, this.lastStats, this.lastStats);
            }
        }, 3);
    }
    stop() {
        assertStatus(this.status, Status.Running);
        this.status = Status.Stopped;
        if (this.startTimeout !== null) {
            clearTimeout(this.startTimeout);
            this.startTimeout = null;
        }
        this.emit(Event.Stop);
    }
    stat() {
        try {
            return this.fakeFs.statSync(this.path, { bigint: this.bigint });
        }
        catch (error) {
            // From observation, all errors seem to be mostly ignored by Node.
            // Checked with ENOENT, ENOTDIR, EPERM
            const statInstance = this.bigint
                ? new statUtils.BigIntStatsEntry()
                : new statUtils.StatEntry();
            return statUtils.clearStats(statInstance);
        }
    }
    /**
     * Creates an interval whose callback compares the current stats with the previous stats and notifies all listeners in case of changes.
     *
     * @param opts.persistent Decides whether the interval should be immediately unref-ed.
     */
    makeInterval(opts) {
        const interval = setInterval(() => {
            const currentStats = this.stat();
            const previousStats = this.lastStats;
            if (statUtils.areStatsEqual(currentStats, previousStats))
                return;
            this.lastStats = currentStats;
            this.emit(Event.Change, currentStats, previousStats);
        }, opts.interval);
        return opts.persistent ? interval : interval.unref();
    }
    /**
     * Registers a listener and assigns it an interval.
     */
    registerChangeListener(listener, opts) {
        this.addListener(Event.Change, listener);
        this.changeListeners.set(listener, this.makeInterval(opts));
    }
    /**
     * Unregisters the listener and clears the assigned interval.
     */
    unregisterChangeListener(listener) {
        this.removeListener(Event.Change, listener);
        const interval = this.changeListeners.get(listener);
        if (typeof interval !== `undefined`)
            clearInterval(interval);
        this.changeListeners.delete(listener);
    }
    /**
     * Unregisters all listeners and clears all assigned intervals.
     */
    unregisterAllChangeListeners() {
        for (const listener of this.changeListeners.keys()) {
            this.unregisterChangeListener(listener);
        }
    }
    hasChangeListeners() {
        return this.changeListeners.size > 0;
    }
    /**
     * Refs all stored intervals.
     */
    ref() {
        for (const interval of this.changeListeners.values())
            interval.ref();
        return this;
    }
    /**
     * Unrefs all stored intervals.
     */
    unref() {
        for (const interval of this.changeListeners.values())
            interval.unref();
        return this;
    }
}
exports.CustomStatWatcher = CustomStatWatcher;
