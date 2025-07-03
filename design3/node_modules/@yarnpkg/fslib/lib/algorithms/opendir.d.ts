/// <reference types="node" />
/// <reference types="node" />
import { NoParamCallback } from 'fs';
import { Dir, Dirent, FakeFS } from '../FakeFS';
import { Filename, Path } from '../path';
export type CustomDirOptions = {
    onClose?: () => void;
};
export declare class CustomDir<P extends Path> implements Dir<P> {
    readonly path: P;
    private readonly nextDirent;
    private readonly opts;
    constructor(path: P, nextDirent: () => Dirent | null, opts?: CustomDirOptions);
    closed: boolean;
    throwIfClosed(): void;
    [Symbol.asyncIterator](): AsyncGenerator<Dirent, void, unknown>;
    read(): Promise<Dirent>;
    read(cb: (err: NodeJS.ErrnoException | null, dirent: Dirent | null) => void): void;
    readSync(): Dirent | null;
    close(): Promise<void>;
    close(cb: NoParamCallback): void;
    closeSync(): void;
}
export declare function opendir<P extends Path>(fakeFs: FakeFS<P>, path: P, entries: Array<Filename>, opts?: CustomDirOptions): CustomDir<P>;
