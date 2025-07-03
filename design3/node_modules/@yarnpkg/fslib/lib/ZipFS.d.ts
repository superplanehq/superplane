/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
import { Libzip } from '@yarnpkg/libzip';
import { ReadStream, WriteStream } from 'fs';
import { WatchOptions, WatchCallback, Watcher, Dir, Stats, BigIntStats, StatSyncOptions, StatOptions } from './FakeFS';
import { FakeFS, MkdirOptions, RmdirOptions, WriteFileOptions, OpendirOptions } from './FakeFS';
import { CreateReadStreamOptions, CreateWriteStreamOptions, BasePortableFakeFS, ExtractHintOptions, WatchFileCallback, WatchFileOptions, StatWatcher } from './FakeFS';
import * as errors from './errors';
import { FSPath, PortablePath, Filename } from './path';
import * as statUtils from './statUtils';
export type ZipCompression = `mixed` | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9;
export declare const DEFAULT_COMPRESSION_LEVEL: ZipCompression;
export type ZipBufferOptions = {
    libzip: Libzip;
    readOnly?: boolean;
    stats?: Stats;
    level?: ZipCompression;
};
export type ZipPathOptions = ZipBufferOptions & {
    baseFs?: FakeFS<PortablePath>;
    create?: boolean;
};
export declare function makeEmptyArchive(): Buffer;
export declare class ZipFS extends BasePortableFakeFS {
    private readonly libzip;
    private readonly baseFs;
    private readonly path;
    private readonly stats;
    private readonly zip;
    private readonly lzSource;
    private readonly level;
    private readonly listings;
    private readonly entries;
    /**
     * A cache of indices mapped to file sources.
     * Populated by `setFileSource` calls.
     * Required for supporting read after write.
     */
    private readonly fileSources;
    private symlinkCount;
    private readonly fds;
    private nextFd;
    private ready;
    private readOnly;
    constructor(p: PortablePath, opts: ZipPathOptions);
    /**
     * Create a ZipFS in memory
     * @param data If null; an empty zip file will be created
     */
    constructor(data: Buffer | null, opts: ZipBufferOptions);
    makeLibzipError(error: number): errors.LibzipError;
    getExtractHint(hints: ExtractHintOptions): boolean;
    getAllFiles(): PortablePath[];
    getRealPath(): PortablePath;
    getBufferAndClose(): Buffer;
    private prepareClose;
    saveAndClose(): void;
    discardAndClose(): void;
    resolve(p: PortablePath): PortablePath;
    openPromise(p: PortablePath, flags: string, mode?: number): Promise<number>;
    openSync(p: PortablePath, flags: string, mode?: number): number;
    hasOpenFileHandles(): boolean;
    opendirPromise(p: PortablePath, opts?: OpendirOptions): Promise<Dir<PortablePath>>;
    opendirSync(p: PortablePath, opts?: OpendirOptions): Dir<PortablePath>;
    readPromise(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number | null): Promise<number>;
    readSync(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number | null): number;
    writePromise(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number): Promise<number>;
    writePromise(fd: number, buffer: string, position?: number): Promise<number>;
    writeSync(fd: number, buffer: Buffer, offset?: number, length?: number, position?: number): number;
    writeSync(fd: number, buffer: string, position?: number): number;
    closePromise(fd: number): Promise<void>;
    closeSync(fd: number): void;
    createReadStream(p: PortablePath | null, { encoding }?: CreateReadStreamOptions): ReadStream;
    createWriteStream(p: PortablePath | null, { encoding }?: CreateWriteStreamOptions): WriteStream;
    realpathPromise(p: PortablePath): Promise<PortablePath>;
    realpathSync(p: PortablePath): PortablePath;
    existsPromise(p: PortablePath): Promise<boolean>;
    existsSync(p: PortablePath): boolean;
    accessPromise(p: PortablePath, mode?: number): Promise<void>;
    accessSync(p: PortablePath, mode?: number): void;
    statPromise(p: PortablePath): Promise<Stats>;
    statPromise(p: PortablePath, opts: (StatOptions & {
        bigint?: false | undefined;
    }) | undefined): Promise<Stats>;
    statPromise(p: PortablePath, opts: StatOptions & {
        bigint: true;
    }): Promise<BigIntStats>;
    statPromise(p: PortablePath, opts?: StatOptions): Promise<Stats | BigIntStats>;
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
    statSync(p: PortablePath, opts?: StatSyncOptions): Stats | BigIntStats | undefined;
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
    lstatPromise(p: PortablePath, opts?: StatOptions): Promise<Stats | BigIntStats>;
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
    lstatSync(p: PortablePath, opts?: StatSyncOptions): Stats | BigIntStats | undefined;
    private statImpl;
    private getUnixMode;
    private registerListing;
    private registerEntry;
    private unregisterListing;
    private unregisterEntry;
    private deleteEntry;
    private resolveFilename;
    private allocateBuffer;
    private allocateUnattachedSource;
    private allocateSource;
    private setFileSource;
    private isSymbolicLink;
    private getFileSource;
    fchmodPromise(fd: number, mask: number): Promise<void>;
    fchmodSync(fd: number, mask: number): void;
    chmodPromise(p: PortablePath, mask: number): Promise<void>;
    chmodSync(p: PortablePath, mask: number): void;
    fchownPromise(fd: number, uid: number, gid: number): Promise<void>;
    fchownSync(fd: number, uid: number, gid: number): void;
    chownPromise(p: PortablePath, uid: number, gid: number): Promise<void>;
    chownSync(p: PortablePath, uid: number, gid: number): void;
    renamePromise(oldP: PortablePath, newP: PortablePath): Promise<never>;
    renameSync(oldP: PortablePath, newP: PortablePath): never;
    copyFilePromise(sourceP: PortablePath, destP: PortablePath, flags?: number): Promise<void>;
    copyFileSync(sourceP: PortablePath, destP: PortablePath, flags?: number): void;
    private prepareCopyFile;
    appendFilePromise(p: FSPath<PortablePath>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): Promise<void>;
    appendFileSync(p: FSPath<PortablePath>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): void;
    private fdToPath;
    writeFilePromise(p: FSPath<PortablePath>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): Promise<void>;
    writeFileSync(p: FSPath<PortablePath>, content: string | Buffer | ArrayBuffer | DataView, opts?: WriteFileOptions): void;
    private prepareWriteFile;
    unlinkPromise(p: PortablePath): Promise<void>;
    unlinkSync(p: PortablePath): void;
    utimesPromise(p: PortablePath, atime: Date | string | number, mtime: Date | string | number): Promise<void>;
    utimesSync(p: PortablePath, atime: Date | string | number, mtime: Date | string | number): void;
    lutimesPromise(p: PortablePath, atime: Date | string | number, mtime: Date | string | number): Promise<void>;
    lutimesSync(p: PortablePath, atime: Date | string | number, mtime: Date | string | number): void;
    private utimesImpl;
    mkdirPromise(p: PortablePath, opts?: MkdirOptions): Promise<string | undefined>;
    mkdirSync(p: PortablePath, { mode, recursive }?: MkdirOptions): string | undefined;
    rmdirPromise(p: PortablePath, opts?: RmdirOptions): Promise<void>;
    rmdirSync(p: PortablePath, { recursive }?: RmdirOptions): void;
    private hydrateDirectory;
    linkPromise(existingP: PortablePath, newP: PortablePath): Promise<void>;
    linkSync(existingP: PortablePath, newP: PortablePath): void;
    symlinkPromise(target: PortablePath, p: PortablePath): Promise<void>;
    symlinkSync(target: PortablePath, p: PortablePath): void;
    readFilePromise(p: FSPath<PortablePath>, encoding: 'utf8'): Promise<string>;
    readFilePromise(p: FSPath<PortablePath>, encoding?: string): Promise<Buffer>;
    readFileSync(p: FSPath<PortablePath>, encoding: 'utf8'): string;
    readFileSync(p: FSPath<PortablePath>, encoding?: string): Buffer;
    private readFileBuffer;
    readdirPromise(p: PortablePath): Promise<Array<Filename>>;
    readdirPromise(p: PortablePath, opts: {
        withFileTypes: false;
    } | null): Promise<Array<Filename>>;
    readdirPromise(p: PortablePath, opts: {
        withFileTypes: true;
    }): Promise<Array<statUtils.DirEntry>>;
    readdirPromise(p: PortablePath, opts: {
        withFileTypes: boolean;
    }): Promise<Array<Filename> | Array<statUtils.DirEntry>>;
    readdirSync(p: PortablePath): Array<Filename>;
    readdirSync(p: PortablePath, opts: {
        withFileTypes: false;
    } | null): Array<Filename>;
    readdirSync(p: PortablePath, opts: {
        withFileTypes: true;
    }): Array<statUtils.DirEntry>;
    readdirSync(p: PortablePath, opts: {
        withFileTypes: boolean;
    }): Array<Filename> | Array<statUtils.DirEntry>;
    readlinkPromise(p: PortablePath): Promise<PortablePath>;
    readlinkSync(p: PortablePath): PortablePath;
    private prepareReadlink;
    truncatePromise(p: PortablePath, len?: number): Promise<void>;
    truncateSync(p: PortablePath, len?: number): void;
    ftruncatePromise(fd: number, len?: number): Promise<void>;
    ftruncateSync(fd: number, len?: number): void;
    watch(p: PortablePath, cb?: WatchCallback): Watcher;
    watch(p: PortablePath, opts: WatchOptions, cb?: WatchCallback): Watcher;
    watchFile(p: PortablePath, cb: WatchFileCallback): StatWatcher;
    watchFile(p: PortablePath, opts: WatchFileOptions, cb: WatchFileCallback): StatWatcher;
    unwatchFile(p: PortablePath, cb?: WatchFileCallback): void;
}
