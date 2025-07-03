/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
/// <reference types="node" />
import type { BigIntStats, ReadStream, StatOptions, Stats, WriteStream, WriteVResult } from 'fs';
import type { CreateReadStreamOptions, CreateWriteStreamOptions, FakeFS } from '../FakeFS';
import type { Path } from '../path';
interface ObjectEncodingOptions {
    encoding?: BufferEncoding | null | undefined;
}
interface FlagAndOpenMode {
    mode?: Mode | undefined;
    flag?: OpenMode | undefined;
}
type OpenMode = number | string;
type Mode = number | string;
interface FileReadResult<T extends ArrayBufferView> {
    bytesRead: number;
    buffer: T;
}
interface FileReadOptions<T extends ArrayBufferView = Buffer> {
    buffer?: T;
    offset?: number | null;
    length?: number | null;
    position?: number | null;
}
interface ReadVResult {
    bytesRead: number;
    buffers: Array<NodeJS.ArrayBufferView>;
}
interface AbortSignal {
    readonly aborted: boolean;
}
interface Abortable {
    signal?: AbortSignal | undefined;
}
type WriteArgsBuffer<TBuffer extends Uint8Array> = [
    buffer: TBuffer,
    offset?: number | null,
    length?: number | null,
    position?: number | null
];
type WriteArgsString = [
    data: string,
    position?: number | null,
    encoding?: BufferEncoding | null
];
declare const kBaseFs: unique symbol;
declare const kFd: unique symbol;
declare const kClosePromise: unique symbol;
declare const kCloseResolve: unique symbol;
declare const kCloseReject: unique symbol;
declare const kRefs: unique symbol;
declare const kRef: unique symbol;
declare const kUnref: unique symbol;
export declare class FileHandle<P extends Path> {
    [kBaseFs]: FakeFS<P>;
    [kFd]: number;
    [kRefs]: number;
    [kClosePromise]: Promise<void> | undefined;
    [kCloseResolve]: (() => void) | undefined;
    [kCloseReject]: (() => void) | undefined;
    constructor(fd: number, baseFs: FakeFS<P>);
    get fd(): number;
    appendFile(data: string | Uint8Array, options?: (ObjectEncodingOptions & FlagAndOpenMode) | BufferEncoding | null): Promise<void>;
    chown(uid: number, gid: number): Promise<void>;
    chmod(mode: number): Promise<void>;
    createReadStream(options?: CreateReadStreamOptions): ReadStream;
    createWriteStream(options?: CreateWriteStreamOptions): WriteStream;
    datasync(): Promise<void>;
    sync(): Promise<void>;
    read(options?: FileReadOptions<Buffer>): Promise<FileReadResult<Buffer>>;
    read(buffer: Buffer, offset?: number | null, length?: number | null, position?: number | null): Promise<FileReadResult<Buffer>>;
    readFile(options?: {
        encoding?: null | undefined;
        flag?: OpenMode | undefined;
    } | null): Promise<Buffer>;
    readFile(options: {
        encoding: BufferEncoding;
        flag?: OpenMode | undefined;
    } | BufferEncoding): Promise<string>;
    readLines(options?: CreateReadStreamOptions): import("readline").Interface;
    stat(opts?: StatOptions & {
        bigint?: false | undefined;
    }): Promise<Stats>;
    stat(opts: StatOptions & {
        bigint: true;
    }): Promise<BigIntStats>;
    truncate(len?: number): Promise<void>;
    utimes(atime: string | number | Date, mtime: string | number | Date): Promise<void>;
    writeFile(data: string | Uint8Array, options?: (ObjectEncodingOptions & FlagAndOpenMode & Abortable) | BufferEncoding | null): Promise<void>;
    write(...args: WriteArgsString): Promise<{
        bytesWritten: number;
        buffer: string;
    }>;
    write<TBuffer extends Uint8Array>(...args: WriteArgsBuffer<TBuffer>): Promise<{
        bytesWritten: number;
        buffer: TBuffer;
    }>;
    writev(buffers: Array<NodeJS.ArrayBufferView>, position?: number): Promise<WriteVResult>;
    readv(buffers: ReadonlyArray<NodeJS.ArrayBufferView>, position?: number): Promise<ReadVResult>;
    close(): Promise<void>;
    [kRef](caller: Function): void;
    [kUnref](): void;
}
export {};
