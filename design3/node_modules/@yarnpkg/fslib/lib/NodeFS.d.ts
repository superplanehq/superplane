/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
import fs, { BigIntStats, Stats } from 'fs';
import { CreateReadStreamOptions, CreateWriteStreamOptions, Dir, StatWatcher, WatchFileCallback, WatchFileOptions, OpendirOptions } from './FakeFS';
import { Dirent, SymlinkType, StatSyncOptions, StatOptions } from './FakeFS';
import { BasePortableFakeFS, WriteFileOptions } from './FakeFS';
import { MkdirOptions, RmdirOptions, WatchOptions, WatchCallback, Watcher } from './FakeFS';
import { FSPath, PortablePath, Filename } from './path';
export declare class NodeFS extends BasePortableFakeFS {
    private readonly realFs;
    constructor(realFs?: typeof fs);
    getExtractHint(): boolean;
    getRealPath(): PortablePath;
    resolve(p: PortablePath): PortablePath;
    openPromise(p: PortablePath, flags: string, mode?: number): Promise<number>;
    openSync(p: PortablePath, flags: string, mode?: number): number;
    opendirPromise(p: PortablePath, opts?: OpendirOptions): Promise<Dir<PortablePath>>;
    opendirSync(p: PortablePath, opts?: OpendirOptions): Dir<PortablePath>;
    readPromise(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number | null): Promise<number>;
    readSync(fd: number, buffer: Buffer, offset: number, length: number, position: number): number;
    writePromise(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number): Promise<number>;
    writePromise(fd: number, buffer: string, position?: number): Promise<number>;
    writeSync(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number): number;
    writeSync(fd: number, buffer: string, position?: number): number;
    closePromise(fd: number): Promise<void>;
    closeSync(fd: number): void;
    createReadStream(p: PortablePath | null, opts?: CreateReadStreamOptions): fs.ReadStream;
    createWriteStream(p: PortablePath | null, opts?: CreateWriteStreamOptions): fs.WriteStream;
    realpathPromise(p: PortablePath): Promise<PortablePath>;
    realpathSync(p: PortablePath): PortablePath;
    existsPromise(p: PortablePath): Promise<boolean>;
    accessSync(p: PortablePath, mode?: number): void;
    accessPromise(p: PortablePath, mode?: number): Promise<void>;
    existsSync(p: PortablePath): boolean;
    statPromise(p: PortablePath): Promise<Stats>;
    statPromise(p: PortablePath, opts: (StatOptions & {
        bigint?: false | undefined;
    }) | undefined): Promise<Stats>;
    statPromise(p: PortablePath, opts: StatOptions & {
        bigint: true;
    }): Promise<BigIntStats>;
    statSync(p: PortablePath): Stats;
    statSync(p: PortablePath, opts?: StatSyncOptions & {
        bigint?: false | undefined;
        throwIfNoEntry: false;
    }): Stats | undefined;
    statSync(p: PortablePath, opts: StatSyncOptions & {
        bigint: true;
        throwIfNoEntry: false;
    }): BigIntStats | undefined;
    statSync(p: PortablePath, opts?: StatSyncOptions & {
        bigint?: false | undefined;
    }): Stats;
    statSync(p: PortablePath, opts: StatSyncOptions & {
        bigint: true;
    }): BigIntStats;
    statSync(p: PortablePath, opts: StatSyncOptions & {
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
    lstatPromise(p: PortablePath): Promise<Stats>;
    lstatPromise(p: PortablePath, opts: (StatOptions & {
        bigint?: false | undefined;
    }) | undefined): Promise<Stats>;
    lstatPromise(p: PortablePath, opts: StatOptions & {
        bigint: true;
    }): Promise<BigIntStats>;
    lstatSync(p: PortablePath): Stats;
    lstatSync(p: PortablePath, opts?: StatSyncOptions & {
        bigint?: false | undefined;
        throwIfNoEntry: false;
    }): Stats | undefined;
    lstatSync(p: PortablePath, opts: StatSyncOptions & {
        bigint: true;
        throwIfNoEntry: false;
    }): BigIntStats | undefined;
    lstatSync(p: PortablePath, opts?: StatSyncOptions & {
        bigint?: false | undefined;
    }): Stats;
    lstatSync(p: PortablePath, opts: StatSyncOptions & {
        bigint: true;
    }): BigIntStats;
    lstatSync(p: PortablePath, opts: StatSyncOptions & {
        bigint: boolean;
        throwIfNoEntry?: false | undefined;
    }): Stats | BigIntStats;
    fchmodPromise(fd: number, mask: number): Promise<void>;
    fchmodSync(fd: number, mask: number): void;
    chmodPromise(p: PortablePath, mask: number): Promise<void>;
    chmodSync(p: PortablePath, mask: number): void;
    fchownPromise(fd: number, uid: number, gid: number): Promise<void>;
    fchownSync(fd: number, uid: number, gid: number): void;
    chownPromise(p: PortablePath, uid: number, gid: number): Promise<void>;
    chownSync(p: PortablePath, uid: number, gid: number): void;
    renamePromise(oldP: PortablePath, newP: PortablePath): Promise<void>;
    renameSync(oldP: PortablePath, newP: PortablePath): void;
    copyFilePromise(sourceP: PortablePath, destP: PortablePath, flags?: number): Promise<void>;
    copyFileSync(sourceP: PortablePath, destP: PortablePath, flags?: number): void;
    appendFilePromise(p: FSPath<PortablePath>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): Promise<void>;
    appendFileSync(p: PortablePath, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): void;
    writeFilePromise(p: FSPath<PortablePath>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): Promise<void>;
    writeFileSync(p: PortablePath, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): void;
    unlinkPromise(p: PortablePath): Promise<void>;
    unlinkSync(p: PortablePath): void;
    utimesPromise(p: PortablePath, atime: Date | string | number, mtime: Date | string | number): Promise<void>;
    utimesSync(p: PortablePath, atime: Date | string | number, mtime: Date | string | number): void;
    private lutimesPromiseImpl;
    private lutimesSyncImpl;
    mkdirPromise(p: PortablePath, opts?: MkdirOptions): Promise<string | undefined>;
    mkdirSync(p: PortablePath, opts?: MkdirOptions): string | undefined;
    rmdirPromise(p: PortablePath, opts?: RmdirOptions): Promise<void>;
    rmdirSync(p: PortablePath, opts?: RmdirOptions): void;
    linkPromise(existingP: PortablePath, newP: PortablePath): Promise<void>;
    linkSync(existingP: PortablePath, newP: PortablePath): void;
    symlinkPromise(target: PortablePath, p: PortablePath, type?: SymlinkType): Promise<void>;
    symlinkSync(target: PortablePath, p: PortablePath, type?: SymlinkType): void;
    readFilePromise(p: FSPath<PortablePath>, encoding: 'utf8'): Promise<string>;
    readFilePromise(p: FSPath<PortablePath>, encoding?: string): Promise<Buffer>;
    readFileSync(p: FSPath<PortablePath>, encoding: 'utf8'): string;
    readFileSync(p: FSPath<PortablePath>, encoding?: string): Buffer;
    readdirPromise(p: PortablePath): Promise<Array<Filename>>;
    readdirPromise(p: PortablePath, opts: {
        withFileTypes: false;
    } | null): Promise<Array<Filename>>;
    readdirPromise(p: PortablePath, opts: {
        withFileTypes: true;
    }): Promise<Array<Dirent>>;
    readdirPromise(p: PortablePath, opts: {
        withFileTypes: boolean;
    }): Promise<Array<Filename> | Array<Dirent>>;
    readdirSync(p: PortablePath): Array<Filename>;
    readdirSync(p: PortablePath, opts: {
        withFileTypes: false;
    } | null): Array<Filename>;
    readdirSync(p: PortablePath, opts: {
        withFileTypes: true;
    }): Array<Dirent>;
    readdirSync(p: PortablePath, opts: {
        withFileTypes: boolean;
    }): Array<Filename> | Array<Dirent>;
    readlinkPromise(p: PortablePath): Promise<PortablePath>;
    readlinkSync(p: PortablePath): PortablePath;
    truncatePromise(p: PortablePath, len?: number): Promise<void>;
    truncateSync(p: PortablePath, len?: number): void;
    ftruncatePromise(fd: number, len?: number): Promise<void>;
    ftruncateSync(fd: number, len?: number): void;
    watch(p: PortablePath, cb?: WatchCallback): Watcher;
    watch(p: PortablePath, opts: WatchOptions, cb?: WatchCallback): Watcher;
    watchFile(p: PortablePath, cb: WatchFileCallback): StatWatcher;
    watchFile(p: PortablePath, opts: WatchFileOptions, cb: WatchFileCallback): StatWatcher;
    unwatchFile(p: PortablePath, cb?: WatchFileCallback): void;
    private makeCallback;
}
