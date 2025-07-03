/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
import { EventEmitter } from 'events';
import { Dirent as NodeDirent, ReadStream } from 'fs';
import { Stats as NodeStats, WriteStream } from 'fs';
import { NoParamCallback, BigIntStats as NodeBigIntStats } from 'fs';
import { LinkStrategy } from './algorithms/copyPromise';
import { FSPath, Path, PortablePath, PathUtils, Filename } from './path';
export type Stats = NodeStats & {
    crc?: number;
};
export type BigIntStats = NodeBigIntStats & {
    crc?: number;
};
export type Dirent = Exclude<NodeDirent, 'name'> & {
    name: Filename;
};
export type Dir<P extends Path> = {
    readonly path: P;
    [Symbol.asyncIterator](): AsyncIterableIterator<Dirent>;
    close(): Promise<void>;
    close(cb: NoParamCallback): void;
    closeSync(): void;
    read(): Promise<Dirent | null>;
    read(cb: (err: NodeJS.ErrnoException | null, dirent: Dirent | null) => void): void;
    readSync(): Dirent | null;
};
export type OpendirOptions = Partial<{
    bufferSize: number;
}>;
export type CreateReadStreamOptions = Partial<{
    encoding: string;
    fd: number;
}>;
export type CreateWriteStreamOptions = Partial<{
    encoding: string;
    fd: number;
    flags: 'a';
}>;
export type MkdirOptions = Partial<{
    recursive: boolean;
    mode: number;
}>;
export type RmdirOptions = Partial<{
    maxRetries: number;
    recursive: boolean;
    retryDelay: number;
}>;
export type WriteFileOptions = Partial<{
    encoding: string;
    mode: number;
    flag: string;
}> | string;
export type WatchOptions = Partial<{
    persistent: boolean;
    recursive: boolean;
    encoding: string;
}> | string;
export type WatchFileOptions = Partial<{
    bigint: boolean;
    persistent: boolean;
    interval: number;
}>;
export type ChangeFileOptions = Partial<{
    automaticNewlines: boolean;
    mode: number;
}>;
export type WatchCallback = (eventType: string, filename: string) => void;
export type Watcher = {
    on: any;
    close: () => void;
};
export type WatchFileCallback = (current: Stats, previous: Stats) => void;
export type StatWatcher = EventEmitter & {
    ref?: () => StatWatcher;
    unref?: () => StatWatcher;
};
export type ExtractHintOptions = {
    relevantExtensions: Set<string>;
};
export type SymlinkType = 'file' | 'dir' | 'junction';
export interface StatOptions {
    bigint?: boolean | undefined;
}
export interface StatSyncOptions extends StatOptions {
    throwIfNoEntry?: boolean | undefined;
}
export declare abstract class FakeFS<P extends Path> {
    readonly pathUtils: PathUtils<P>;
    protected constructor(pathUtils: PathUtils<P>);
    /**
     * @deprecated: Moved to jsInstallUtils
     */
    abstract getExtractHint(hints: ExtractHintOptions): boolean;
    abstract getRealPath(): P;
    abstract resolve(p: P): P;
    abstract opendirPromise(p: P, opts?: OpendirOptions): Promise<Dir<P>>;
    abstract opendirSync(p: P, opts?: OpendirOptions): Dir<P>;
    abstract openPromise(p: P, flags: string, mode?: number): Promise<number>;
    abstract openSync(p: P, flags: string, mode?: number): number;
    abstract readPromise(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number | null): Promise<number>;
    abstract readSync(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number | null): number;
    abstract writePromise(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number): Promise<number>;
    abstract writePromise(fd: number, buffer: string, position?: number): Promise<number>;
    abstract writeSync(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number): number;
    abstract writeSync(fd: number, buffer: string, position?: number): number;
    abstract closePromise(fd: number): Promise<void>;
    abstract closeSync(fd: number): void;
    abstract createWriteStream(p: P | null, opts?: CreateWriteStreamOptions): WriteStream;
    abstract createReadStream(p: P | null, opts?: CreateReadStreamOptions): ReadStream;
    abstract realpathPromise(p: P): Promise<P>;
    abstract realpathSync(p: P): P;
    abstract readdirPromise(p: P): Promise<Array<Filename>>;
    abstract readdirPromise(p: P, opts: {
        withFileTypes: false;
    } | null): Promise<Array<Filename>>;
    abstract readdirPromise(p: P, opts: {
        withFileTypes: true;
    }): Promise<Array<Dirent>>;
    abstract readdirPromise(p: P, opts: {
        withFileTypes: boolean;
    }): Promise<Array<Filename> | Array<Dirent>>;
    abstract readdirSync(p: P): Array<Filename>;
    abstract readdirSync(p: P, opts: {
        withFileTypes: false;
    } | null): Array<Filename>;
    abstract readdirSync(p: P, opts: {
        withFileTypes: true;
    }): Array<Dirent>;
    abstract readdirSync(p: P, opts: {
        withFileTypes: boolean;
    }): Array<Filename> | Array<Dirent>;
    abstract existsPromise(p: P): Promise<boolean>;
    abstract existsSync(p: P): boolean;
    abstract accessPromise(p: P, mode?: number): Promise<void>;
    abstract accessSync(p: P, mode?: number): void;
    abstract statPromise(p: P): Promise<Stats>;
    abstract statPromise(p: P, opts: (StatOptions & {
        bigint?: false | undefined;
    }) | undefined): Promise<Stats>;
    abstract statPromise(p: P, opts: StatOptions & {
        bigint: true;
    }): Promise<BigIntStats>;
    abstract statPromise(p: P, opts?: StatOptions): Promise<Stats | BigIntStats>;
    abstract statSync(p: P): Stats;
    abstract statSync(p: P, opts?: StatSyncOptions & {
        bigint?: false | undefined;
        throwIfNoEntry: false;
    }): Stats | undefined;
    abstract statSync(p: P, opts: StatSyncOptions & {
        bigint: true;
        throwIfNoEntry: false;
    }): BigIntStats | undefined;
    abstract statSync(p: P, opts?: StatSyncOptions & {
        bigint?: false | undefined;
    }): Stats;
    abstract statSync(p: P, opts: StatSyncOptions & {
        bigint: true;
    }): BigIntStats;
    abstract statSync(p: P, opts: StatSyncOptions & {
        bigint: boolean;
        throwIfNoEntry?: false | undefined;
    }): Stats | BigIntStats;
    abstract statSync(p: P, opts?: StatSyncOptions): Stats | BigIntStats | undefined;
    abstract fstatPromise(fd: number): Promise<Stats>;
    abstract fstatPromise(fd: number, opts: {
        bigint: true;
    }): Promise<BigIntStats>;
    abstract fstatPromise(fd: number, opts?: {
        bigint: boolean;
    }): Promise<BigIntStats | Stats>;
    abstract fstatSync(fd: number): Stats;
    abstract fstatSync(fd: number, opts: {
        bigint: true;
    }): BigIntStats;
    abstract fstatSync(fd: number, opts?: {
        bigint: boolean;
    }): BigIntStats | Stats;
    abstract lstatPromise(p: P): Promise<Stats>;
    abstract lstatPromise(p: P, opts: (StatOptions & {
        bigint?: false | undefined;
    }) | undefined): Promise<Stats>;
    abstract lstatPromise(p: P, opts: StatOptions & {
        bigint: true;
    }): Promise<BigIntStats>;
    abstract lstatPromise(p: P, opts?: StatOptions): Promise<Stats | BigIntStats>;
    abstract lstatSync(p: P): Stats;
    abstract lstatSync(p: P, opts?: StatSyncOptions & {
        bigint?: false | undefined;
        throwIfNoEntry: false;
    }): Stats | undefined;
    abstract lstatSync(p: P, opts: StatSyncOptions & {
        bigint: true;
        throwIfNoEntry: false;
    }): BigIntStats | undefined;
    abstract lstatSync(p: P, opts?: StatSyncOptions & {
        bigint?: false | undefined;
    }): Stats;
    abstract lstatSync(p: P, opts: StatSyncOptions & {
        bigint: true;
    }): BigIntStats;
    abstract lstatSync(p: P, opts: StatSyncOptions & {
        bigint: boolean;
        throwIfNoEntry?: false | undefined;
    }): Stats | BigIntStats;
    abstract lstatSync(p: P, opts?: StatSyncOptions): Stats | BigIntStats | undefined;
    abstract chmodPromise(p: P, mask: number): Promise<void>;
    abstract chmodSync(p: P, mask: number): void;
    abstract fchmodPromise(fd: number, mask: number): Promise<void>;
    abstract fchmodSync(fd: number, mask: number): void;
    abstract fchownPromise(fd: number, uid: number, gid: number): Promise<void>;
    abstract fchownSync(fd: number, uid: number, gid: number): void;
    abstract chownPromise(p: P, uid: number, gid: number): Promise<void>;
    abstract chownSync(p: P, uid: number, gid: number): void;
    abstract mkdirPromise(p: P, opts?: MkdirOptions): Promise<string | undefined>;
    abstract mkdirSync(p: P, opts?: MkdirOptions): string | undefined;
    abstract rmdirPromise(p: P, opts?: RmdirOptions): Promise<void>;
    abstract rmdirSync(p: P, opts?: RmdirOptions): void;
    abstract linkPromise(existingP: P, newP: P): Promise<void>;
    abstract linkSync(existingP: P, newP: P): void;
    abstract symlinkPromise(target: P, p: P, type?: SymlinkType): Promise<void>;
    abstract symlinkSync(target: P, p: P, type?: SymlinkType): void;
    abstract renamePromise(oldP: P, newP: P): Promise<void>;
    abstract renameSync(oldP: P, newP: P): void;
    abstract copyFilePromise(sourceP: P, destP: P, flags?: number): Promise<void>;
    abstract copyFileSync(sourceP: P, destP: P, flags?: number): void;
    abstract appendFilePromise(p: FSPath<P>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): Promise<void>;
    abstract appendFileSync(p: FSPath<P>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): void;
    abstract writeFilePromise(p: FSPath<P>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): Promise<void>;
    abstract writeFileSync(p: FSPath<P>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): void;
    abstract unlinkPromise(p: P): Promise<void>;
    abstract unlinkSync(p: P): void;
    abstract utimesPromise(p: P, atime: Date | string | number, mtime: Date | string | number): Promise<void>;
    abstract utimesSync(p: P, atime: Date | string | number, mtime: Date | string | number): void;
    lutimesPromise?(p: P, atime: Date | string | number, mtime: Date | string | number): Promise<void>;
    lutimesSync?(p: P, atime: Date | string | number, mtime: Date | string | number): void;
    abstract readFilePromise(p: FSPath<P>, encoding: 'utf8'): Promise<string>;
    abstract readFilePromise(p: FSPath<P>, encoding?: string): Promise<Buffer>;
    abstract readFileSync(p: FSPath<P>, encoding: 'utf8'): string;
    abstract readFileSync(p: FSPath<P>, encoding?: string): Buffer;
    abstract readlinkPromise(p: P): Promise<P>;
    abstract readlinkSync(p: P): P;
    abstract ftruncatePromise(fd: number, len?: number): Promise<void>;
    abstract ftruncateSync(fd: number, len?: number): void;
    abstract truncatePromise(p: P, len?: number): Promise<void>;
    abstract truncateSync(p: P, len?: number): void;
    abstract watch(p: P, cb?: WatchCallback): Watcher;
    abstract watch(p: P, opts: WatchOptions, cb?: WatchCallback): Watcher;
    abstract watchFile(p: P, cb: WatchFileCallback): StatWatcher;
    abstract watchFile(p: P, opts: WatchFileOptions, cb: WatchFileCallback): StatWatcher;
    abstract unwatchFile(p: P, cb?: WatchFileCallback): void;
    genTraversePromise(init: P, { stableSort }?: {
        stableSort?: boolean;
    }): AsyncGenerator<P, void, unknown>;
    removePromise(p: P, { recursive, maxRetries }?: {
        recursive?: boolean;
        maxRetries?: number;
    }): Promise<void>;
    removeSync(p: P, { recursive }?: {
        recursive?: boolean;
    }): void;
    mkdirpPromise(p: P, { chmod, utimes }?: {
        chmod?: number;
        utimes?: [Date | string | number, Date | string | number];
    }): Promise<string | undefined>;
    mkdirpSync(p: P, { chmod, utimes }?: {
        chmod?: number;
        utimes?: [Date | string | number, Date | string | number];
    }): string | undefined;
    copyPromise(destination: P, source: P, options?: {
        baseFs?: undefined;
        overwrite?: boolean;
        stableSort?: boolean;
        stableTime?: boolean;
        linkStrategy?: LinkStrategy;
    }): Promise<void>;
    copyPromise<P2 extends Path>(destination: P, source: P2, options: {
        baseFs: FakeFS<P2>;
        overwrite?: boolean;
        stableSort?: boolean;
        stableTime?: boolean;
        linkStrategy?: LinkStrategy;
    }): Promise<void>;
    /** @deprecated Prefer using `copyPromise` instead */
    copySync(destination: P, source: P, options?: {
        baseFs?: undefined;
        overwrite?: boolean;
    }): void;
    copySync<P2 extends Path>(destination: P, source: P2, options: {
        baseFs: FakeFS<P2>;
        overwrite?: boolean;
    }): void;
    changeFilePromise(p: P, content: Buffer): Promise<void>;
    changeFilePromise(p: P, content: string, opts?: ChangeFileOptions): Promise<void>;
    private changeFileBufferPromise;
    private changeFileTextPromise;
    changeFileSync(p: P, content: Buffer): void;
    changeFileSync(p: P, content: string, opts?: ChangeFileOptions): void;
    private changeFileBufferSync;
    private changeFileTextSync;
    movePromise(fromP: P, toP: P): Promise<void>;
    moveSync(fromP: P, toP: P): void;
    lockPromise<T>(affectedPath: P, callback: () => Promise<T>): Promise<T>;
    readJsonPromise(p: P): Promise<any>;
    readJsonSync(p: P): any;
    writeJsonPromise(p: P, data: any): Promise<void>;
    writeJsonSync(p: P, data: any): void;
    preserveTimePromise(p: P, cb: () => Promise<P | void>): Promise<void>;
    preserveTimeSync(p: P, cb: () => P | void): Promise<void>;
}
export declare abstract class BasePortableFakeFS extends FakeFS<PortablePath> {
    protected constructor();
}
export declare function normalizeLineEndings(originalContent: string, newContent: string): string;
