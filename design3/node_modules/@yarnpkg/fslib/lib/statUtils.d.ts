/// <reference types="node" />
/// <reference types="node" />
import { BigIntStats, Stats } from 'fs';
import { Filename } from './path';
export declare const DEFAULT_MODE: number;
export declare class DirEntry {
    name: Filename;
    mode: number;
    isBlockDevice(): boolean;
    isCharacterDevice(): boolean;
    isDirectory(): boolean;
    isFIFO(): boolean;
    isFile(): boolean;
    isSocket(): boolean;
    isSymbolicLink(): boolean;
}
export declare class StatEntry {
    uid: number;
    gid: number;
    size: number;
    blksize: number;
    atimeMs: number;
    mtimeMs: number;
    ctimeMs: number;
    birthtimeMs: number;
    atime: Date;
    mtime: Date;
    ctime: Date;
    birthtime: Date;
    dev: number;
    ino: number;
    mode: number;
    nlink: number;
    rdev: number;
    blocks: number;
    isBlockDevice(): boolean;
    isCharacterDevice(): boolean;
    isDirectory(): boolean;
    isFIFO(): boolean;
    isFile(): boolean;
    isSocket(): boolean;
    isSymbolicLink(): boolean;
}
export declare class BigIntStatsEntry {
    uid: bigint;
    gid: bigint;
    size: bigint;
    blksize: bigint;
    atimeMs: bigint;
    mtimeMs: bigint;
    ctimeMs: bigint;
    birthtimeMs: bigint;
    atimeNs: bigint;
    mtimeNs: bigint;
    ctimeNs: bigint;
    birthtimeNs: bigint;
    atime: Date;
    mtime: Date;
    ctime: Date;
    birthtime: Date;
    dev: bigint;
    ino: bigint;
    mode: bigint;
    nlink: bigint;
    rdev: bigint;
    blocks: bigint;
    isBlockDevice(): boolean;
    isCharacterDevice(): boolean;
    isDirectory(): boolean;
    isFIFO(): boolean;
    isFile(): boolean;
    isSocket(): boolean;
    isSymbolicLink(): boolean;
}
export declare function makeDefaultStats(): StatEntry;
export declare function makeEmptyStats(): Stats | BigIntStats;
/**
 * Mutates the provided stats object to zero it out then returns it for convenience
 */
export declare function clearStats(stats: Stats | BigIntStats): Stats | BigIntStats;
export declare function convertToBigIntStats(stats: Stats): BigIntStats;
export declare function areStatsEqual(a: Stats | BigIntStatsEntry, b: Stats | BigIntStatsEntry): boolean;
