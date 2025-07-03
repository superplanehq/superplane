/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
import { Stats, BigIntStats } from 'fs';
import { CreateReadStreamOptions, CreateWriteStreamOptions, FakeFS, ExtractHintOptions, WatchFileCallback, WatchFileOptions, StatWatcher, Dir, OpendirOptions } from './FakeFS';
import { Dirent, SymlinkType, StatSyncOptions, StatOptions } from './FakeFS';
import { MkdirOptions, RmdirOptions, WriteFileOptions, WatchCallback, WatchOptions, Watcher } from './FakeFS';
import { FSPath, Filename, Path } from './path';
export declare abstract class ProxiedFS<P extends Path, IP extends Path> extends FakeFS<P> {
    protected abstract readonly baseFs: FakeFS<IP>;
    /**
     * Convert a path from the user format into what should be fed into the internal FS.
     */
    protected abstract mapToBase(path: P): IP;
    /**
     * Convert a path from the format supported by the base FS into the user one.
     */
    protected abstract mapFromBase(path: IP): P;
    getExtractHint(hints: ExtractHintOptions): boolean;
    resolve(path: P): P;
    getRealPath(): P;
    openPromise(p: P, flags: string, mode?: number): Promise<number>;
    openSync(p: P, flags: string, mode?: number): number;
    opendirPromise(p: P, opts?: OpendirOptions): Promise<Dir<P>>;
    opendirSync(p: P, opts?: OpendirOptions): Dir<P>;
    readPromise(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number | null): Promise<number>;
    readSync(fd: number, buffer: Buffer, offset: number, length: number, position: number): number;
    writePromise(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number): Promise<number>;
    writePromise(fd: number, buffer: string, position?: number): Promise<number>;
    writeSync(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number): number;
    writeSync(fd: number, buffer: string, position?: number): number;
    closePromise(fd: number): Promise<void>;
    closeSync(fd: number): void;
    createReadStream(p: P | null, opts?: CreateReadStreamOptions): import("fs").ReadStream;
    createWriteStream(p: P | null, opts?: CreateWriteStreamOptions): import("fs").WriteStream;
    realpathPromise(p: P): Promise<P>;
    realpathSync(p: P): P;
    existsPromise(p: P): Promise<boolean>;
    existsSync(p: P): boolean;
    accessSync(p: P, mode?: number): void;
    accessPromise(p: P, mode?: number): Promise<void>;
    statPromise(p: P): Promise<Stats>;
    statPromise(p: P, opts: (StatOptions & {
        bigint?: false | undefined;
    }) | undefined): Promise<Stats>;
    statPromise(p: P, opts: StatOptions & {
        bigint: true;
    }): Promise<BigIntStats>;
    statSync(p: P): Stats;
    statSync(p: P, opts?: StatSyncOptions & {
        bigint?: false | undefined;
        throwIfNoEntry: false;
    }): Stats | undefined;
    statSync(p: P, opts: StatSyncOptions & {
        bigint: true;
        throwIfNoEntry: false;
    }): BigIntStats | undefined;
    statSync(p: P, opts?: StatSyncOptions & {
        bigint?: false | undefined;
    }): Stats;
    statSync(p: P, opts: StatSyncOptions & {
        bigint: true;
    }): BigIntStats;
    statSync(p: P, opts: StatSyncOptions & {
        bigint: boolean;
        throwIfNoEntry?: false | undefined;
    }): Stats | BigIntStats;
    fstatPromise(fd: number): Promise<Stats>;
    fstatPromise(fd: number, opts: {
        bigint: true;
    }): Promise<BigIntStats>;
    fstatPromise(fd: number, opts?: {
        bigint: boolean;
    }): Promise<BigIntStats | Stats>;
    fstatSync(fd: number): Stats;
    fstatSync(fd: number, opts: {
        bigint: true;
    }): BigIntStats;
    fstatSync(fd: number, opts?: {
        bigint: boolean;
    }): BigIntStats | Stats;
    lstatPromise(p: P): Promise<Stats>;
    lstatPromise(p: P, opts: (StatOptions & {
        bigint?: false | undefined;
    }) | undefined): Promise<Stats>;
    lstatPromise(p: P, opts: StatOptions & {
        bigint: true;
    }): Promise<BigIntStats>;
    lstatSync(p: P): Stats;
    lstatSync(p: P, opts?: StatSyncOptions & {
        bigint?: false | undefined;
        throwIfNoEntry: false;
    }): Stats | undefined;
    lstatSync(p: P, opts: StatSyncOptions & {
        bigint: true;
        throwIfNoEntry: false;
    }): BigIntStats | undefined;
    lstatSync(p: P, opts?: StatSyncOptions & {
        bigint?: false | undefined;
    }): Stats;
    lstatSync(p: P, opts: StatSyncOptions & {
        bigint: true;
    }): BigIntStats;
    lstatSync(p: P, opts: StatSyncOptions & {
        bigint: boolean;
        throwIfNoEntry?: false | undefined;
    }): Stats | BigIntStats;
    fchmodPromise(fd: number, mask: number): Promise<void>;
    fchmodSync(fd: number, mask: number): void;
    chmodPromise(p: P, mask: number): Promise<void>;
    chmodSync(p: P, mask: number): void;
    fchownPromise(fd: number, uid: number, gid: number): Promise<void>;
    fchownSync(fd: number, uid: number, gid: number): void;
    chownPromise(p: P, uid: number, gid: number): Promise<void>;
    chownSync(p: P, uid: number, gid: number): void;
    renamePromise(oldP: P, newP: P): Promise<void>;
    renameSync(oldP: P, newP: P): void;
    copyFilePromise(sourceP: P, destP: P, flags?: number): Promise<void>;
    copyFileSync(sourceP: P, destP: P, flags?: number): void;
    appendFilePromise(p: FSPath<P>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): Promise<void>;
    appendFileSync(p: FSPath<P>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): void;
    writeFilePromise(p: FSPath<P>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): Promise<void>;
    writeFileSync(p: FSPath<P>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): void;
    unlinkPromise(p: P): Promise<void>;
    unlinkSync(p: P): void;
    utimesPromise(p: P, atime: Date | string | number, mtime: Date | string | number): Promise<void>;
    utimesSync(p: P, atime: Date | string | number, mtime: Date | string | number): void;
    mkdirPromise(p: P, opts?: MkdirOptions): Promise<string | undefined>;
    mkdirSync(p: P, opts?: MkdirOptions): string | undefined;
    rmdirPromise(p: P, opts?: RmdirOptions): Promise<void>;
    rmdirSync(p: P, opts?: RmdirOptions): void;
    linkPromise(existingP: P, newP: P): Promise<void>;
    linkSync(existingP: P, newP: P): void;
    symlinkPromise(target: P, p: P, type?: SymlinkType): Promise<void>;
    symlinkSync(target: P, p: P, type?: SymlinkType): void;
    readFilePromise(p: FSPath<P>, encoding: 'utf8'): Promise<string>;
    readFilePromise(p: FSPath<P>, encoding?: string): Promise<Buffer>;
    readFileSync(p: FSPath<P>, encoding: 'utf8'): string;
    readFileSync(p: FSPath<P>, encoding?: string): Buffer;
    readdirPromise(p: P): Promise<Array<Filename>>;
    readdirPromise(p: P, opts: {
        withFileTypes: false;
    } | null): Promise<Array<Filename>>;
    readdirPromise(p: P, opts: {
        withFileTypes: true;
    }): Promise<Array<Dirent>>;
    readdirPromise(p: P, opts: {
        withFileTypes: boolean;
    }): Promise<Array<Filename> | Array<Dirent>>;
    readdirSync(p: P): Array<Filename>;
    readdirSync(p: P, opts: {
        withFileTypes: false;
    } | null): Array<Filename>;
    readdirSync(p: P, opts: {
        withFileTypes: true;
    }): Array<Dirent>;
    readdirSync(p: P, opts: {
        withFileTypes: boolean;
    }): Array<Filename> | Array<Dirent>;
    readlinkPromise(p: P): Promise<P>;
    readlinkSync(p: P): P;
    truncatePromise(p: P, len?: number): Promise<void>;
    truncateSync(p: P, len?: number): void;
    ftruncatePromise(fd: number, len?: number): Promise<void>;
    ftruncateSync(fd: number, len?: number): void;
    watch(p: P, cb?: WatchCallback): Watcher;
    watch(p: P, opts: WatchOptions, cb?: WatchCallback): Watcher;
    watchFile(p: P, cb: WatchFileCallback): StatWatcher;
    watchFile(p: P, opts: WatchFileOptions, cb: WatchFileCallback): StatWatcher;
    unwatchFile(p: P, cb?: WatchFileCallback): void;
    private fsMapToBase;
}
