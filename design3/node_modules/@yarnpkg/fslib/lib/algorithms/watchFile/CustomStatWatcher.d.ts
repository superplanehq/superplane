/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
import { EventEmitter } from 'events';
import { BigIntStats, Stats } from 'fs';
import { StatWatcher, WatchFileCallback, WatchFileOptions, FakeFS } from '../../FakeFS';
import { Path } from '../../path';
export declare enum Event {
    Change = "change",
    Stop = "stop"
}
export declare enum Status {
    Ready = "ready",
    Running = "running",
    Stopped = "stopped"
}
export declare function assertStatus<T extends Status>(current: Status, expected: T): asserts current is T;
export type ListenerOptions = Omit<Required<WatchFileOptions>, 'bigint'>;
export type CustomStatWatcherOptions = {
    bigint?: boolean;
};
export declare class CustomStatWatcher<P extends Path> extends EventEmitter implements StatWatcher {
    readonly fakeFs: FakeFS<P>;
    readonly path: P;
    readonly bigint: boolean;
    private status;
    private changeListeners;
    private lastStats;
    private startTimeout;
    static create<P extends Path>(fakeFs: FakeFS<P>, path: P, opts?: CustomStatWatcherOptions): CustomStatWatcher<P>;
    private constructor();
    start(): void;
    stop(): void;
    stat(): Stats | BigIntStats;
    /**
     * Creates an interval whose callback compares the current stats with the previous stats and notifies all listeners in case of changes.
     *
     * @param opts.persistent Decides whether the interval should be immediately unref-ed.
     */
    makeInterval(opts: ListenerOptions): NodeJS.Timeout;
    /**
     * Registers a listener and assigns it an interval.
     */
    registerChangeListener(listener: WatchFileCallback, opts: ListenerOptions): void;
    /**
     * Unregisters the listener and clears the assigned interval.
     */
    unregisterChangeListener(listener: WatchFileCallback): void;
    /**
     * Unregisters all listeners and clears all assigned intervals.
     */
    unregisterAllChangeListeners(): void;
    hasChangeListeners(): boolean;
    /**
     * Refs all stored intervals.
     */
    ref(): this;
    /**
     * Unrefs all stored intervals.
     */
    unref(): this;
}
